//go:build integration

package auth_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/momerefe/Kabine101/internal/auth"
	"github.com/momerefe/Kabine101/internal/config"
	"github.com/momerefe/Kabine101/internal/httpapi"
	"github.com/momerefe/Kabine101/internal/migrations"
)

const testOrigin = "http://127.0.0.1:5173"
const testPassword = "Test-only long password 2026!"
const changedPassword = "Changed long password 2026!"

type testAccount struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Username          string `json:"username"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	EmailVerified     bool   `json:"email_verified"`
	EligibilityStatus string `json:"eligibility_status"`
}

type testResponse struct {
	Status int
	Body   []byte
	Header http.Header
}

type capturedMessage struct{ To, Subject, Body string }
type capturingMailer struct {
	mu       sync.Mutex
	messages []capturedMessage
	fail     bool
}

func (m *capturingMailer) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, capturedMessage{to, subject, body})
	if m.fail {
		return fmt.Errorf("synthetic SMTP delivery failure")
	}
	return nil
}

func (m *capturingMailer) token(t *testing.T, email, purpose string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	pattern := regexp.MustCompile(`#` + regexp.QuoteMeta(purpose) + `\?token=([A-Za-z0-9_-]+)`)
	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := m.messages[i]
		if msg.To == email {
			if match := pattern.FindStringSubmatch(msg.Body); len(match) == 2 {
				return match[1]
			}
		}
	}
	t.Fatalf("no %s email captured for synthetic account", purpose)
	return ""
}

func (m *capturingMailer) count() int { m.mu.Lock(); defer m.mu.Unlock(); return len(m.messages) }

type authHarness struct {
	t       *testing.T
	pool    *pgxpool.Pool
	server  *httptest.Server
	mailer  *capturingMailer
	service *auth.Service
	cfg     config.Config
}

type browser struct {
	h      *authHarness
	client *http.Client
	csrf   string
}

func newHarness(t *testing.T, overrides ...func(*config.Config)) *authHarness {
	t.Helper()
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Fatal("TEST_DATABASE_URL is required for explicitly requested integration tests")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("auth integration tests require a dedicated PostgreSQL database ending in _test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, raw)
	if err != nil {
		t.Fatal("cannot connect to test database")
	}
	schema := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		admin.Close(context.Background())
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("cannot remove owned auth test schema: %v", err)
		}
		admin.Close(context.Background())
	})
	params := u.Query()
	params.Set("search_path", schema)
	u.RawQuery = params.Encode()
	if err = migrations.Up(u.String()); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal("cannot create test schema pool")
	}
	t.Cleanup(pool.Close)
	mailer := &capturingMailer{}
	cfg := config.Config{Environment: "test", AppOrigin: testOrigin, SchoolDomain: "ogrenci.example", EligibilityPolicy: "manual", RegistrationOpen: true, LocalEmail: true}
	for _, override := range overrides {
		override(&cfg)
	}
	service, err := auth.New(pool, cfg, mailer)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.Router(func(context.Context) error { return nil }, service))
	t.Cleanup(server.Close)
	return &authHarness{t: t, pool: pool, server: server, mailer: mailer, service: service, cfg: cfg}
}

func (h *authHarness) browser() *browser {
	h.t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		h.t.Fatal(err)
	}
	b := &browser{h: h, client: &http.Client{Jar: jar, Timeout: 15 * time.Second}}
	b.session()
	return b
}

