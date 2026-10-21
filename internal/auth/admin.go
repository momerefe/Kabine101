package auth

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

func (s *Service) accounts(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	query := r.URL.Query()
	q := strings.TrimSpace(query.Get("q"))
	role, status, eligibility := query.Get("role"), query.Get("status"), query.Get("eligibility_status")
	validChoice := func(v string, choices ...string) bool {
		if v == "" {
			return true
		}
		for _, choice := range choices {
			if v == choice {
				return true
			}
		}
		return false
	}
	if !utf8.ValidString(q) || utf8.RuneCountInString(q) > 100 || strings.ContainsRune(q, 0) || !validChoice(role, "student", "admin") || !validChoice(status, "active", "pending", "rejected", "disabled") || !validChoice(eligibility, "pending", "approved", "rejected") {
		fail(w, 400, "invalid_request", "Geçersiz hesap filtresi.")
		return
	}
	rows, err := s.q.ListAccounts(r.Context(), database.ListAccountsParams{Q: q, Role: role, Status: status, EligibilityStatus: eligibility})
	if err != nil {
		unavailable(w)
		return
	}
	out := make([]Account, 0, len(rows))
	for i, a := range rows {
		if i == 1000 {
			break
		}
		out = append(out, publicAccount(a))
	}
	reply(w, 200, map[string]any{"accounts": out, "truncated": len(rows) > 1000})
}

func (s *Service) status(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status          string `json:"status"`
		CurrentPassword string `json:"current_password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Status != "active" && in.Status != "disabled" {
		fail(w, 400, "invalid_request", "Durum active veya disabled olmalıdır.")
		return
	}
	s.adminChange(w, r, in.CurrentPassword, "status", in.Status)
}

func (s *Service) review(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision        string `json:"decision"`
		CurrentPassword string `json:"current_password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Decision != "approved" && in.Decision != "rejected" {
		fail(w, 400, "invalid_request", "Karar approved veya rejected olmalıdır.")
		return
	}
	s.adminChange(w, r, in.CurrentPassword, "review", in.Decision)
}

func (s *Service) role(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role            string `json:"role"`
		CurrentPassword string `json:"current_password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Role != "student" && in.Role != "admin" {
		fail(w, 400, "invalid_request", "Rol student veya admin olmalıdır.")
		return
	}
	s.adminChange(w, r, in.CurrentPassword, "role", in.Role)
}

func (s *Service) adminChange(w http.ResponseWriter, r *http.Request, password, kind, value string) {
	actor, sessionHash, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var targetID pgtype.UUID
	if err := targetID.Scan(chi.URLParam(r, "id")); err != nil || !targetID.Valid {
		fail(w, 400, "invalid_request", "Geçersiz hesap kimliği.")
		return
	}
	if s.limited(w, r, "admin_reauth", accountIdentity(actor)) {
		return
	}
	if len(password) > 512 {
		fail(w, 401, "invalid_credentials", "Mevcut parola doğrulanamadı.")
		return
	}
	matched := false
	if err := s.passwordWork(r.Context(), func() error { matched = checkPassword(actor.PasswordHash, password); return nil }); err != nil {
		s.busy(w)
		return
	}
	if !matched {
		fail(w, 401, "invalid_credentials", "Mevcut parola doğrulanamadı.")
		return
	}
	s.limiter.resetEmail("admin_reauth", accountIdentity(actor))
	err := s.changeAccount(r, actor, sessionHash, targetID, kind, value)
	switch {
	case err == nil:
		message(w, 200, "Hesap güncellendi.")
	case errors.Is(err, errDenied):
		fail(w, 403, "forbidden", "Yönetici yetkisi veya oturum geçerli değil.")
	case errors.Is(err, errLastAdmin):
		fail(w, 409, "last_admin", "Son etkin yöneticinin erişimi kaldırılamaz.")
	case errors.Is(err, errConflict):
		fail(w, 409, "account_state", "Hesabın mevcut durumu bu işleme uygun değil.")
	case errors.Is(err, pgx.ErrNoRows):
		fail(w, 400, "invalid_request", "Hesap bulunamadı.")
	default:
		unavailable(w)
	}
}

func (s *Service) changeAccount(r *http.Request, priorActor database.Account, sessionHash []byte, targetID pgtype.UUID, kind, value string) error {
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// A transaction-scoped PostgreSQL lock serializes approval, role and status changes.
	// Concurrent removals therefore cannot both observe two administrators.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(180181001)"); err != nil {
		return err
	}
	q := s.q.WithTx(tx)
	actor, err := q.LockAccount(ctx, priorActor.ID)
	if err != nil {
		return err
	}
	if !s.eligible(actor) || actor.Role != "admin" || actor.PasswordHash != priorActor.PasswordHash {
		return errDenied
	}
	current, err := q.SessionAccount(ctx, sessionHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return errDenied
	}
	if err != nil {
		return err
	}
	if current.ID != actor.ID {
		return errDenied
	}
	target, err := q.LockAccount(ctx, targetID)
	if err != nil {
		return err
	}
	if kind == "review" {
		if target.EligibilityStatus != "pending" || target.Status != "pending" {
			return errConflict
		}
		status := "active"
		if value == "rejected" {
			status = "rejected"
		}
		if err = q.ReviewAccount(ctx, database.ReviewAccountParams{ID: target.ID, Status: status, EligibilityStatus: value}); err != nil {
			return err
		}
		if value == "rejected" {
			if err = q.RevokeAccountSessions(ctx, target.ID); err != nil {
				return err
			}
		}
	} else if kind == "status" {
		if target.Status == value || target.EligibilityStatus != "approved" {
			return errConflict
		}
		if value == "disabled" {
			if !s.eligible(target) {
				return errConflict
			}
			if target.Role == "admin" {
				count, err := q.CountActiveAdmins(ctx, s.cfg.SchoolDomain)
				if err != nil {
					return err
				}
				if count <= 1 {
					return errLastAdmin
				}
			}
		} else {
			if target.Status != "disabled" {
				return errConflict
			}
			candidate := target
			candidate.Status = "active"
			if !s.eligible(candidate) {
				return errConflict
			}
		}
		if err = q.ChangeAccountStatus(ctx, database.ChangeAccountStatusParams{ID: target.ID, Status: value}); err != nil {
			return err
		}
		if err = q.RevokeAccountSessions(ctx, target.ID); err != nil {
			return err
		}
	} else {
		if target.Username.Valid && value != "admin" {
			return errConflict
		}
		if target.Role == value {
			return errConflict
		}
		if value == "admin" && !s.eligible(target) {
			return errConflict
		}
		if target.Role == "admin" && s.eligible(target) {
			count, err := q.CountActiveAdmins(ctx, s.cfg.SchoolDomain)
			if err != nil {
				return err
			}
			if count <= 1 {
				return errLastAdmin
			}
		}
		if err = q.ChangeRole(ctx, database.ChangeRoleParams{ID: target.ID, Role: value}); err != nil {
			return err
		}
		// All role changes require a fresh login with canonical permissions.
		if err = q.RevokeAccountSessions(ctx, target.ID); err != nil {
			return err
		}
	}
	if err = audit(ctx, q, actor.ID, "account."+kind+"."+value, target.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
