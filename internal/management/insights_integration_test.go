//go:build integration

package management_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func insightJSON(t *testing.T, r response) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func insightKeys(t *testing.T, v map[string]any, keys ...string) {
	t.Helper()
	if len(v) != len(keys) {
		t.Fatalf("unexpected response fields: %v; want %v", v, keys)
	}
	for _, key := range keys {
		if _, ok := v[key]; !ok {
			t.Errorf("missing %q in %v", key, v)
		}
	}
}

func insightCount(t *testing.T, h *managementHarness, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := h.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestManagement3EAuthorizationAndSettingsContract(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("insight-admin@ogrenci.example", "admin")
	student, _ := h.account("insight-student@ogrenci.example", "student")
	for _, b := range []*browser{h.browser(), student} {
		for _, path := range []string{"/api/admin/audit", "/api/admin/settings", "/api/admin/analytics", "/api/admin/accounts"} {
			b.send("GET", path, nil).denied(t)
		}
		b.send("PUT", "/api/admin/settings", map[string]any{"platform_name": "Forbidden", "support_contact": "", "show_featured": false, "revision": 1}).denied(t)
	}
	r := admin.send("GET", "/api/admin/settings", nil).want(t, 200)
	d := insightJSON(t, r)
	insightKeys(t, d, "settings", "school_domain")
	s := d["settings"].(map[string]any)
	insightKeys(t, s, "platform_name", "support_contact", "show_featured", "revision")
	if d["school_domain"] != "ogrenci.example" || s["revision"] != float64(1) {
		t.Fatalf("unexpected initial settings: %s", r.Body)
	}
	body := map[string]any{"platform_name": "Synthetic platform", "support_contact": "support@example.test", "show_featured": false, "revision": 1}
	admin.request("PUT", "/api/admin/settings", body, "", h.origin).want(t, 403)
	admin.request("PUT", "/api/admin/settings", body, admin.csrf, "https://foreign.example").want(t, 403)
	for _, change := range []map[string]any{{"platform_name": " "}, {"platform_name": strings.Repeat("x", 81)}, {"support_contact": "javascript:alert(1)"}, {"revision": 0}, {"school_domain": "attacker.test"}} {
		bad := map[string]any{}
		for key, value := range body {
			bad[key] = value
		}
		for key, value := range change {
			bad[key] = value
		}
		admin.send("PUT", "/api/admin/settings", bad).want(t, 400)
	}
	r = admin.send("PUT", "/api/admin/settings", body).want(t, 200)
	if entity(r, "settings")["revision"] != float64(2) {
		t.Fatalf("revision not advanced: %s", r.Body)
	}
	admin.send("PUT", "/api/admin/settings", body).want(t, 409)
	studentSettings := insightJSON(t, student.send("GET", "/api/student/settings", nil).want(t, 200))
	if studentSettings["platform_name"] != body["platform_name"] || studentSettings["support_contact"] != body["support_contact"] || studentSettings["show_featured"] != false {
		t.Fatalf("student settings did not reflect saved values: %v", studentSettings)
	}
	for _, forbidden := range []string{"revision", "school_domain"} {
		if _, ok := studentSettings[forbidden]; ok {
			t.Errorf("student settings leaked %s", forbidden)
		}
	}
}

func TestManagement3ESettingsConcurrencyAndAuditRollback(t *testing.T) {
	h := newHarness(t)
	first, _ := h.account("settings-first@ogrenci.example", "admin")
	second, _ := h.account("settings-second@ogrenci.example", "admin")
	body := map[string]any{"platform_name": "Concurrent platform", "support_contact": "", "show_featured": true, "revision": 1}
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for _, b := range []*browser{first, second} {
		wg.Add(1)
		go func(b *browser) { defer wg.Done(); results <- b.send("PUT", "/api/admin/settings", body).Status }(b)
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for status := range results {
		if status == 200 {
			wins++
		}
		if status == 409 {
			conflicts++
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("settings concurrency: wins=%d conflicts=%d", wins, conflicts)
	}
	if insightCount(t, h, "SELECT count(*) FROM audit_events WHERE action='platform.settings'") != 1 {
		t.Fatal("successful settings write did not record exactly one audit event")
	}
	before := first.send("GET", "/api/admin/settings", nil).want(t, 200)
	audits := insightCount(t, h, "SELECT count(*) FROM audit_events")
	overviewExec(t, h, `CREATE FUNCTION reject_insight_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic audit failure'; END $$; CREATE TRIGGER reject_insight_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_insight_audit()`)
	body["revision"], body["platform_name"] = 2, "Must roll back"
	first.send("PUT", "/api/admin/settings", body).want(t, 503)
	after := first.send("GET", "/api/admin/settings", nil).want(t, 200)
	if string(before.Body) != string(after.Body) || insightCount(t, h, "SELECT count(*) FROM audit_events") != audits {
		t.Fatal("failed audit left a partial settings mutation")
	}
}

func TestManagement3EAuditFilteringPaginationAndPrivacy(t *testing.T) {
	h := newHarness(t)
	admin, actor := h.account("audit-private@ogrenci.example", "admin")
	business := h.uuid()
	overviewExec(t, h, `UPDATE accounts SET name='Audit Actor' WHERE id=$1`, actor)
	overviewExec(t, h, `INSERT INTO businesses(id,name,category,notes,contact_info) VALUES($1,'Searchable subject','food','PRIVATE-NOTES','PRIVATE-CONTACT')`, business)
	// 53 events share an exact timestamp, exercising the stable id tie-breaker.
	overviewExec(t, h, `INSERT INTO audit_events(actor_id,action,entity_type,entity_id,created_at) SELECT $1,'synthetic.audit','business',$2,'2026-10-20 00:00:00+03' FROM generate_series(1,53)`, actor, business)
	overviewExec(t, h, `INSERT INTO audit_events(actor_id,action,entity_type,entity_id,created_at) VALUES($1,'synthetic.audit','business',$2,'2026-10-19 23:59:59+03'),($1,'synthetic.audit','business',$2,'2026-10-20 23:59:59+03'),($1,'synthetic.audit','business',$2,'2026-10-21 00:00:00+03')`, actor, business)
	base := "/api/admin/audit?action=synthetic.audit&from=2026-10-20&until=2026-10-20&q=" + url.QueryEscape("SEARCHABLE")
	seen := map[string]bool{}
	var lastID int64 = 1<<63 - 1
	for page, length := range []int{50, 4, 0} {
		r := admin.send("GET", base+"&page="+strconv.Itoa(page+1), nil).want(t, 200)
		d := insightJSON(t, r)
		insightKeys(t, d, "events", "page", "page_size", "total")
		events := d["events"].([]any)
		if len(events) != length || d["total"] != float64(54) || d["page_size"] != float64(50) || d["page"] != float64(page+1) {
			t.Fatalf("audit page contract: %s", r.Body)
		}
		for _, row := range events {
			e := row.(map[string]any)
			insightKeys(t, e, "id", "action", "actor_name", "entity_type", "entity_id", "entity_label", "created_at")
			id := e["id"].(string)
			if seen[id] {
				t.Fatalf("duplicate event across pages: %s", id)
			}
			seen[id] = true
			// The latest event sorts first; remaining equal timestamps sort by id.
			stamp, err := time.Parse(time.RFC3339Nano, e["created_at"].(string))
			if err != nil {
				t.Fatal(err)
			}
			if stamp.Equal(time.Date(2026, 10, 19, 21, 0, 0, 0, time.UTC)) {
				n, _ := strconv.ParseInt(id, 10, 64)
				if n >= lastID {
					t.Fatal("audit ties are not stable")
				}
				lastID = n
			}
			if e["actor_name"] != "Audit Actor" || e["entity_label"] != "Searchable subject" {
				t.Fatalf("missing safe labels: %v", e)
			}
		}
		for _, private := range []string{"PRIVATE-", "password", "audit-private@", "token", "metadata"} {
			if strings.Contains(string(r.Body), private) {
				t.Errorf("audit leaked %q", private)
			}
		}
	}
	for _, query := range []string{"page=0", "page=-1", "page=no", "from=2026-02-30", "until=no", "from=2026-10-21&until=2026-10-20"} {
		admin.send("GET", "/api/admin/audit?"+query, nil).want(t, 400)
	}
	for _, q := range []string{"Audit Actor", "synthetic.audit"} {
		d := insightJSON(t, admin.send("GET", "/api/admin/audit?action=synthetic.audit&q="+url.QueryEscape(q), nil).want(t, 200))
		if d["total"] != float64(56) {
			t.Fatalf("audit search %q omitted matches: %v", q, d)
		}
	}
	d := insightJSON(t, admin.send("GET", "/api/admin/audit?action=synthetic.nonexistent", nil).want(t, 200))
	if d["total"] != float64(0) || len(d["events"].([]any)) != 0 {
		t.Fatal("unknown action returned unrelated audit events")
	}
}

func insightBenefit(t *testing.T, h *managementHarness, admin *browser, featured bool) string {
	t.Helper()
	business, benefit := h.uuid(), h.uuid()
	admin.send("PUT", "/api/admin/businesses/"+business, businessBody(0)).want(t, 200)
	body := benefitBody(business, 0)
	body["featured"] = featured
	r := admin.send("PUT", "/api/admin/benefits/"+benefit, body).want(t, 200)
	admin.send("POST", "/api/admin/benefits/"+benefit+"/publish", map[string]any{"revision": revision(r, "benefit"), "conditions_reviewed": true}).want(t, 200)
	return benefit
}

func TestManagement3EViewsAreEligibleDailyAndNotUsage(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("views-admin@ogrenci.example", "admin")
	student, studentID := h.account("views-student@ogrenci.example", "student")
	second, _ := h.account("views-second@ogrenci.example", "student")
	id := insightBenefit(t, h, admin, true)
	path := "/api/student/benefits/" + id + "/view"
	h.browser().send("POST", path, map[string]any{}).denied(t)
	student.request("POST", path, map[string]any{}, "", h.origin).want(t, 403)
	student.request("POST", path, map[string]any{}, student.csrf, "https://foreign.example").want(t, 403)
	student.send("POST", path, map[string]any{"account_id": studentID}).want(t, 400)
	// A prior-day view must not suppress the same student's next calendar day.
	overviewExec(t, h, "INSERT INTO benefit_daily_views(account_id,benefit_id,view_date) VALUES($1,$2,(now() AT TIME ZONE 'Europe/Istanbul')::date-1)", studentID, id)
	for i := 0; i < 3; i++ {
		student.send("POST", path, map[string]any{}).want(t, 200)
	}
	second.send("POST", path, map[string]any{}).want(t, 200)
	admin.send("POST", path, map[string]any{}).want(t, 200)
	if n := insightCount(t, h, "SELECT count(*) FROM benefit_daily_views WHERE benefit_id=$1", id); n != 3 {
		t.Fatalf("daily student dedupe/admin exclusion: %d", n)
	}
	if n := insightCount(t, h, "SELECT count(*) FROM benefit_daily_views WHERE benefit_id=$1 AND view_date=(now() AT TIME ZONE 'Europe/Istanbul')::date", id); n != 2 {
		t.Fatalf("new view dates must use Istanbul: %d", n)
	}
	draft := h.uuid()
	var business string
	if err := h.pool.QueryRow(context.Background(), "SELECT business_id::text FROM benefits WHERE id=$1", id).Scan(&business); err != nil {
		t.Fatal(err)
	}
	admin.send("PUT", "/api/admin/benefits/"+draft, benefitBody(business, 0)).want(t, 200)
	student.send("POST", "/api/student/benefits/"+draft+"/view", map[string]any{}).denied(t)
	student.send("POST", "/api/student/benefits/"+h.uuid()+"/view", map[string]any{}).denied(t)
	student.send("POST", "/api/student/benefits/not-a-uuid/view", map[string]any{}).want(t, 400)
	overviewExec(t, h, "UPDATE accounts SET eligibility_status='pending' WHERE id=$1", studentID)
	student.send("POST", path, map[string]any{}).denied(t)
	d := insightJSON(t, admin.send("GET", "/api/admin/analytics", nil).want(t, 200))
	insightKeys(t, d, "period_start", "period_end", "generated_at", "totals", "benefits", "limit")
	insightKeys(t, d["totals"].(map[string]any), "views", "favorites", "benefits")
	if d["totals"].(map[string]any)["views"] != float64(3) {
		t.Fatalf("views not reflected: %v", d)
	}
}

func TestManagement3EAnalyticsWindowFavoritesAndUncappedTotals(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("analytics-admin@ogrenci.example", "admin")
	student, studentID := h.account("analytics-student@ogrenci.example", "student")
	id := insightBenefit(t, h, admin, false)
	var day string
	if err := h.pool.QueryRow(context.Background(), "SELECT (now() AT TIME ZONE 'Europe/Istanbul')::date::text").Scan(&day); err != nil {
		t.Fatal(err)
	}
	// First and last days are included. Old and future rows are outside the window.
	for _, offset := range []int{-30, -29, 0, 1} {
		overviewExec(t, h, "INSERT INTO benefit_daily_views(account_id,benefit_id,view_date) VALUES($1,$2,$3::date)", studentID, id, dateOffset(t, day, offset))
	}
	student.send("PUT", "/api/student/favorites/"+id, map[string]any{}).want(t, 200)
	overviewExec(t, h, "UPDATE favorites SET created_at='2001-01-01' WHERE benefit_id=$1", id)
	var business string
	if err := h.pool.QueryRow(context.Background(), "SELECT business_id::text FROM benefits WHERE id=$1", id).Scan(&business); err != nil {
		t.Fatal(err)
	}
	overviewExec(t, h, `INSERT INTO benefits(business_id,title,method) SELECT $1,'Analytics draft '||n,'code' FROM generate_series(1,25)n`, business)
	d := insightJSON(t, admin.send("GET", "/api/admin/analytics", nil).want(t, 200))
	if d["period_end"] != day {
		t.Skip("Istanbul midnight crossed while arranging fixtures")
	}
	if d["period_start"] != dateOffset(t, day, -29) || d["limit"] != float64(20) || len(d["benefits"].([]any)) != 20 {
		t.Fatalf("analytics window/top limit: %v", d)
	}
	if _, err := time.Parse(time.RFC3339, d["generated_at"].(string)); err != nil {
		t.Fatal("analytics timestamp missing")
	}
	totals := d["totals"].(map[string]any)
	if totals["views"] != float64(2) || totals["favorites"] != float64(1) || totals["benefits"] != float64(26) {
		t.Fatalf("totals were capped, synthesized, or mixed with date-limited favorites: %v", totals)
	}
	first := d["benefits"].([]any)[0].(map[string]any)
	insightKeys(t, first, "id", "business_name", "title", "status", "view_count", "favorite_count")
	if first["id"] != id || first["view_count"] != float64(2) || first["favorite_count"] != float64(1) {
		t.Fatalf("top benefit counts: %v", first)
	}
	student.send("DELETE", "/api/student/favorites/"+id, map[string]any{}).want(t, 200)
	d = insightJSON(t, admin.send("GET", "/api/admin/analytics", nil).want(t, 200))
	if d["totals"].(map[string]any)["favorites"] != float64(0) {
		t.Fatal("removed favorite retained as historical usage")
	}
}

func TestManagement3EAccountStatusSecurityAndRevocation(t *testing.T) {
	h := newHarness(t)
	admin, adminID := h.account("status-admin@ogrenci.example", "admin")
	student, id := h.account("status-student@ogrenci.example", "student")
	path := "/api/admin/accounts/" + id + "/status"
	body := map[string]any{"status": "disabled", "current_password": testPassword}
	student.send("POST", path, body).denied(t)
	h.browser().send("POST", path, body).denied(t)
	admin.request("POST", path, body, "", h.origin).want(t, 403)
	admin.request("POST", path, body, admin.csrf, "https://foreign.example").want(t, 403)
	for _, password := range []string{"", "wrong password"} {
		admin.send("POST", path, map[string]any{"status": "disabled", "current_password": password}).denied(t)
	}
	admin.send("POST", path, map[string]any{"status": "pending", "current_password": testPassword}).want(t, 400)
	admin.send("POST", path, body).want(t, 200)
	if n := insightCount(t, h, "SELECT count(*) FROM sessions WHERE account_id=$1", id); n != 0 {
		t.Fatalf("disabled target retained %d sessions", n)
	}
	student.send("GET", "/api/student/catalog", nil).denied(t)
	student.send("POST", "/api/auth/login", map[string]any{"email": "status-student@ogrenci.example", "password": testPassword}).denied(t)
	admin.send("POST", path, map[string]any{"status": "active", "current_password": testPassword}).want(t, 200)
	student.send("GET", "/api/student/catalog", nil).denied(t)
	student.send("POST", "/api/auth/login", map[string]any{"email": "status-student@ogrenci.example", "password": testPassword}).want(t, 200)
	student.session()
	student.send("GET", "/api/student/catalog", nil).want(t, 200)
	admin.send("POST", "/api/admin/accounts/"+adminID+"/status", body).want(t, 409)
	for i, state := range []struct{ status, eligibility string }{{"pending", "pending"}, {"rejected", "rejected"}, {"disabled", "pending"}} {
		target := h.uuid()
		overviewExec(t, h, `INSERT INTO accounts(id,name,email,password_hash,status,eligibility_status,email_verified_at) VALUES($1,'Restricted target',$2,'PRIVATE-HASH',$3,$4,now())`, target, fmt.Sprintf("restricted%d@example.test", i), state.status, state.eligibility)
		admin.send("POST", "/api/admin/accounts/"+target+"/status", map[string]any{"status": "active", "current_password": testPassword}).denied(t)
	}
}

func TestManagement3EStatusAuditRollbackAndLastAdminRace(t *testing.T) {
	h := newHarness(t)
	first, firstID := h.account("race-first@ogrenci.example", "admin")
	second, secondID := h.account("race-second@ogrenci.example", "admin")
	student, id := h.account("race-target@ogrenci.example", "student")
	beforeSessions := insightCount(t, h, "SELECT count(*) FROM sessions WHERE account_id=$1", id)
	overviewExec(t, h, `CREATE FUNCTION reject_status_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic audit failure'; END $$; CREATE TRIGGER reject_status_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_status_audit()`)
	body := map[string]any{"status": "disabled", "current_password": testPassword}
	first.send("POST", "/api/admin/accounts/"+id+"/status", body).want(t, 503)
	if insightCount(t, h, "SELECT count(*) FROM accounts WHERE id=$1 AND status='active'", id) != 1 || insightCount(t, h, "SELECT count(*) FROM sessions WHERE account_id=$1", id) != beforeSessions {
		t.Fatal("audit failure changed account or sessions")
	}
	student.send("GET", "/api/student/catalog", nil).want(t, 200)
	overviewExec(t, h, "DROP TRIGGER reject_status_audit ON audit_events")
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for _, target := range []struct {
		b  *browser
		id string
	}{{first, firstID}, {second, secondID}} {
		wg.Add(1)
		go func(b *browser, id string) {
			defer wg.Done()
			results <- b.send("POST", "/api/admin/accounts/"+id+"/status", body).Status
		}(target.b, target.id)
	}
	wg.Wait()
	close(results)
	wins := 0
	for status := range results {
		if status == 200 {
			wins++
		} else if status < 400 || status >= 500 {
			t.Fatalf("last-admin race returned %d", status)
		}
	}
	if wins != 1 || insightCount(t, h, "SELECT count(*) FROM accounts WHERE role='admin' AND status='active' AND eligibility_status='approved' AND email_verified_at IS NOT NULL") != 1 {
		t.Fatal("concurrent disable removed all administrators")
	}
}

func TestManagement3EAccountListFiltersAndTruncation(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("list-admin@ogrenci.example", "admin")
	_, studentID := h.account("list-student@ogrenci.example", "student")
	overviewExec(t, h, "UPDATE accounts SET name='Distinct List Student' WHERE id=$1", studentID)
	for _, query := range []string{"q=Distinct", "role=student&status=active&eligibility_status=approved&q=Distinct"} {
		d := insightJSON(t, admin.send("GET", "/api/admin/accounts?"+query, nil).want(t, 200))
		accounts := d["accounts"].([]any)
		if len(accounts) != 1 || accounts[0].(map[string]any)["id"] != studentID || d["truncated"] != false {
			t.Fatalf("account filters not applied: %v", d)
		}
	}
	overviewExec(t, h, `INSERT INTO accounts(name,email,password_hash,status,eligibility_status,email_verified_at) SELECT 'Bulk list student','listbulk'||n||'@example.test','PRIVATE-HASH','active','approved',now() FROM generate_series(1,1001)n`)
	r := admin.send("GET", "/api/admin/accounts?q=Bulk", nil).want(t, 200)
	d := insightJSON(t, r)
	if len(d["accounts"].([]any)) != 1000 || d["truncated"] != true {
		t.Fatalf("account list did not report truncation: %v", d["truncated"])
	}
	for _, secret := range []string{"PRIVATE-HASH", "password_hash", "token_hash"} {
		if strings.Contains(string(r.Body), secret) {
			t.Errorf("account list leaked %s", secret)
		}
	}
}

func TestManagement3EOperatorStatusPolicy(t *testing.T) {
	h := newHarness(t)
	const username = "Synthetic3EOperator"
	if err := h.service.BootstrapOperator(context.Background(), "Synthetic operator", username, testPassword); err != nil {
		t.Fatal(err)
	}
	operator := h.browser()
	operator.send("POST", "/api/auth/admin-login", map[string]any{"username": username, "password": testPassword}).want(t, 200)
	operator.session()
	var id string
	if err := h.pool.QueryRow(context.Background(), "SELECT id::text FROM accounts WHERE username=$1", username).Scan(&id); err != nil {
		t.Fatal(err)
	}
	path := "/api/admin/accounts/" + id + "/status"
	operator.send("POST", path, map[string]any{"status": "disabled", "current_password": testPassword}).want(t, 409)
	emailAdmin, _ := h.account("operator-peer@ogrenci.example", "admin")
	emailAdmin.send("POST", path, map[string]any{"status": "disabled", "current_password": testPassword}).want(t, 200)
	operator.send("GET", "/api/admin/settings", nil).denied(t)
	emailAdmin.send("POST", path, map[string]any{"status": "active", "current_password": testPassword}).want(t, 200)
	operator.send("GET", "/api/admin/settings", nil).denied(t)
	operator.send("POST", "/api/auth/admin-login", map[string]any{"username": username, "password": testPassword}).want(t, 200)
	operator.session()
	operator.send("GET", "/api/admin/settings", nil).want(t, 200)
	operator.send("POST", "/api/admin/accounts/"+id+"/role", map[string]any{"role": "student", "current_password": testPassword}).want(t, 409)
	if insightCount(t, h, "SELECT count(*) FROM accounts WHERE id=$1 AND email IS NULL AND email_verified_at IS NULL AND role='admin' AND status='active'", id) != 1 {
		t.Fatal("operator identity policy was lost")
	}
}

func TestManagement3EStatusAndRoleShareLastAdminGuard(t *testing.T) {
	h := newHarness(t)
	first, firstID := h.account("mixed-first@ogrenci.example", "admin")
	second, secondID := h.account("mixed-second@ogrenci.example", "admin")
	results := make(chan int, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		results <- first.send("POST", "/api/admin/accounts/"+firstID+"/status", map[string]any{"status": "disabled", "current_password": testPassword}).Status
	}()
	go func() {
		defer wg.Done()
		results <- second.send("POST", "/api/admin/accounts/"+secondID+"/role", map[string]any{"role": "student", "current_password": testPassword}).Status
	}()
	wg.Wait()
	close(results)
	wins := 0
	for status := range results {
		if status == 200 {
			wins++
		} else if status < 400 || status >= 500 {
			t.Fatalf("mixed guard race returned %d", status)
		}
	}
	if wins != 1 || insightCount(t, h, "SELECT count(*) FROM accounts WHERE role='admin' AND status='active' AND eligibility_status='approved' AND email_verified_at IS NOT NULL") != 1 {
		t.Fatal("status and role changes bypassed shared last-admin guard")
	}
}