func (b *browser) request(method, path string, payload any, csrf, origin string) testResponse {
	b.h.t.Helper()
	var data []byte
	var err error
	if payload != nil {
		data, err = json.Marshal(payload)
		if err != nil {
			b.h.t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, b.h.server.URL+path, bytes.NewReader(data))
	if err != nil {
		b.h.t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		b.h.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		b.h.t.Fatal(err)
	}
	return testResponse{resp.StatusCode, body, resp.Header}
}

func (b *browser) post(path string, payload any) testResponse {
	return b.request(http.MethodPost, path, payload, b.csrf, testOrigin)
}

func (r testResponse) want(t *testing.T, status int) testResponse {
	t.Helper()
	if r.Status != status {
		t.Fatalf("HTTP status = %d, want %d; body=%s", r.Status, status, r.Body)
	}
	return r
}

func (r testResponse) denied(t *testing.T) {
	t.Helper()
	if r.Status < 400 || r.Status >= 500 {
		t.Fatalf("expected client rejection, got %d; body=%s", r.Status, r.Body)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(r.Body, &body); err != nil || body.Code == "" || body.Message == "" {
		t.Fatalf("rejection must use documented JSON error: %s", r.Body)
	}
}

func (b *browser) session() *testAccount {
	b.h.t.Helper()
	r := b.request(http.MethodGet, "/api/auth/session", nil, "", "").want(b.h.t, http.StatusOK)
	var body struct {
		User             *testAccount `json:"user"`
		CSRF             string       `json:"csrf_token"`
		SchoolDomain     string       `json:"school_domain"`
		RegistrationOpen bool         `json:"registration_open"`
		LocalEmail       bool         `json:"local_email"`
	}
	if err := json.Unmarshal(r.Body, &body); err != nil {
		b.h.t.Fatal(err)
	}
	if body.CSRF == "" || body.SchoolDomain != b.h.cfg.SchoolDomain || body.RegistrationOpen != b.h.cfg.RegistrationOpen || body.LocalEmail != b.h.cfg.LocalEmail {
		b.h.t.Fatalf("invalid session metadata: %s", r.Body)
	}
	if !strings.Contains(r.Header.Get("Cache-Control"), "no-store") {
		b.h.t.Fatal("session response must forbid caching")
	}
	b.csrf = body.CSRF
	return body.User
}

func (h *authHarness) accountID(email string) string {
	h.t.Helper()
	var id string
	if err := h.pool.QueryRow(context.Background(), "SELECT id::text FROM accounts WHERE email=$1", email).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

func (b *browser) register(email string) string {
	b.h.t.Helper()
	b.post("/api/auth/register", map[string]any{"name": "Synthetic student", "email": email, "password": testPassword}).want(b.h.t, http.StatusAccepted)
	return b.h.accountID(email)
}

func (b *browser) verify(email string) string {
	b.h.t.Helper()
	token := b.h.mailer.token(b.h.t, email, "verify-email")
	b.post("/api/auth/verify-email", map[string]string{"token": token}).want(b.h.t, http.StatusOK)
	return token
}

func (b *browser) login(email, password string) testResponse {
	r := b.post("/api/auth/login", map[string]string{"email": email, "password": password})
	if r.Status == http.StatusOK {
		b.session()
	}
	return r
}

func (h *authHarness) admin() (*browser, string) {
	h.t.Helper()
	const email = "bootstrap@ogrenci.example"
	if err := h.service.BootstrapAdmin(context.Background(), "Synthetic admin", email, testPassword); err != nil {
		h.t.Fatal(err)
	}
	b := h.browser()
	b.login(email, testPassword).denied(h.t)
	b.verify(email)
	b.login(email, testPassword).want(h.t, http.StatusOK)
	if u := b.session(); u == nil || u.Role != "admin" || !u.EmailVerified {
		h.t.Fatal("verified bootstrap administrator did not receive admin session")
	}
	return b, h.accountID(email)
}

func (b *browser) review(id, decision string) testResponse {
	return b.post("/api/admin/accounts/"+id+"/review", map[string]string{"decision": decision, "current_password": testPassword})
}

func (h *authHarness) activeStudent(admin *browser, email string) (*browser, string) {
	h.t.Helper()
	b := h.browser()
	id := b.register(email)
	b.verify(email)
	admin.review(id, "approved").want(h.t, http.StatusOK)
	b.login(email, testPassword).want(h.t, http.StatusOK)
	return b, id
}

func TestRegistrationValidationAndGenericResponses(t *testing.T) {
	h := newHarness(t)
	b := h.browser()
	for _, email := range []string{"student@example.net", "student@sub.ogrenci.example", "student@ogrenci.example.attacker.test", "student@evilogrenci.example", "student@@ogrenci.example", "Student <student@ogrenci.example>"} {
		b.post("/api/auth/register", map[string]any{"name": "Test", "email": email, "password": testPassword}).denied(t)
	}
	for _, extra := range []map[string]any{{"role": "admin"}, {"status": "active"}, {"email_verified": true}, {"eligibility_status": "approved"}, {"username": "InjectedOperator"}, {"unexpected": "field"}} {
		payload := map[string]any{"name": "Test", "email": "injection@ogrenci.example", "password": testPassword}
		for k, v := range extra {
			payload[k] = v
		}
		b.post("/api/auth/register", payload).denied(t)
	}
	for _, password := range []string{"short", "246813579", strings.Repeat("a", 14), strings.Repeat("a", 129)} {
		b.post("/api/auth/register", map[string]string{"name": "Test", "email": "length@ogrenci.example", "password": password}).denied(t)
	}
	for _, name := range []string{"", "   ", strings.Repeat("a", 121), "Test\nInjected", "Test\x00Injected"} {
		b.post("/api/auth/register", map[string]string{"name": name, "email": "name@ogrenci.example", "password": testPassword}).want(t, http.StatusBadRequest)
	}
	r := b.post("/api/auth/register", map[string]string{"name": "Test", "email": "  NORMALIZED@OGRENCI.EXAMPLE  ", "password": testPassword}).want(t, http.StatusAccepted)
	if h.accountID("normalized@ogrenci.example") == "" {
		t.Fatal("email was not normalized")
	}
	before := h.mailer.count()
	duplicate := b.post("/api/auth/register", map[string]string{"name": "Other", "email": "normalized@ogrenci.example", "password": changedPassword}).want(t, http.StatusAccepted)
	if !bytes.Equal(r.Body, duplicate.Body) {
		t.Fatal("registration response reveals existing account")
	}
	if h.mailer.count() != before {
		t.Fatal("duplicate registration unexpectedly generated verification message")
	}
	known := b.post("/api/auth/forgot-password", map[string]string{"email": "normalized@ogrenci.example"}).want(t, http.StatusAccepted)
	unknown := b.post("/api/auth/forgot-password", map[string]string{"email": "unknown@ogrenci.example"}).want(t, http.StatusAccepted)
	if !bytes.Equal(known.Body, unknown.Body) {
		t.Fatal("password reset response reveals existing account")
	}
	known = b.post("/api/auth/resend-verification", map[string]string{"email": "normalized@ogrenci.example"}).want(t, http.StatusAccepted)
	unknown = b.post("/api/auth/resend-verification", map[string]string{"email": "unknown@ogrenci.example"}).want(t, http.StatusAccepted)
	if !bytes.Equal(known.Body, unknown.Body) {
		t.Fatal("verification resend reveals existing account")
	}
	wrong := b.login("normalized@ogrenci.example", "Wrong long password 2026!")
	missing := b.login("unknown@ogrenci.example", "Wrong long password 2026!")
	wrong.want(t, http.StatusUnauthorized)
	missing.want(t, http.StatusUnauthorized)
	if !bytes.Equal(wrong.Body, missing.Body) {
		t.Fatal("login reveals existence through error body")
	}
}

func TestVerificationAndApprovalBothRequired(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.admin()
	verifiedFirst := h.browser()
	firstID := verifiedFirst.register("verify-first@ogrenci.example")
	verifiedFirst.login("verify-first@ogrenci.example", testPassword).denied(t)
	verifiedFirst.verify("verify-first@ogrenci.example")
	verifiedFirst.login("verify-first@ogrenci.example", testPassword).denied(t)
	admin.review(firstID, "approved").want(t, http.StatusOK)
	verifiedFirst.login("verify-first@ogrenci.example", testPassword).want(t, http.StatusOK)
	user := verifiedFirst.session()
	if user == nil || user.Role != "student" || user.Status != "active" || user.EligibilityStatus != "approved" || !user.EmailVerified {
		t.Fatal("unexpected approved student account")
	}
	approvedFirst := h.browser()
	secondID := approvedFirst.register("approve-first@ogrenci.example")
	admin.review(secondID, "approved").want(t, http.StatusOK)
	approvedFirst.login("approve-first@ogrenci.example", testPassword).denied(t)
	var verified bool
	if err := h.pool.QueryRow(context.Background(), "SELECT email_verified_at IS NOT NULL FROM accounts WHERE id=$1", secondID).Scan(&verified); err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("admin approval waived mailbox verification")
	}
	approvedFirst.verify("approve-first@ogrenci.example")
	approvedFirst.login("approve-first@ogrenci.example", testPassword).want(t, http.StatusOK)
	verifiedFirst.request(http.MethodGet, "/api/admin/accounts", nil, "", "").want(t, http.StatusForbidden)
	verifiedFirst.review(secondID, "rejected").want(t, http.StatusForbidden)
	verifiedFirst.post("/api/admin/accounts/"+secondID+"/role", map[string]string{"role": "admin", "current_password": testPassword}).want(t, http.StatusForbidden)
	if err := h.pool.QueryRow(context.Background(), "SELECT email_verified_at IS NOT NULL FROM accounts WHERE id=$1", secondID).Scan(&verified); err != nil || !verified {
		t.Fatal("forbidden mutation changed account")
	}
	rejected := h.browser()
	rejectedID := rejected.register("rejected@ogrenci.example")
	rejected.verify("rejected@ogrenci.example")
	admin.review(rejectedID, "rejected").want(t, http.StatusOK)
	rejected.login("rejected@ogrenci.example", testPassword).want(t, http.StatusUnauthorized)
	admin.review(rejectedID, "approved").want(t, http.StatusConflict)
	listed := admin.request(http.MethodGet, "/api/admin/accounts", nil, "", "").want(t, http.StatusOK)
	var accounts struct {
		Accounts []testAccount `json:"accounts"`
	}
	if err := json.Unmarshal(listed.Body, &accounts); err != nil || len(accounts.Accounts) != 4 {
		t.Fatal("administrator account list omitted synthetic accounts")
	}
	for _, secret := range []string{"password_hash", "token_hash", testPassword} {
		if bytes.Contains(listed.Body, []byte(secret)) {
			t.Fatal("administrator account list exposed authentication secrets")
		}
	}
}

func TestCSRFOriginAndSessionCookieProtection(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.admin()
	student, id := h.activeStudent(admin, "csrf@ogrenci.example")
	for _, tc := range []struct{ name, csrf, origin string }{{"missing token", "", testOrigin}, {"wrong token", "attacker", testOrigin}, {"untrusted origin", student.csrf, "https://attacker.example"}, {"null origin", student.csrf, "null"}, {"missing origin", student.csrf, ""}} {
		t.Run(tc.name, func(t *testing.T) {
			student.request(http.MethodPost, "/api/auth/logout", map[string]any{}, tc.csrf, tc.origin).want(t, http.StatusForbidden)
		})
	}
	if student.session() == nil {
		t.Fatal("rejected logout revoked session")
	}
	other := h.browser()
	other.request(http.MethodPost, "/api/auth/login", map[string]string{"email": "csrf@ogrenci.example", "password": testPassword}, student.csrf, testOrigin).want(t, http.StatusForbidden)
	other.request(http.MethodPost, "/api/auth/login", map[string]string{"email": "csrf@ogrenci.example", "password": testPassword}, "", testOrigin).want(t, http.StatusForbidden)
	other.request(http.MethodPost, "/api/auth/login", map[string]string{"email": "csrf@ogrenci.example", "password": testPassword}, other.csrf, "https://attacker.example").want(t, http.StatusForbidden)
	admin.request(http.MethodPost, "/api/admin/accounts/"+id+"/review", map[string]string{"decision": "rejected", "current_password": testPassword}, "", testOrigin).want(t, http.StatusForbidden)
	login := other.login("csrf@ogrenci.example", testPassword).want(t, http.StatusOK)
	var found bool
	for _, raw := range login.Header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(raw)
		if err != nil {
			t.Fatal(err)
		}
		if cookie.HttpOnly {
			found = true
			if cookie.Path != "/" || (cookie.SameSite != http.SameSiteLaxMode && cookie.SameSite != http.SameSiteStrictMode) {
				t.Fatal("session cookie must have root path and SameSite protection")
			}
		}
	}
	if !found {
		t.Fatal("login did not set an HttpOnly session cookie")
	}
}

func TestTokenPurposeExpiryAndSingleUse(t *testing.T) {
	h := newHarness(t)
	b := h.browser()
	id := b.register("tokens@ogrenci.example")
	verify := h.mailer.token(t, "tokens@ogrenci.example", "verify-email")
	b.post("/api/auth/reset-password", map[string]string{"token": verify, "password": changedPassword}).denied(t)
	b.post("/api/auth/verify-email", map[string]string{"token": verify}).want(t, http.StatusOK)
	b.post("/api/auth/verify-email", map[string]string{"token": verify}).denied(t)
	b.post("/api/auth/forgot-password", map[string]string{"email": "tokens@ogrenci.example"}).want(t, http.StatusAccepted)
	reset := h.mailer.token(t, "tokens@ogrenci.example", "reset-password")
	b.post("/api/auth/verify-email", map[string]string{"token": reset}).denied(t)
	b.post("/api/auth/reset-password", map[string]string{"token": reset, "password": changedPassword}).want(t, http.StatusOK)
	b.post("/api/auth/reset-password", map[string]string{"token": reset, "password": testPassword}).denied(t)
	b.post("/api/auth/reset-password", map[string]string{"token": "not-a-real-token", "password": testPassword}).denied(t)
	b.post("/api/auth/forgot-password", map[string]string{"email": "tokens@ogrenci.example"}).want(t, http.StatusAccepted)
	expired := h.mailer.token(t, "tokens@ogrenci.example", "reset-password")
	if _, err := h.pool.Exec(context.Background(), "UPDATE account_tokens SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE account_id=$1 AND consumed_at IS NULL", id); err != nil {
		t.Fatal(err)
	}
	b.post("/api/auth/reset-password", map[string]string{"token": expired, "password": testPassword}).denied(t)
	secondID := b.register("expired-verify@ogrenci.example")
	expiredVerify := h.mailer.token(t, "expired-verify@ogrenci.example", "verify-email")
	if _, err := h.pool.Exec(context.Background(), "UPDATE account_tokens SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE account_id=$1", secondID); err != nil {
		t.Fatal(err)
	}
	b.post("/api/auth/verify-email", map[string]string{"token": expiredVerify}).denied(t)
}

func TestResetRevokesAllSessionsAndSecretsAreHashed(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.admin()
	student, id := h.activeStudent(admin, "reset@ogrenci.example")
	second := h.browser()
	login := second.login("reset@ogrenci.example", testPassword).want(t, http.StatusOK)
	var passwordHash string
	if err := h.pool.QueryRow(context.Background(), "SELECT password_hash FROM accounts WHERE id=$1", id).Scan(&passwordHash); err != nil {
		t.Fatal(err)
	}
	if passwordHash == testPassword || !strings.HasPrefix(passwordHash, "$argon2id$") {
		t.Fatal("password must be stored as an Argon2id hash")
	}
	var count int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM sessions WHERE account_id=$1 AND octet_length(token_hash)=32", id).Scan(&count); err != nil || count != 2 {
		t.Fatal("expected two independently hashed sessions")
	}
	for _, raw := range login.Header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(raw)
		if err != nil || !cookie.HttpOnly || cookie.Value == "" {
			continue
		}
		hash := sha256.Sum256([]byte(cookie.Value))
		if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM sessions WHERE account_id=$1 AND token_hash=$2", id, hash[:]).Scan(&count); err != nil || count != 1 {
			t.Fatal("session material is not stored as SHA-256 of opaque cookie")
		}
	}
	anonymous := h.browser()
	anonymous.post("/api/auth/forgot-password", map[string]string{"email": "reset@ogrenci.example"}).want(t, http.StatusAccepted)
	token := h.mailer.token(t, "reset@ogrenci.example", "reset-password")
	hash := sha256.Sum256([]byte(token))
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM account_tokens WHERE account_id=$1 AND purpose='reset_password' AND token_hash=$2", id, hash[:]).Scan(&count); err != nil || count != 1 {
		t.Fatal("reset token is not stored as SHA-256 of opaque token")
	}
	anonymous.post("/api/auth/reset-password", map[string]string{"token": token, "password": changedPassword}).want(t, http.StatusOK)
	if student.session() != nil || second.session() != nil {
		t.Fatal("password reset did not revoke every existing session")
	}
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM sessions WHERE account_id=$1", id).Scan(&count); err != nil || count != 0 {
		t.Fatal("password reset left sessions in database")
	}
	anonymous.login("reset@ogrenci.example", testPassword).want(t, http.StatusUnauthorized)
	anonymous.login("reset@ogrenci.example", changedPassword).want(t, http.StatusOK)
}

