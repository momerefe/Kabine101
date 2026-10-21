// Package auth owns server-side authentication and canonical account authority.
package auth

import (
	"context"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/momerefe/Kabine101/internal/config"
	"github.com/momerefe/Kabine101/internal/database"
)

type Mailer interface {
	Send(context.Context, string, string, string) error
}

type Service struct {
	pool          *pgxpool.Pool
	q             *database.Queries
	cfg           config.Config
	mailer        Mailer
	limiter       limiter
	passwordSlots chan struct{}
	dummyHash     string
}

type Account struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	Username          string `json:"username"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	EmailVerified     bool   `json:"email_verified"`
	EligibilityStatus string `json:"eligibility_status"`
}

func publicAccount(a database.Account) Account {
	return Account{ID: a.ID.String(), Name: a.Name, Email: a.Email.String, Username: a.Username.String, Role: a.Role, Status: a.Status, EmailVerified: a.EmailVerifiedAt.Valid, EligibilityStatus: a.EligibilityStatus}
}

func New(pool *pgxpool.Pool, cfg config.Config, mailer Mailer) (*Service, error) {
	if pool == nil || mailer == nil || !config.ValidDomain(cfg.SchoolDomain) || cfg.AppOrigin == "" || cfg.EligibilityPolicy != "manual" {
		return nil, errors.New("invalid authentication configuration")
	}
	dummy, err := HashPassword("random unused authentication placeholder")
	if err != nil {
		return nil, err
	}
	return &Service{pool: pool, q: database.New(pool), cfg: cfg, mailer: mailer, limiter: limiter{entries: make(map[string]rateEntry)}, passwordSlots: make(chan struct{}, 2), dummyHash: dummy}, nil
}

func (s *Service) email(raw string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) > 254 || strings.Count(email, "@") != 1 {
		return "", false
	}
	a, err := mail.ParseAddress(email)
	if err != nil || a.Address != email || strings.ContainsAny(email, "\r\n") {
		return "", false
	}
	p := strings.Split(email, "@")
	return email, len(p[0]) > 0 && p[1] == s.cfg.SchoolDomain
}

func (s *Service) eligible(a database.Account) bool {
	if a.Status != "active" || a.EligibilityStatus != "approved" {
		return false
	}
	if a.Username.Valid {
		return validUsername(a.Username.String) && !a.Email.Valid && !a.EmailVerifiedAt.Valid && a.Role == "admin"
	}
	_, school := s.email(a.Email.String)
	return a.Email.Valid && school && a.EmailVerifiedAt.Valid && (a.Role == "student" || a.Role == "admin")
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{2,31}$`)

func validUsername(username string) bool { return usernamePattern.MatchString(username) }

// Identity namespaces keep per-account rate limits distinct across login types.
func accountIdentity(a database.Account) string {
	if a.Username.Valid {
		return "username:" + strings.ToLower(a.Username.String)
	}
	return "email:" + a.Email.String
}

func (s *Service) passwordWork(ctx context.Context, work func() error) error {
	select {
	case s.passwordSlots <- struct{}{}:
		defer func() { <-s.passwordSlots }()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errBusy
	}
	return work()
}

var errBusy = errors.New("authentication busy")
var errInvalidToken = errors.New("invalid or expired token")
var errConflict = errors.New("account already exists")
var errDenied = errors.New("authorization denied")
var errLastAdmin = errors.New("last active administrator")
var errMailUnavailable = errors.New("email delivery temporarily unavailable")

func audit(ctx context.Context, q *database.Queries, actor pgtype.UUID, action string, target pgtype.UUID) error {
	return q.InsertAuthAudit(ctx, database.InsertAuthAuditParams{ActorID: actor, Action: action, EntityID: target})
}

func stamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func (s *Service) createAccount(ctx context.Context, name, email, password, role string) (database.Account, error) {
	var empty database.Account
	name = strings.TrimSpace(name)
	if !validName(name) {
		return empty, errors.New("invalid name")
	}
	var ok bool
	email, ok = s.email(email)
	if !ok {
		return empty, errors.New("school email required")
	}
	var hash string
	err := s.passwordWork(ctx, func() error { var e error; hash, e = HashPassword(password); return e })
	if err != nil {
		return empty, err
	}
	status, eligibility := "pending", "pending"
	if role == "admin" {
		status, eligibility = "active", "approved"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return empty, err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	a, err := q.CreateAccount(ctx, database.CreateAccountParams{Name: name, Email: pgtype.Text{String: email, Valid: true}, PasswordHash: hash, Role: role, Status: status, EligibilityStatus: eligibility})
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return empty, errConflict
		}
		return empty, err
	}
	action := "account.registered"
	actor := pgtype.UUID{}
	if role == "admin" {
		action = "account.admin_bootstrapped"
	}
	if err = audit(ctx, q, actor, action, a.ID); err != nil {
		return empty, err
	}
	if err = tx.Commit(ctx); err != nil {
		return empty, err
	}
	return a, nil
}

func validName(name string) bool {
	name = strings.TrimSpace(name)
	return utf8.ValidString(name) && utf8.RuneCountInString(name) >= 1 && utf8.RuneCountInString(name) <= 120 && !strings.ContainsAny(name, "\x00\r\n")
}

