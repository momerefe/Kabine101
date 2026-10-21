//go:build integration

package management_test

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
	"sync"
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
)

const testOrigin = "http://127.0.0.1:5173"
const testPassword = "Management synthetic password 2026!"

type discardMailer struct{}

func (discardMailer) Send(context.Context, string, string, string) error { return nil }

type managementHarness struct {
	t       *testing.T
	pool    *pgxpool.Pool
	server  *httptest.Server
	origin  string
	service *auth.Service
}
type browser struct {
	h      *managementHarness
	client *http.Client
	csrf   string
}
type response struct {
	Status int
	Body   []byte
}

func newHarness(t *testing.T, overrides ...func(*config.Config)) *managementHarness {
	t.Helper()
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))
	raw := os.Getenv("TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("management integration requires TEST_DATABASE_URL naming a dedicated _test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, raw)
	if err != nil {
		t.Fatal("cannot connect to isolated test database")
	}
	schema := pgx.Identifier{fmt.Sprintf("management_test_%d", time.Now().UnixNano())}.Sanitize()
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
	server := httptest.NewServer(httpapi.Router(func(context.Context) error { return nil }, service, management.New(pool, service)))
	t.Cleanup(server.Close)
	return &managementHarness{t, pool, server, cfg.AppOrigin, service}
}
func (h *managementHarness) browser() *browser {
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
func (h *managementHarness) account(email, role string) (*browser, string) {
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
func (h *managementHarness) uuid() string {
	var id string
	if err := h.pool.QueryRow(context.Background(), "SELECT gen_random_uuid()::text").Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	return id
}
func businessBody(revision int) map[string]any {
	return map[string]any{"revision": revision, "name": "Synthetic business", "category": "food", "description": "Description", "image_path": "", "contact_person": "", "contact_info": "", "branches": "", "partnership_status": "active", "last_contact": "", "notes": ""}
}
func benefitBody(business string, revision int) map[string]any {
	return map[string]any{"revision": revision, "business_id": business, "title": "Synthetic benefit", "description": "Description", "perk": "10 percent", "conditions": []string{"Student"}, "instructions": []string{"Show code"}, "products": "", "branches": "", "exceptions": "", "minimum_spend": "", "currency": "TRY", "verification_requirement": "", "valid_from": "", "valid_until": "", "method": "code", "code": "SYNTHETIC", "target_url": "", "qr_content": "", "membership_label": "", "support_contact": "", "featured": false}
}
func entity(r response, key string) map[string]any {
	var v map[string]any
	_ = json.Unmarshal(r.Body, &v)
	e, _ := v[key].(map[string]any)
	return e
}
func revision(r response, key string) int {
	n, _ := entity(r, key)["revision"].(float64)
	return int(n)
}

func TestManagementAuthorizationAndValidation(t *testing.T) {
	h := newHarness(t)
	admin, adminID := h.account("admin@ogrenci.example", "admin")
	student, _ := h.account("student@ogrenci.example", "student")
	anon := h.browser()
	id := h.uuid()
	path := "/api/admin/businesses/" + id
	for _, b := range []*browser{anon, student} {
		for _, p := range []string{"/api/admin/businesses", "/api/admin/benefits", path, "/api/admin/benefits/" + id} {
			b.send("GET", p, nil).denied(t)
		}
		b.send("PUT", path, businessBody(0)).denied(t)
		b.send("POST", "/api/admin/benefits/"+id+"/publish", map[string]any{"revision": 1, "conditions_reviewed": true}).denied(t)
	}
	admin.request("PUT", path, businessBody(0), "", h.origin).want(t, 403)
	admin.request("PUT", path, businessBody(0), admin.csrf, "https://other.example").want(t, 403)
	bad := businessBody(0)
	bad["last_contact"] = "2026-02-30"
	admin.send("PUT", path, bad).want(t, 400)
	admin.send("PUT", path, businessBody(0)).want(t, 200)
	admin.send("PUT", path, businessBody(0)).want(t, 409)
	for _, change := range []map[string]any{{"method": "unknown"}, {"target_url": "javascript:alert(1)"}, {"valid_from": "2026-02-30"}, {"valid_from": "2026-10-23", "valid_until": "2026-10-22"}} {
		body := benefitBody(id, 0)
		for k, v := range change {
			body[k] = v
		}
		admin.send("PUT", "/api/admin/benefits/"+h.uuid(), body).want(t, 400)
	}
	if _, err := h.pool.Exec(context.Background(), "UPDATE accounts SET status='disabled' WHERE id=$1", adminID); err != nil {
		t.Fatal(err)
	}
	admin.send("GET", "/api/admin/businesses", nil).denied(t)
}

func TestManagementPublicationAndRevisionIsolation(t *testing.T) {
	h := newHarness(t)
	b, _ := h.account("admin@ogrenci.example", "admin")
	second, _ := h.account("second@ogrenci.example", "admin")
	business := h.uuid()
	bp := "/api/admin/businesses/" + business
	businessRev := revision(b.send("PUT", bp, businessBody(0)).want(t, 200), "business")
	id := h.uuid()
	p := "/api/admin/benefits/" + id
	r := b.send("PUT", p, benefitBody(business, 0)).want(t, 200)
	rev := revision(r, "benefit")
	if entity(r, "benefit")["publication_status"] != "draft" {
		t.Fatal("save must create a draft")
	}
	b.send("GET", p, nil).want(t, 200)
	b.send("PUT", p, benefitBody(business, 0)).want(t, 409)
	b.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": false}).denied(t)
	r = b.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200)
	rev = revision(r, "benefit")
	r = b.send("PUT", p, benefitBody(business, rev)).want(t, 200)
	rev = revision(r, "benefit")
	e := entity(r, "benefit")
	if e["publication_status"] != "draft" || e["reviewed_at"] != "" || e["reviewed_by"] != nil {
		t.Fatalf("editing must clear publication approval: %s", r.Body)
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, writer := range []*browser{b, second} {
		wg.Add(1)
		go func() { defer wg.Done(); statuses <- writer.send("PUT", p, benefitBody(business, rev)).Status }()
	}
	wg.Wait()
	close(statuses)
	wins, conflicts := 0, 0
	for s := range statuses {
		if s == 200 {
			wins++
		}
		if s == 409 {
			conflicts++
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("concurrent writes: wins=%d conflicts=%d", wins, conflicts)
	}
	rev = revision(b.send("GET", p, nil).want(t, 200), "benefit")
	b.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200)
	closed := businessBody(businessRev)
	closed["partnership_status"] = "ended"
	r = b.send("PUT", bp, closed).want(t, 200)
	businessRev = revision(r, "business")
	if entity(b.send("GET", p, nil).want(t, 200), "benefit")["publication_status"] != "suspended" {
		t.Fatal("closing business must suspend its published benefits")
	}
	b.send("PUT", bp, businessBody(businessRev)).want(t, 200)
	if entity(b.send("GET", p, nil).want(t, 200), "benefit")["publication_status"] != "suspended" {
		t.Fatal("reopening must not reactivate benefits")
	}
	var count int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM benefits WHERE id=$1", id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate creation count=%d err=%v", count, err)
	}
}