func TestSessionExpiryLogoutAndLiveAuthorization(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.admin()
	student, id := h.activeStudent(admin, "lifecycle@ogrenci.example")
	if _, err := h.pool.Exec(context.Background(), "UPDATE sessions SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE account_id=$1", id); err != nil {
		t.Fatal(err)
	}
	if student.session() != nil {
		t.Fatal("expired session remained authenticated")
	}
	student.login("lifecycle@ogrenci.example", testPassword).want(t, http.StatusOK)
	student.post("/api/auth/logout", map[string]any{}).want(t, http.StatusOK)
	if student.session() != nil {
		t.Fatal("logout left authenticated session")
	}
	var count int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM sessions WHERE account_id=$1 AND expires_at>now()", id).Scan(&count); err != nil || count != 0 {
		t.Fatal("logout left live server session")
	}
	student.login("lifecycle@ogrenci.example", testPassword).want(t, http.StatusOK)
	admin.post("/api/admin/accounts/"+id+"/role", map[string]string{"role": "admin", "current_password": testPassword}).want(t, http.StatusOK)
	if student.session() != nil {
		t.Fatal("role promotion did not revoke the previous session")
	}
	student.login("lifecycle@ogrenci.example", testPassword).want(t, http.StatusOK)
	if user := student.session(); user == nil || user.Role != "admin" {
		t.Fatal("role change not reflected after fresh login")
	}
	student.request(http.MethodGet, "/api/admin/accounts", nil, "", "").want(t, http.StatusOK)
	admin.post("/api/admin/accounts/"+id+"/role", map[string]string{"role": "student", "current_password": testPassword}).want(t, http.StatusOK)
	student.request(http.MethodGet, "/api/admin/accounts", nil, "", "").want(t, http.StatusUnauthorized)
	if student.session() != nil {
		t.Fatal("demoted administrator retained authenticated session")
	}
	student.login("lifecycle@ogrenci.example", testPassword).want(t, http.StatusOK)
	student.request(http.MethodGet, "/api/admin/accounts", nil, "", "").want(t, http.StatusForbidden)
	// Account moderation reviews pending applications. Simulate an operational
	// suspension directly to prove every request rechecks persisted status.
	if _, err := h.pool.Exec(context.Background(), "UPDATE accounts SET status='disabled' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	if student.session() != nil {
		t.Fatal("disabled account retained authenticated session")
	}
	student.login("lifecycle@ogrenci.example", testPassword).denied(t)
}

