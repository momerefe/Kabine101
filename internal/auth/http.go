package auth

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

const SessionCookie = "kb_session"
const CSRFCookie = "kb_csrf"

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Get("/api/auth/session", s.session)
	r.Get("/api/admin/accounts", s.accounts)
	r.Group(func(r chi.Router) {
		r.Use(s.csrf)
		r.Post("/api/auth/register", s.register)
		r.Post("/api/auth/login", s.login)
		r.Post("/api/auth/admin-login", s.adminLogin)
		r.Post("/api/auth/logout", s.logout)
		r.Post("/api/auth/verify-email", s.verify)
		r.Post("/api/auth/resend-verification", s.resend)
		r.Post("/api/auth/forgot-password", s.forgot)
		r.Post("/api/auth/reset-password", s.reset)
		r.Post("/api/admin/accounts/{id}/review", s.review)
		r.Post("/api/admin/accounts/{id}/role", s.role)
		r.Post("/api/admin/accounts/{id}/status", s.status)
	})
}

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, code, message string) {
	reply(w, status, map[string]string{"code": code, "message": message})
}
func message(w http.ResponseWriter, status int, text string) {
	reply(w, status, map[string]string{"message": text})
}
func unavailable(w http.ResponseWriter) {
	fail(w, 503, "unavailable", "İşlem şu anda tamamlanamıyor. Lütfen yeniden deneyin.")
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil || len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '{' {
		fail(w, 400, "invalid_request", "İstek bir JSON nesnesi içermelidir.")
		return false
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		fail(w, 400, "invalid_request", "Geçerli bir JSON isteği gönderin.")
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		fail(w, 400, "invalid_request", "İstek tek bir JSON nesnesi içermelidir.")
		return false
	}
	return true
}

func (s *Service) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != s.cfg.AppOrigin {
			fail(w, 403, "csrf", "İstek kaynağı doğrulanamadı.")
			return
		}
		c, err := r.Cookie(s.cookieName(CSRFCookie))
		token := r.Header.Get("X-CSRF-Token")
		if err != nil {
			fail(w, 403, "csrf", "Güvenlik anahtarı eksik. Sayfayı yenileyin.")
			return
		}
		if _, ok := tokenHash(c.Value); !ok || subtle.ConstantTimeCompare([]byte(c.Value), []byte(token)) != 1 {
			fail(w, 403, "csrf", "Güvenlik anahtarı geçersiz. Sayfayı yenileyin.")
			return
		}
		content, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || content != "application/json" {
			fail(w, 400, "invalid_request", "Content-Type application/json olmalıdır.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) cookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	c := &http.Cookie{Name: s.cookieName(name), Value: value, Path: "/", HttpOnly: true, Secure: s.cfg.Environment == "production", SameSite: http.SameSiteLaxMode, MaxAge: int(ttl.Seconds()), Expires: time.Now().Add(ttl)}
	if ttl < 0 {
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, c)
}

func (s *Service) cookieName(name string) string {
	if s.cfg.Environment == "production" {
		return "__Host-" + name
	}
	return name
}

func (s *Service) limited(w http.ResponseWriter, r *http.Request, scope, email string) bool {
	if !s.limiter.allow(r, scope, email) {
		w.Header().Set("Retry-After", "900")
		fail(w, 429, "rate_limited", "Çok fazla deneme yapıldı. Lütfen daha sonra yeniden deneyin.")
		return true
	}
	return false
}

func (s *Service) authenticated(r *http.Request) (database.Account, []byte, error) {
	c, err := r.Cookie(s.cookieName(SessionCookie))
	if err != nil {
		return database.Account{}, nil, errDenied
	}
	hash, ok := tokenHash(c.Value)
	if !ok {
		return database.Account{}, nil, errDenied
	}
	a, err := s.q.SessionAccount(r.Context(), hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, nil, errDenied
	}
	if err != nil {
		return a, nil, err
	}
	if !s.eligible(a) {
		return a, nil, errDenied
	}
	return a, hash, nil
}

func (s *Service) requireAdmin(w http.ResponseWriter, r *http.Request) (database.Account, []byte, bool) {
	a, hash, err := s.authenticated(r)
	if err != nil {
		if errors.Is(err, errDenied) {
			s.cookie(w, SessionCookie, "", -time.Second)
			fail(w, 401, "unauthorized", "Oturum açmanız gerekiyor.")
		} else {
			unavailable(w)
		}
		return a, nil, false
	}
	if a.Role != "admin" {
		fail(w, 403, "forbidden", "Bu işlem için yönetici yetkisi gerekiyor.")
		return a, nil, false
	}
	return a, hash, true
}

func (s *Service) session(w http.ResponseWriter, r *http.Request) {
	var user *Account
	if _, err := r.Cookie(s.cookieName(SessionCookie)); err == nil {
		a, _, err := s.authenticated(r)
		if err == nil {
			p := publicAccount(a)
			user = &p
		} else if errors.Is(err, errDenied) {
			s.cookie(w, SessionCookie, "", -time.Second)
		} else {
			unavailable(w)
			return
		}
	}
	var token string
	if c, err := r.Cookie(s.cookieName(CSRFCookie)); err == nil {
		if _, ok := tokenHash(c.Value); ok {
			token = c.Value
		}
	}
	if token == "" {
		var err error
		token, _, err = newToken()
		if err != nil {
			unavailable(w)
			return
		}
		s.cookie(w, CSRFCookie, token, 24*time.Hour)
	}
	reply(w, 200, struct {
		User             *Account `json:"user"`
		CSRFToken        string   `json:"csrf_token"`
		SchoolDomain     string   `json:"school_domain"`
		RegistrationOpen bool     `json:"registration_open"`
		LocalEmail       bool     `json:"local_email"`
	}{user, token, s.cfg.SchoolDomain, s.cfg.RegistrationOpen, s.cfg.LocalEmail})
}

func (s *Service) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	email, ok := s.email(in.Email)
	if !ok || !validPassword(in.Password) || !validName(in.Name) {
		fail(w, 400, "invalid_request", "Ad, okul e-posta adresi ve 15–128 karakterlik parola gerekiyor.")
		return
	}
	if !s.cfg.RegistrationOpen {
		fail(w, 403, "registration_closed", "Yeni hesap kaydı şu anda kapalı.")
		return
	}
	if s.limited(w, r, "register", email) {
		return
	}
	a, err := s.createAccount(r.Context(), in.Name, email, in.Password, "student")
	if errors.Is(err, errBusy) {
		s.busy(w)
		return
	}
	if err != nil && !errors.Is(err, errConflict) {
		unavailable(w)
		return
	}
	if err == nil {
		if err = s.sendToken(r.Context(), a.ID, "verify_email"); err != nil && !errors.Is(err, errMailUnavailable) {
			unavailable(w)
			return
		}
		if errors.Is(err, errMailUnavailable) {
			slog.Error("authentication email delivery failed; resend is available")
		}
	}
	message(w, 202, "İstek alındı. Gelen kutunuzu kontrol edin; ileti ulaşmazsa doğrulama iletisini yeniden isteyin.")
}