// BootstrapAdmin only creates a new school-domain account. It never promotes
// existing accounts and never bypasses proof of ownership of the email inbox.
func (s *Service) BootstrapAdmin(ctx context.Context, name, email, password string) error {
	a, err := s.createAccount(ctx, name, email, password, "admin")
	if err != nil {
		return err
	}
	return s.sendToken(ctx, a.ID, "verify_email")
}

// BootstrapOperator creates a new, email-free administrator through the private
// provisioning path. It never promotes an existing account or sends email.
func (s *Service) BootstrapOperator(ctx context.Context, name, username, password string) error {
	name, username = strings.TrimSpace(name), strings.TrimSpace(username)
	if !validName(name) || !validUsername(username) {
		return errors.New("invalid operator name or username")
	}
	var hash string
	if err := s.passwordWork(ctx, func() error { var err error; hash, err = HashOperatorPassword(password); return err }); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	a, err := q.CreateOperator(ctx, database.CreateOperatorParams{Name: name, Username: pgtype.Text{String: username, Valid: true}, PasswordHash: hash})
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return errConflict
		}
		return err
	}
	if err = audit(ctx, q, pgtype.UUID{}, "account.operator_bootstrapped", a.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// sendToken commits token replacement before SMTP, so a failed delivery can be
// recovered using resend. Tokens and passwords are never included in errors.
func (s *Service) sendToken(ctx context.Context, id pgtype.UUID, purpose string) error {
	token, hash, err := newToken()
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	a, err := q.LockAccount(ctx, id)
	if err != nil {
		return err
	}
	if !a.Email.Valid || a.Username.Valid {
		return errInvalidToken
	}
	if a.Status == "disabled" || a.Status == "rejected" || (purpose == "verify_email" && a.EmailVerifiedAt.Valid) {
		return nil
	}
	if err = q.InvalidateAccountTokens(ctx, database.InvalidateAccountTokensParams{AccountID: id, Purpose: purpose}); err != nil {
		return err
	}
	ttl, path, subject := 24*time.Hour, "verify-email", "KoçBenefits e-posta doğrulama"
	if purpose == "reset_password" {
		ttl, path, subject = 30*time.Minute, "reset-password", "KoçBenefits parola yenileme"
	}
	if err = q.CreateAccountToken(ctx, database.CreateAccountTokenParams{TokenHash: hash, AccountID: id, Purpose: purpose, ExpiresAt: stamp(time.Now().Add(ttl))}); err != nil {
		return err
	}
	// A public resend/reset request identifies its target but authenticates no
	// actor. Recording the target as actor would falsely attribute the request.
	if err = audit(ctx, q, pgtype.UUID{}, "account."+purpose+"_requested", id); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	mailctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err = s.mailer.Send(mailctx, a.Email.String, subject, "İşlemi tamamlamak için bu bağlantıyı açın:\n\n"+s.cfg.AppOrigin+"/#"+path+"?token="+token+"\n\nBu işlemi siz istemediyseniz bu iletiyi yok sayın."); err != nil {
		return errMailUnavailable
	}
	return nil
}

func (s *Service) consumeToken(ctx context.Context, raw, purpose, password string) error {
	hash, ok := tokenHash(raw)
	if !ok {
		return errInvalidToken
	}
	// Hash without holding a connection, then lock the account before consuming.
	// All password/session changes use the same account lock ordering.
	var passwordHash string
	if purpose == "reset_password" {
		if err := s.passwordWork(ctx, func() error { var e error; passwordHash, e = HashPassword(password); return e }); err != nil {
			return err
		}
	}
	candidate, err := s.q.FindAccountToken(ctx, database.FindAccountTokenParams{TokenHash: hash, Purpose: purpose})
	if errors.Is(err, pgx.ErrNoRows) {
		return errInvalidToken
	}
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	a, err := q.LockAccount(ctx, candidate.AccountID)
	if err != nil {
		return err
	}
	if !a.Email.Valid || a.Username.Valid || a.Status == "disabled" || a.Status == "rejected" {
		return errInvalidToken
	}
	if _, err = q.ConsumeAccountToken(ctx, database.ConsumeAccountTokenParams{TokenHash: hash, Purpose: purpose}); errors.Is(err, pgx.ErrNoRows) {
		return errInvalidToken
	} else if err != nil {
		return err
	}
	if purpose == "verify_email" {
		err = q.VerifyAccountEmail(ctx, a.ID)
	} else {
		err = q.ChangePassword(ctx, database.ChangePasswordParams{ID: a.ID, PasswordHash: passwordHash})
	}
	if err != nil {
		return err
	}
	if err = q.InvalidateAccountTokens(ctx, database.InvalidateAccountTokensParams{AccountID: a.ID, Purpose: purpose}); err != nil {
		return err
	}
	if purpose == "reset_password" {
		if err = q.RevokeAccountSessions(ctx, a.ID); err != nil {
			return err
		}
	}
	if err = audit(ctx, q, a.ID, "account."+purpose+"_completed", a.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