func TestAdminSensitiveActionsRequirePasswordAndProtectLastAdmin(t *testing.T) {
	h := newHarness(t)
	admin, adminID := h.admin()
	student, id := h.activeStudent(admin, "admin-actions@ogrenci.example")
	_ = student
	for _, password := range []string{"", "Incorrect long password!"} {
		admin.post("/api/admin/accounts/"+id+"/review", map[string]string{"decision": "rejected", "current_password": password}).denied(t)
		admin.post("/api/admin/accounts/"+id+"/role", map[string]string{"role": "admin", "current_password": password}).denied(t)
	}
	admin.post("/api/admin/accounts/"+adminID+"/role", map[string]string{"role": "student", "current_password": testPassword}).denied(t)
	admin.review(adminID, "rejected").denied(t)
	if user := admin.session(); user == nil || user.Role != "admin" {
		t.Fatal("last administrator became inaccessible")
	}
	admin.request(http.MethodGet, "/api/admin/accounts", nil, "", "").want(t, http.StatusOK)
	admin.post("/api/admin/accounts/"+id+"/role", map[string]string{"role": "owner", "current_password": testPassword}).denied(t)
	admin.review(id, "active").denied(t)
}

func TestConcurrentPasswordResetHasOneWinner(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.admin()
	student, id := h.activeStudent(admin, "concurrent-reset@ogrenci.example")
	first, second := h.browser(), h.browser()
	first.post("/api/auth/forgot-password", map[string]string{"email": "concurrent-reset@ogrenci.example"}).want(t, http.StatusAccepted)
	token := h.mailer.token(t, "concurrent-reset@ogrenci.example", "reset-password")
	start := make(chan struct{})
	results := make(chan testResponse, 2)
	for _, b := range []*browser{first, second} {
		go func(b *browser) {
			<-start
			results <- b.post("/api/auth/reset-password", map[string]string{"token": token, "password": changedPassword})
		}(b)
	}
	close(start)
	winners := 0
	for range 2 {
		r := <-results
		if r.Status == http.StatusOK {
			winners++
		} else {
			r.denied(t)
		}
	}
	if winners != 1 {
		t.Fatalf("one-use reset token accepted by %d concurrent requests", winners)
	}
	if student.session() != nil {
		t.Fatal("concurrent reset left old session authenticated")
	}
	var remaining int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM sessions WHERE account_id=$1", id).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatal("concurrent reset left old server session")
	}
	first.login("concurrent-reset@ogrenci.example", changedPassword).want(t, http.StatusOK)
}