func (s *Service) busy(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "2")
	fail(w, 429, "rate_limited", "Kimlik doğrulama yoğun. Lütfen yeniden deneyin.")
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	email, school := s.email(in.Email)
	if !school {
		email = strings.ToLower(strings.TrimSpace(in.Email))
	}
	if s.limited(w, r, "login", email) {
		return
	}
	if len(in.Password) > 512 {
		fail(w, 401, "invalid_credentials", "E-posta veya parola hatalı.")
		return
	}
	a, err := s.q.FindAccountByEmail(r.Context(), pgtype.Text{String: email, Valid: true})
	s.loginAccount(w, r, a, err, in.Password, "login", email, school && !a.Username.Valid)
}

func (s *Service) adminLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	username := strings.TrimSpace(in.Username)
	identity := "username:" + strings.ToLower(username)
	if s.limited(w, r, "admin_login", identity) {
		return
	}
	if len(in.Password) > 512 {
		fail(w, 401, "invalid_credentials", "Kullanıcı adı veya parola hatalı.")
		return
	}
	a, err := s.q.FindAccountByUsername(r.Context(), username)
	s.loginAccount(w, r, a, err, in.Password, "admin_login", identity, validUsername(username) && a.Username.Valid && !a.Email.Valid && a.Role == "admin")
}