func TestManagementAuditFailureRollsBackMutation(t *testing.T) {
	h := newHarness(t)
	b, _ := h.account("admin@ogrenci.example", "admin")
	id := h.uuid()
	p := "/api/admin/businesses/" + id
	r := b.send("PUT", p, businessBody(0)).want(t, 200)
	rev := revision(r, "business")
	_, err := h.pool.Exec(context.Background(), `CREATE FUNCTION reject_management_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic audit failure'; END $$; CREATE TRIGGER reject_management_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_management_audit()`)
	if err != nil {
		t.Fatal(err)
	}
	body := businessBody(rev)
	body["name"] = "Must roll back"
	b.send("PUT", p, body).want(t, 503)
	r = b.send("GET", p, nil).want(t, 200)
	if revision(r, "business") != rev || entity(r, "business")["name"] != "Synthetic business" {
		t.Fatalf("audit failure left a partial business write: %s", r.Body)
	}
}

func TestManagementPublishPrerequisitesAndBusinessCloseRace(t *testing.T) {
	h := newHarness(t)
	b, _ := h.account("admin@ogrenci.example", "admin")
	second, _ := h.account("second@ogrenci.example", "admin")
	business := h.uuid()
	bp := "/api/admin/businesses/" + business
	br := revision(b.send("PUT", bp, businessBody(0)).want(t, 200), "business")
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	yesterday := time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02")
	for _, change := range []map[string]any{{"code": ""}, {"description": ""}, {"conditions": []string{}}, {"instructions": []string{}}, {"valid_until": yesterday}} {
		id := h.uuid()
		p := "/api/admin/benefits/" + id
		body := benefitBody(business, 0)
		for k, v := range change {
			body[k] = v
		}
		rev := revision(b.send("PUT", p, body).want(t, 200), "benefit")
		b.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).denied(t)
	}
	closed := businessBody(br)
	closed["partnership_status"] = "ended"
	br = revision(b.send("PUT", bp, closed).want(t, 200), "business")
	id := h.uuid()
	p := "/api/admin/benefits/" + id
	rev := revision(b.send("PUT", p, benefitBody(business, 0)).want(t, 200), "benefit")
	b.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).denied(t)
	br = revision(b.send("PUT", bp, businessBody(br)).want(t, 200), "business")
	closed = businessBody(br)
	closed["partnership_status"] = "ended"
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	wg.Add(2)
	go func() { defer wg.Done(); statuses <- b.send("PUT", bp, closed).Status }()
	go func() {
		defer wg.Done()
		statuses <- second.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).Status
	}()
	wg.Wait()
	close(statuses)
	for s := range statuses {
		if s != 200 && (s < 400 || s >= 500) {
			t.Fatalf("race returned unexpected HTTP %d", s)
		}
	}
	var invalid int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM benefits be JOIN businesses bu ON bu.id=be.business_id WHERE bu.id=$1 AND bu.partnership_status<>'active' AND be.publication_status='published'", business).Scan(&invalid); err != nil || invalid != 0 {
		t.Fatalf("close/publication race violated invariant: count=%d err=%v", invalid, err)
	}
}