func TestConcurrentDemotionsCannotRemoveAllAdministrators(t *testing.T) {
	h := newHarness(t)
	first, firstID := h.admin()
	second, secondID := h.activeStudent(first, "second-admin@ogrenci.example")
	first.post("/api/admin/accounts/"+secondID+"/role", map[string]string{"role": "admin", "current_password": testPassword}).want(t, http.StatusOK)
	second.login("second-admin@ogrenci.example", testPassword).want(t, http.StatusOK)
	start := make(chan struct{})
	results := make(chan testResponse, 2)
	for _, target := range []struct {
		b  *browser
		id string
	}{{first, firstID}, {second, secondID}} {
		go func(b *browser, id string) {
			<-start
			results <- b.post("/api/admin/accounts/"+id+"/role", map[string]string{"role": "student", "current_password": testPassword})
		}(target.b, target.id)
	}
	close(start)
	successes := 0
	for range 2 {
		r := <-results
		if r.Status == http.StatusOK {
			successes++
		} else {
			r.denied(t)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one safe demotion, got %d successful requests", successes)
	}
	var remaining int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM accounts WHERE role='admin' AND status='active' AND eligibility_status='approved' AND email_verified_at IS NOT NULL").Scan(&remaining); err != nil || remaining != 1 {
		t.Fatal("concurrent demotions did not preserve one accessible administrator")
	}
}

