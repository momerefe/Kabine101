//go:build integration

package student_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/momerefe/Kabine101/internal/auth"
	"github.com/momerefe/Kabine101/internal/config"
	"github.com/momerefe/Kabine101/internal/httpapi"
	"github.com/momerefe/Kabine101/internal/management"
	"github.com/momerefe/Kabine101/internal/migrations"
	"github.com/momerefe/Kabine101/internal/student"
)

const testOrigin = "http://127.0.0.1:5173"
const testPassword = "Management synthetic password 2026!"

type discardMailer struct{}

func (discardMailer) Send(context.Context, string, string, string) error { return nil }

type studentHarness struct {
	t       *testing.T
	pool    *pgxpool.Pool
	server  *httptest.Server
	origin  string
	service *auth.Service
}
type browser struct {
	h      *studentHarness
	client *http.Client
	csrf   string
}
type response struct {
	Status int
	Body   []byte
}

func newHarness(t *testing.T, overrides ...func(*config.Config)) *studentHarness {
	t.Helper()
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))
	raw := os.Getenv("TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("student integration requires TEST_DATABASE_URL naming a dedicated _test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, raw)
	if err != nil {
		t.Fatal("cannot connect to isolated test database")
	}
	schema := pgx.Identifier{fmt.Sprintf("student_test_%d", time.Now().UnixNano())}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close(ctx)
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, e := admin.Exec(c, "DROP SCHEMA "+schema+" CASCADE"); e != nil {
			t.Error(e)
		}
		admin.Close(c)
	})
	q := u.Query()
	q.Set("search_path", strings.Trim(schema, "\""))
	u.RawQuery = q.Encode()
	if err = migrations.Up(u.String()); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal("cannot create isolated pool")
	}
	t.Cleanup(pool.Close)
	cfg := config.Config{Environment: "test", AppOrigin: testOrigin, SchoolDomain: "ogrenci.example", EligibilityPolicy: "manual", RegistrationOpen: true, LocalEmail: true}
	for _, override := range overrides {
		override(&cfg)
	}
	service, err := auth.New(pool, cfg, discardMailer{})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.Router(func(context.Context) error { return nil }, service, management.New(pool, service), student.New(pool, service)))
	t.Cleanup(server.Close)
	return &studentHarness{t, pool, server, cfg.AppOrigin, service}
}
func (h *studentHarness) browser() *browser {
	jar, _ := cookiejar.New(nil)
	b := &browser{h: h, client: &http.Client{Jar: jar, Timeout: 15 * time.Second}}
	b.session()
	return b
}
func (b *browser) request(method, path string, payload any, csrf, origin string) response {
	data, _ := json.Marshal(payload)
	req, err := http.NewRequest(method, b.h.server.URL+path, bytes.NewReader(data))
	if err != nil {
		b.h.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	r, err := b.client.Do(req)
	if err != nil {
		b.h.t.Fatal(err)
	}
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)
	return response{r.StatusCode, body}
}
func (b *browser) send(method, path string, payload any) response {
	return b.request(method, path, payload, b.csrf, b.h.origin)
}
func (b *browser) session() {
	r := b.send("GET", "/api/auth/session", nil).want(b.h.t, 200)
	var v struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(r.Body, &v); err != nil {
		b.h.t.Fatal(err)
	}
	b.csrf = v.CSRF
}
func (r response) want(t *testing.T, status int) response {
	t.Helper()
	if r.Status != status {
		t.Fatalf("HTTP %d want %d: %s", r.Status, status, r.Body)
	}
	return r
}
func (r response) denied(t *testing.T) {
	t.Helper()
	if r.Status < 400 || r.Status >= 500 {
		t.Fatalf("expected rejection, got %d: %s", r.Status, r.Body)
	}
}
func (h *studentHarness) account(email, role string) (*browser, string) {
	b := h.browser()
	b.send("POST", "/api/auth/register", map[string]any{"name": "Synthetic account", "email": email, "password": testPassword}).want(h.t, 202)
	var id string
	err := h.pool.QueryRow(context.Background(), "UPDATE accounts SET role=$2,status='active',eligibility_status='approved',email_verified_at=now() WHERE email=$1 RETURNING id::text", email, role).Scan(&id)
	if err != nil {
		h.t.Fatal(err)
	}
	b.send("POST", "/api/auth/login", map[string]any{"email": email, "password": testPassword}).want(h.t, 200)
	b.session()
	return b, id
}
func (h *studentHarness) uuid() string {
	var id string
	if err := h.pool.QueryRow(context.Background(), "SELECT gen_random_uuid()::text").Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}