func TestManagementDatesAreInclusiveAndExpiryIsReadDerived(t *testing.T) {
	h := newHarness(t)
	b, _ := h.account("admin@ogrenci.example", "admin")
	business := h.uuid()
	b.send("PUT", "/api/admin/businesses/"+business, businessBody(0)).want(t, 200)
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	for _, tc := range []struct{ from, until, want string }{{tomorrow, "", "scheduled"}, {today, today, "active"}} {
		id := h.uuid()
		p := "/api/admin/benefits/" + id
		body := benefitBody(business, 0)
		body["valid_from"] = tc.from
		body["valid_until"] = tc.until
		rev := revision(b.send("PUT", p, body).want(t, 200), "benefit")
		r := b.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200)
		if entity(r, "benefit")["status"] != tc.want {
			t.Fatalf("date bounds computed wrong state: %s", r.Body)
		}
		if _, err := h.pool.Exec(context.Background(), "UPDATE benefits SET valid_from=NULL,valid_until=$2::date WHERE id=$1", id, yesterday); err != nil {
			t.Fatal(err)
		}
		r = b.send("GET", p, nil).want(t, 200)
		if entity(r, "benefit")["status"] != "expired" || entity(r, "benefit")["publication_status"] != "published" {
			t.Fatalf("expiry must be derived at read time: %s", r.Body)
		}
	}
}