func TestClosedRegistrationDoesNotCreateAccount(t *testing.T) {
	h := newHarness(t, func(c *config.Config) { c.RegistrationOpen = false })
	b := h.browser()
	b.post("/api/auth/register", map[string]string{"name": "Synthetic student", "email": "closed@ogrenci.example", "password": testPassword}).want(t, http.StatusForbidden)
	var count int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM accounts").Scan(&count); err != nil || count != 0 {
		t.Fatal("closed registration created an account")
	}
	if h.mailer.count() != 0 {
		t.Fatal("closed registration sent a verification email")
	}
}

func TestDeliveryFailuresDoNotRevealExistingAccounts(t *testing.T) {
	h := newHarness(t)
	h.mailer.fail = true
	b := h.browser()
	registration := map[string]string{"name": "Synthetic student", "email": "mail-failure@ogrenci.example", "password": testPassword}
	created := b.post("/api/auth/register", registration).want(t, http.StatusAccepted)
	duplicate := b.post("/api/auth/register", registration).want(t, http.StatusAccepted)
	if !bytes.Equal(created.Body, duplicate.Body) {
		t.Fatal("mail delivery failure reveals whether registration created an account")
	}
	id := h.accountID("mail-failure@ogrenci.example")
	for _, path := range []string{"/api/auth/resend-verification", "/api/auth/forgot-password"} {
		known := b.post(path, map[string]string{"email": "mail-failure@ogrenci.example"}).want(t, http.StatusAccepted)
		unknown := b.post(path, map[string]string{"email": "unknown@ogrenci.example"}).want(t, http.StatusAccepted)
		if !bytes.Equal(known.Body, unknown.Body) {
			t.Fatal("mail delivery failure reveals an existing account")
		}
	}
	var count int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM account_tokens WHERE account_id=$1 AND consumed_at IS NULL", id).Scan(&count); err != nil || count != 2 {
		t.Fatal("delivery failure did not preserve replaceable verification and reset tokens")
	}
}

const operatorUsername = "SyntheticOperator"
const operatorPassword = "246813579"

func (b *browser) operatorLogin(username, password string) testResponse {
	r := b.post("/api/auth/admin-login", map[string]string{"username": username, "password": password})
	if r.Status == http.StatusOK {
		b.session()
	}
	return r
}

func (h *authHarness) operator() (*browser, string) {
	h.t.Helper()
	if err := h.service.BootstrapOperator(context.Background(), "Synthetic operator", operatorUsername, operatorPassword); err != nil {
		h.t.Fatal(err)
	}
	var id string
	if err := h.pool.QueryRow(context.Background(), "SELECT id::text FROM accounts WHERE username=$1", operatorUsername).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	b := h.browser()
	b.operatorLogin(operatorUsername, operatorPassword).want(h.t, http.StatusOK)
	return b, id
}

func TestOperatorBootstrapLoginAndSessionLifecycle(t *testing.T) {
	h := newHarness(t)
	b, id := h.operator()
	u := b.session()
	if u == nil || u.ID != id || u.Username != operatorUsername || u.Email != "" || u.EmailVerified || u.Role != "admin" || u.Status != "active" || u.EligibilityStatus != "approved" {
		t.Fatal("operator session did not expose its canonical email-free administrator identity")
	}
	var emailAbsent, verifiedAbsent bool
	var passwordHash string
	if err := h.pool.QueryRow(context.Background(), "SELECT email IS NULL,email_verified_at IS NULL,password_hash FROM accounts WHERE id=$1", id).Scan(&emailAbsent, &verifiedAbsent, &passwordHash); err != nil {
		t.Fatal(err)
	}
	if !emailAbsent || !verifiedAbsent || !strings.HasPrefix(passwordHash, "$argon2id$") || passwordHash == operatorPassword {
		t.Fatal("operator bootstrap must store no mailbox and an Argon2id password hash")
	}
	var tokenCount int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM account_tokens WHERE account_id=$1", id).Scan(&tokenCount); err != nil || tokenCount != 0 || h.mailer.count() != 0 {
		t.Fatal("operator bootstrap must not issue mailbox tokens or send mail")
	}
	listed := b.request(http.MethodGet, "/api/admin/accounts", nil, "", "").want(t, http.StatusOK)
	if !bytes.Contains(listed.Body, []byte(operatorUsername)) || bytes.Contains(listed.Body, []byte("password_hash")) || bytes.Contains(listed.Body, []byte(operatorPassword)) {
		t.Fatal("operator account list must expose its username without authentication secrets")
	}
	b.post("/api/auth/logout", map[string]any{}).want(t, http.StatusOK)
	if b.session() != nil {
		t.Fatal("operator logout retained authentication")
	}
	b.operatorLogin(strings.ToUpper(operatorUsername), operatorPassword).want(t, http.StatusOK)
	if _, err := h.pool.Exec(context.Background(), "UPDATE sessions SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE account_id=$1", id); err != nil {
		t.Fatal(err)
	}
	if b.session() != nil {
		t.Fatal("expired operator session remained authenticated")
	}
	b.operatorLogin(operatorUsername, operatorPassword).want(t, http.StatusOK)
	if _, err := h.pool.Exec(context.Background(), "UPDATE accounts SET status='disabled' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	if b.session() != nil {
		t.Fatal("disabled operator retained its session")
	}
	b.operatorLogin(operatorUsername, operatorPassword).want(t, http.StatusUnauthorized)
}