func (h *studentHarness) exec(query string, args ...any) {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), query, args...); err != nil {
		h.t.Fatal(err)
	}
}
func (h *studentHarness) fixture(reviewer string) (string, string) {
	business, benefit := h.uuid(), h.uuid()
	h.exec(`INSERT INTO businesses(id,name,category,partnership_status,contact_person,contact_info,notes) VALUES($1,'Campus kitchen','food','active','PRIVATE PERSON','PRIVATE CONTACT','PRIVATE NOTES')`, business)
	h.exec(`INSERT INTO benefits(id,business_id,title,description,perk,conditions,instructions,method,code,target_url,qr_content,membership_label,publication_status,reviewed_at,reviewed_by,published_at) VALUES($1,$2,'Lunch reward','Fresh lunch','20 percent',ARRAY['Student'],ARRAY['Show reward'],'code','SECRET-CODE','https://secret.example/redeem','SECRET-QR','SECRET-MEMBER','published',now(),$3,now())`, benefit, business, reviewer)
	return business, benefit
}
func decode(t *testing.T, r response) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func benefits(t *testing.T, r response) []any {
	t.Helper()
	v := decode(t, r)
	list, ok := v["benefits"].([]any)
	if !ok {
		t.Fatalf("missing benefits: %s", r.Body)
	}
	return list
}
func noSecrets(t *testing.T, r response) {
	t.Helper()
	for _, secret := range []string{"SECRET-CODE", "secret.example", "SECRET-QR", "SECRET-MEMBER", "PRIVATE PERSON", "PRIVATE CONTACT", "PRIVATE NOTES", "password_hash", "reviewed_by", "suspension_reason"} {
		if strings.Contains(string(r.Body), secret) {
			t.Fatalf("private data %q leaked: %s", secret, r.Body)
		}
	}
}
func TestStudentCatalogProjectionAndAvailability(t *testing.T) {
	h := newHarness(t)
	b, id := h.account("catalog@ogrenci.example", "student")
	business, benefit := h.fixture(id)
	r := b.send("GET", "/api/student/catalog", nil).want(t, 200)
	noSecrets(t, r)
	if len(benefits(t, r)) != 1 {
		t.Fatalf("catalog: %s", r.Body)
	}
	detail := b.send("GET", "/api/student/benefits/"+benefit, nil).want(t, 200)
	noSecrets(t, detail)
	for _, key := range []string{"image_path", "products", "branches", "exceptions", "minimum_spend", "verification_requirement", "valid_from", "valid_until", "support_contact"} {
		if value, ok := decode(t, detail)["benefit"].(map[string]any)[key]; !ok || value != "" {
			t.Fatalf("optional %s must be empty string, got %#v", key, value)
		}
	}
	for _, query := range []string{"?q=missing", "?category=tech"} {
		if len(benefits(t, b.send("GET", "/api/student/catalog"+query, nil).want(t, 200))) != 0 {
			t.Fatal("filter ignored")
		}
	}
	if len(benefits(t, b.send("GET", "/api/student/catalog?q=Lunch&category=food", nil).want(t, 200))) != 1 {
		t.Fatal("matching filters lost benefit")
	}
	for _, state := range []string{"draft", "suspended"} {
		h.exec("UPDATE benefits SET publication_status=$2 WHERE id=$1", benefit, state)
		b.send("GET", "/api/student/benefits/"+benefit, nil).want(t, 404)
		if len(benefits(t, b.send("GET", "/api/student/catalog", nil).want(t, 200))) != 0 {
			t.Fatal("unpublished benefit leaked")
		}
	}
	h.exec("UPDATE benefits SET publication_status='published' WHERE id=$1", benefit)
	h.exec("UPDATE businesses SET partnership_status='ended' WHERE id=$1", business)
	b.send("GET", "/api/student/benefits/"+benefit, nil).want(t, 404)
	if len(benefits(t, b.send("GET", "/api/student/catalog", nil).want(t, 200))) != 0 {
		t.Fatal("inactive partnership leaked")
	}
}
func TestStudentFavoritesIsolationPersistenceAndCSRF(t *testing.T) {
	h := newHarness(t)
	a, aid := h.account("favorite-a@ogrenci.example", "student")
	b, bid := h.account("favorite-b@ogrenci.example", "student")
	_, id := h.fixture(aid)
	path := "/api/student/favorites/" + id
	a.request("PUT", path, nil, "", h.origin).want(t, 403)
	a.request("PUT", path, nil, a.csrf, "https://hostile.example").want(t, 403)
	for i := 0; i < 2; i++ {
		a.send("PUT", path, map[string]any{"account_id": bid}).want(t, 200)
	}
	if len(benefits(t, b.send("GET", "/api/student/favorites?account_id="+aid, nil).want(t, 200))) != 0 {
		t.Fatal("cross-account favorites leaked")
	}
	again := h.browser()
	again.send("POST", "/api/auth/login", map[string]any{"email": "favorite-a@ogrenci.example", "password": testPassword}).want(t, 200)
	again.session()
	r := again.send("GET", "/api/student/favorites", nil).want(t, 200)
	noSecrets(t, r)
	if len(benefits(t, r)) != 1 {
		t.Fatal("favorite did not persist exactly once")
	}
	h.exec("UPDATE benefits SET publication_status='draft',title='HIDDEN DRAFT TITLE' WHERE id=$1", id)
	r = a.send("GET", "/api/student/favorites", nil).want(t, 200)
	noSecrets(t, r)
	items := benefits(t, r)
	if len(items) != 1 {
		t.Fatal("unavailable favorite missing")
	}
	item := items[0].(map[string]any)
	if item["status"] != "unavailable" || item["is_favorite"] != true || strings.Contains(string(r.Body), "HIDDEN DRAFT TITLE") {
		t.Fatalf("unsafe tombstone: %s", r.Body)
	}
	a.send("PUT", path, nil).denied(t)
	for i := 0; i < 2; i++ {
		a.send("DELETE", path, nil).want(t, 200)
	}
	if len(benefits(t, a.send("GET", "/api/student/favorites", nil).want(t, 200))) != 0 {
		t.Fatal("favorite removal failed")
	}
}
func TestStudentAuthorizationRechecksDatabase(t *testing.T) {
	h := newHarness(t)
	b, id := h.account("auth-student@ogrenci.example", "student")
	_, benefit := h.fixture(id)
	paths := []string{"/api/student/catalog", "/api/student/favorites", "/api/student/settings", "/api/student/benefits/" + benefit, "/api/student/benefits/" + benefit + "/redemption"}
	anon := h.browser()
	for _, path := range paths {
		anon.send("GET", path, nil).denied(t)
	}
	for _, update := range []string{"status='pending'", "status='rejected'", "status='disabled'", "email_verified_at=NULL", "eligibility_status='pending'", "eligibility_status='rejected'"} {
		h.exec("UPDATE accounts SET status='active',eligibility_status='approved',email_verified_at=now(),role='student' WHERE id=$1", id)
		h.exec("UPDATE accounts SET "+update+" WHERE id=$1", id)
		for _, path := range paths {
			b.send("GET", path, nil).denied(t)
		}
		b.send("PUT", "/api/student/favorites/"+benefit, nil).denied(t)
	}
}
func TestStudentDatesRedemptionRechecksAndSettings(t *testing.T) {
	h := newHarness(t)
	b, aid := h.account("redemption@ogrenci.example", "student")
	business, id := h.fixture(aid)
	path := "/api/student/benefits/" + id
	for _, method := range []string{"code", "link", "qr", "membership"} {
		h.exec("UPDATE benefits SET method=$2 WHERE id=$1", id, method)
		r := b.send("GET", path+"/redemption", nil).want(t, 200)
		obj := decode(t, r)
		expected := map[string]string{"code": "code", "link": "target_url", "qr": "qr_content", "membership": "membership_label"}[method]
		if obj["method"] != method || obj[expected] == nil || obj[expected] == "" {
			t.Fatalf("missing method value: %s", r.Body)
		}
		for _, field := range []string{"code", "target_url", "qr_content", "membership_label"} {
			if field != expected {
				if _, exists := obj[field]; exists {
					t.Fatalf("unrelated redemption field %s: %s", field, r.Body)
				}
			}
		}
	}
	h.exec("UPDATE benefits SET valid_from=(now() AT TIME ZONE 'Europe/Istanbul')::date,valid_until=(now() AT TIME ZONE 'Europe/Istanbul')::date WHERE id=$1", id)
	b.send("GET", path+"/redemption", nil).want(t, 200)
	for _, state := range []struct{ sql, status string }{{"valid_from=(now() AT TIME ZONE 'Europe/Istanbul')::date+1,valid_until=NULL", "scheduled"}, {"valid_from=NULL,valid_until=(now() AT TIME ZONE 'Europe/Istanbul')::date-1", "expired"}} {
		h.exec("UPDATE benefits SET "+state.sql+" WHERE id=$1", id)
		r := b.send("GET", path, nil).want(t, 200)
		if decode(t, r)["benefit"].(map[string]any)["status"] != state.status {
			t.Fatalf("wrong date status: %s", r.Body)
		}
		b.send("GET", path+"/redemption", nil).denied(t)
		b.send("PUT", "/api/student/favorites/"+id, nil).want(t, 200)
	}
	h.exec("UPDATE benefits SET valid_from=NULL,valid_until=NULL,publication_status='suspended' WHERE id=$1", id)
	b.send("GET", path+"/redemption", nil).denied(t)
	h.exec("UPDATE benefits SET publication_status='published' WHERE id=$1", id)
	h.exec("UPDATE businesses SET partnership_status='ended' WHERE id=$1", business)
	b.send("GET", path+"/redemption", nil).denied(t)
	h.exec("UPDATE platform_settings SET platform_name='Actual campus',support_contact='help@campus.example' WHERE id=1")
	r := b.send("GET", "/api/student/settings", nil).want(t, 200)
	obj := decode(t, r)
	if obj["platform_name"] != "Actual campus" || obj["support_contact"] != "help@campus.example" {
		t.Fatalf("settings ignored persisted values: %s", r.Body)
	}
}
func TestStudentCatalogAndFavoritesBoundedAtOneThousand(t *testing.T) {
	h := newHarness(t)
	b, aid := h.account("bounded@ogrenci.example", "student")
	business, _ := h.fixture(aid)
	h.exec(`INSERT INTO benefits(business_id,title,description,perk,conditions,instructions,method,code,publication_status,reviewed_at,reviewed_by,published_at) SELECT $1,'Bulk reward '||n,'Description','Discount',ARRAY['Student'],ARRAY['Show code'],'code','SECRET-CODE','published',now(),$2,now() FROM generate_series(1,1000) n`, business, aid)
	h.exec(`INSERT INTO favorites(account_id,benefit_id) SELECT $1,id FROM benefits`, aid)
	for _, path := range []string{"/api/student/catalog", "/api/student/favorites"} {
		r := b.send("GET", path, nil).want(t, 200)
		noSecrets(t, r)
		if len(benefits(t, r)) != 1000 || decode(t, r)["truncated"] != true {
			t.Fatalf("bounded list contract violated: length=%d truncated=%v", len(benefits(t, r)), decode(t, r)["truncated"])
		}
	}
}
func TestStudentCannotReadAdministrativeDrafts(t *testing.T) {
	h := newHarness(t)
	b, aid := h.account("draft@ogrenci.example", "student")
	_, id := h.fixture(aid)
	h.exec("UPDATE benefits SET publication_status='draft' WHERE id=$1", id)
	for _, path := range []string{"/api/admin/benefits", "/api/admin/benefits/" + id, "/api/admin/businesses"} {
		b.send("GET", path, nil).denied(t)
	}
	b.send("GET", "/api/student/benefits/"+id, nil).want(t, 404)
	b.send("GET", "/api/student/benefits/"+id+"/redemption", nil).want(t, 404)
	b.send("PUT", "/api/student/favorites/"+id, nil).denied(t)
}