// Both login identities share password work, state checks, account locking,
// audit events and session issuance.
func (s *Service) loginAccount(w http.ResponseWriter, r *http.Request, a database.Account, err error, password, scope, identity string, allowedIdentity bool) {
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		unavailable(w)
		return
	}
	encoded := s.dummyHash
	if err == nil {
		encoded = a.PasswordHash
	}
	matched := false
	if err = s.passwordWork(r.Context(), func() error { matched = checkPassword(encoded, password); return nil }); err != nil {
		s.busy(w)
		return
	}
	if !matched || !allowedIdentity || !a.ID.Valid || a.Status == "disabled" || a.Status == "rejected" {
		fail(w, 401, "invalid_credentials", "Giriş bilgileri hatalı.")
		return
	}
	s.limiter.resetEmail(scope, identity)
	if a.Status != "active" || a.EligibilityStatus != "approved" {
		fail(w, 403, "pending_approval", "Hesabınız yönetici onayı bekliyor.")
		return
	}
	if !a.Username.Valid && !a.EmailVerifiedAt.Valid {
		fail(w, 403, "email_unverified", "Önce e-posta adresinizi doğrulayın.")
		return
	}
	raw, hash, err := newToken()
	if err != nil {
		unavailable(w)
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		unavailable(w)
		return
	}
	defer tx.Rollback(r.Context())
	q := s.q.WithTx(tx)
	current, err := q.LockAccount(r.Context(), a.ID)
	if err != nil {
		unavailable(w)
		return
	}
	if !s.eligible(current) || current.PasswordHash != a.PasswordHash {
		fail(w, 401, "invalid_credentials", "E-posta veya parola hatalı.")
		return
	}
	if err = q.CreateSession(r.Context(), database.CreateSessionParams{TokenHash: hash, AccountID: a.ID, ExpiresAt: stamp(time.Now().Add(24 * time.Hour))}); err != nil {
		unavailable(w)
		return
	}
	if err = audit(r.Context(), q, a.ID, "account.login", a.ID); err != nil {
		unavailable(w)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		unavailable(w)
		return
	}
	s.cookie(w, SessionCookie, raw, 24*time.Hour)
	reply(w, 200, map[string]Account{"user": publicAccount(current)})
}

func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	var in struct{}
	if !decode(w, r, &in) {
		return
	}
	if c, err := r.Cookie(s.cookieName(SessionCookie)); err == nil {
		if h, ok := tokenHash(c.Value); ok {
			if err = s.endSession(r, h); err != nil {
				unavailable(w)
				return
			}
		}
	}
	s.cookie(w, SessionCookie, "", -time.Second)
	message(w, 200, "Oturum kapatıldı.")
}

func (s *Service) verify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	if s.limited(w, r, "verify", "") {
		return
	}
	if err := s.consumeToken(r.Context(), in.Token, "verify_email", ""); err != nil {
		s.tokenError(w, err)
		return
	}
	message(w, 200, "E-posta adresiniz doğrulandı. Yönetici onayınız tamamlandığında giriş yapabilirsiniz.")
}

func (s *Service) reset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !validPassword(in.Password) {
		fail(w, 400, "invalid_request", "Parola 15–128 karakter olmalıdır.")
		return
	}
	if s.limited(w, r, "reset", "") {
		return
	}
	if err := s.consumeToken(r.Context(), in.Token, "reset_password", in.Password); err != nil {
		s.tokenError(w, err)
		return
	}
	s.cookie(w, SessionCookie, "", -time.Second)
	message(w, 200, "Parolanız yenilendi. Yeniden giriş yapın.")
}

func (s *Service) tokenError(w http.ResponseWriter, err error) {
	if errors.Is(err, errInvalidToken) {
		fail(w, 400, "invalid_token", "Bağlantı geçersiz veya süresi dolmuş. Yeni bir ileti isteyin.")
	} else if errors.Is(err, errBusy) {
		s.busy(w)
	} else {
		unavailable(w)
	}
}

func (s *Service) resend(w http.ResponseWriter, r *http.Request) {
	s.requestToken(w, r, "verify_email")
}
func (s *Service) forgot(w http.ResponseWriter, r *http.Request) {
	s.requestToken(w, r, "reset_password")
}
func (s *Service) requestToken(w http.ResponseWriter, r *http.Request, purpose string) {
	var in struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	email, ok := s.email(in.Email)
	if s.limited(w, r, purpose, strings.ToLower(strings.TrimSpace(in.Email))) {
		return
	}
	if ok {
		a, err := s.q.FindAccountByEmail(r.Context(), pgtype.Text{String: email, Valid: true})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			unavailable(w)
			return
		}
		if err == nil {
			if err = s.sendToken(r.Context(), a.ID, purpose); err != nil && !errors.Is(err, errMailUnavailable) {
				unavailable(w)
				return
			}
			if errors.Is(err, errMailUnavailable) {
				slog.Error("authentication email delivery failed; resend is available")
			}
		}
	}
	message(w, 202, "İstek alındı. Gelen kutunuzu kontrol edin; ileti ulaşmazsa yeniden deneyin.")
}

func (s *Service) endSession(r *http.Request, hash []byte) error {
	ctx := r.Context()
	prior, err := s.q.FindSession(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
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
	if _, err = q.LockAccount(ctx, prior.AccountID); err != nil {
		return err
	}
	if _, err = q.FindSession(ctx, hash); errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	if err = q.DeleteSession(ctx, hash); err != nil {
		return err
	}
	if err = audit(ctx, q, prior.AccountID, "account.logout", prior.AccountID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