func TestOperatorLoginIsolationCSRFAndGenericFailures(t *testing.T) {
	h := newHarness(t)
	_, _ = h.operator()
	b := h.browser()
	wrong := b.operatorLogin(operatorUsername, "Wrong operator password").want(t, http.StatusUnauthorized)
	unknown := b.operatorLogin("UnknownSyntheticOperator", operatorPassword).want(t, http.StatusUnauthorized)
	if !bytes.Equal(wrong.Body, unknown.Body) {
		t.Fatal("operator login reveals username existence")
	}
	b.login(operatorUsername, operatorPassword).denied(t)
	b.post("/api/auth/login", map[string]string{"username": operatorUsername, "password": operatorPassword}).want(t, http.StatusBadRequest)
	for _, tc := range []struct{ name, csrf, origin string }{
		{"missing token", "", testOrigin},
		{"wrong token", "wrong-token", testOrigin},
		{"missing origin", b.csrf, ""},
		{"untrusted origin", b.csrf, "https://attacker.example"},
		{"null origin", b.csrf, "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b.request(http.MethodPost, "/api/auth/admin-login", map[string]string{"username": operatorUsername, "password": operatorPassword}, tc.csrf, tc.origin).want(t, http.StatusForbidden)
		})
	}
	if b.session() != nil {
		t.Fatal("rejected operator login created an authenticated session")
	}
	b.register("operator-isolation@ogrenci.example")
	b.operatorLogin("operator-isolation@ogrenci.example", testPassword).denied(t)
	if b.session() != nil {
		t.Fatal("email credentials authenticated through the operator endpoint")
	}
}

func TestOperatorLoginRateLimitNormalizesUsername(t *testing.T) {
	h := newHarness(t)
	b := h.browser()
	for i := 0; i < 12; i++ {
		username := "UnknownSyntheticOperator"
		if i%2 != 0 {
			username = strings.ToLower(username)
		}
		b.operatorLogin(username, operatorPassword).want(t, http.StatusUnauthorized)
	}
	r := b.operatorLogin("UNKNOWNSYNTHETICOPERATOR", operatorPassword).want(t, http.StatusTooManyRequests)
	if r.Header.Get("Retry-After") == "" {
		t.Fatal("operator rate limit did not specify a retry delay")
	}
}

func TestOperatorBootstrapCannotReplaceAccountsOrRelaxNormalPasswordPolicy(t *testing.T) {
	h := newHarness(t)
	_, id := h.operator()
	var originalHash string
	if err := h.pool.QueryRow(context.Background(), "SELECT password_hash FROM accounts WHERE id=$1", id).Scan(&originalHash); err != nil {
		t.Fatal(err)
	}
	if err := h.service.BootstrapOperator(context.Background(), "Replacement name", strings.ToLower(operatorUsername), changedPassword); err == nil {
		t.Fatal("case-insensitive duplicate operator bootstrap succeeded")
	}
	var hash, name string
	if err := h.pool.QueryRow(context.Background(), "SELECT name,password_hash FROM accounts WHERE id=$1", id).Scan(&name, &hash); err != nil || hash != originalHash || name != "Synthetic operator" {
		t.Fatal("duplicate bootstrap mutated an existing operator")
	}
	for _, username := range []string{"ab", "3operator", "bad-user", "has space", "school@ogrenci.example", strings.Repeat("a", 33)} {
		if err := h.service.BootstrapOperator(context.Background(), "Synthetic operator", username, operatorPassword); err == nil {
			t.Fatalf("bootstrap accepted invalid username %q", username)
		}
	}
	if err := h.service.BootstrapOperator(context.Background(), "Synthetic operator", "ShortPasswordOperator", "1234567"); err == nil {
		t.Fatal("bootstrap accepted an operator password shorter than eight characters")
	}
	if _, err := auth.HashPassword(operatorPassword); err == nil {
		t.Fatal("operator password exception weakened regular account password policy")
	}
	if err := h.service.BootstrapAdmin(context.Background(), "Synthetic email admin", "short-admin@ogrenci.example", operatorPassword); err == nil {
		t.Fatal("operator password exception weakened email administrator bootstrap")
	}
	b := h.browser()
	studentID := b.register("bootstrap-isolation@ogrenci.example")
	if err := h.service.BootstrapOperator(context.Background(), "Replacement", "bootstrap-isolation@ogrenci.example", operatorPassword); err == nil {
		t.Fatal("operator bootstrap accepted a student's email identity")
	}
	var role string
	if err := h.pool.QueryRow(context.Background(), "SELECT role FROM accounts WHERE id=$1", studentID).Scan(&role); err != nil || role != "student" {
		t.Fatal("operator bootstrap promoted an existing student")
	}
}

func TestOperatorIdentityConstraintsAndEmailTokenIsolation(t *testing.T) {
	h := newHarness(t)
	b, id := h.operator()
	for _, statement := range []string{
		"UPDATE accounts SET role='student' WHERE id=$1",
		"UPDATE accounts SET email='injected@ogrenci.example' WHERE id=$1",
		"UPDATE accounts SET email_verified_at=now() WHERE id=$1",
		"UPDATE accounts SET eligibility_status='pending' WHERE id=$1",
		"UPDATE accounts SET status='pending' WHERE id=$1",
	} {
		if _, err := h.pool.Exec(context.Background(), statement, id); err == nil {
			t.Fatalf("database allowed invalid operator identity: %s", statement)
		}
	}
	before := h.mailer.count()
	for _, path := range []string{"/api/auth/forgot-password", "/api/auth/resend-verification"} {
		known := b.post(path, map[string]string{"email": operatorUsername})
		unknown := b.post(path, map[string]string{"email": "UnknownSyntheticOperator"})
		if known.Status != unknown.Status || !bytes.Equal(known.Body, unknown.Body) {
			t.Fatal("email recovery reveals operator existence")
		}
		known.want(t, http.StatusAccepted)
	}
	if h.mailer.count() != before {
		t.Fatal("email recovery sent a message for an email-free operator")
	}
	var originalHash string
	if err := h.pool.QueryRow(context.Background(), "SELECT password_hash FROM accounts WHERE id=$1", id).Scan(&originalHash); err != nil {
		t.Fatal(err)
	}
	for i, purpose := range []string{"verify_email", "reset_password"} {
		raw := bytes.Repeat([]byte{byte(i + 1)}, 32)
		token := base64.RawURLEncoding.EncodeToString(raw)
		hash := sha256.Sum256([]byte(token))
		if _, err := h.pool.Exec(context.Background(), "INSERT INTO account_tokens(token_hash,account_id,purpose,expires_at) VALUES($1,$2,$3,now()+interval '1 hour')", hash[:], id, purpose); err != nil {
			t.Fatal(err)
		}
		if purpose == "verify_email" {
			b.post("/api/auth/verify-email", map[string]string{"token": token}).want(t, http.StatusBadRequest)
		} else {
			b.post("/api/auth/reset-password", map[string]string{"token": token, "password": changedPassword}).want(t, http.StatusBadRequest)
		}
	}
	var hash string
	var verified bool
	if err := h.pool.QueryRow(context.Background(), "SELECT password_hash,email_verified_at IS NOT NULL FROM accounts WHERE id=$1", id).Scan(&hash, &verified); err != nil || hash != originalHash || verified {
		t.Fatal("injected email-purpose token changed an operator identity")
	}
	if b.session() == nil {
		t.Fatal("rejected operator recovery token revoked a valid session")
	}
}

func TestOperatorRemainsAdminAndCountsTowardLastAdminProtection(t *testing.T) {
	h := newHarness(t)
	emailAdmin, emailID := h.admin()
	operator, operatorID := h.operator()
	operator.post("/api/admin/accounts/"+operatorID+"/role", map[string]string{"role": "student", "current_password": operatorPassword}).want(t, http.StatusConflict)
	emailAdmin.post("/api/admin/accounts/"+operatorID+"/role", map[string]string{"role": "student", "current_password": testPassword}).want(t, http.StatusConflict)
	if user := operator.session(); user == nil || user.Role != "admin" {
		t.Fatal("rejected operator demotion changed its authority")
	}
	student := h.browser()
	studentID := student.register("operator-reviewed@ogrenci.example")
	student.verify("operator-reviewed@ogrenci.example")
	operator.post("/api/admin/accounts/"+studentID+"/review", map[string]string{"decision": "approved", "current_password": operatorPassword}).want(t, http.StatusOK)
	student.login("operator-reviewed@ogrenci.example", testPassword).want(t, http.StatusOK)
	if user := student.session(); user == nil || user.Username != "" {
		t.Fatal("ordinary student session gained an operator username")
	}
	emailAdmin.post("/api/admin/accounts/"+emailID+"/role", map[string]string{"role": "student", "current_password": testPassword}).want(t, http.StatusOK)
	if emailAdmin.session() != nil {
		t.Fatal("email administrator demotion did not revoke its session")
	}
	operator.post("/api/admin/accounts/"+emailID+"/role", map[string]string{"role": "admin", "current_password": operatorPassword}).want(t, http.StatusOK)
	emailAdmin.login("bootstrap@ogrenci.example", testPassword).want(t, http.StatusOK)
	if _, err := h.pool.Exec(context.Background(), "UPDATE accounts SET status='disabled' WHERE id=$1", operatorID); err != nil {
		t.Fatal(err)
	}
	emailAdmin.post("/api/admin/accounts/"+emailID+"/role", map[string]string{"role": "student", "current_password": testPassword}).want(t, http.StatusConflict)
	if user := emailAdmin.session(); user == nil || user.Role != "admin" {
		t.Fatal("disabled operator incorrectly counted as an accessible administrator")
	}
}
