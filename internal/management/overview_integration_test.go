//go:build integration

package management_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

type overviewDocument struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Businesses  map[string]int64    `json:"businesses"`
	Benefits    map[string]int64    `json:"benefits"`
	Students    map[string]int64    `json:"students"`
	Expiring    []map[string]string `json:"expiring_benefits"`
	Activity    []map[string]string `json:"recent_activity"`
}

func readOverview(t *testing.T, b *browser) (overviewDocument, []byte) {
	t.Helper()
	r := b.send("GET", "/api/admin/overview", nil).want(t, 200)
	var d overviewDocument
	if err := json.Unmarshal(r.Body, &d); err != nil {
		t.Fatal(err)
	}
	if d.GeneratedAt.IsZero() || d.Expiring == nil || d.Activity == nil {
		t.Fatalf("missing timestamp or non-null arrays: %s", r.Body)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(r.Body, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 6 {
		t.Fatalf("unexpected response fields: %s", r.Body)
	}
	return d, r.Body
}
func overviewCounts(t *testing.T, got, want map[string]int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("count fields: got %v want %v", got, want)
	}
	for key, n := range want {
		if got[key] != n {
			t.Errorf("%s: got %d want %d", key, got[key], n)
		}
	}
}
func overviewExec(t *testing.T, h *managementHarness, sql string, args ...any) {
	t.Helper()
	if _, err := h.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestOverviewAuthorizationAndEmptyContract(t *testing.T) {
	h := newHarness(t)
	h.browser().send("GET", "/api/admin/overview", nil).denied(t)
	admin, adminID := h.account("overview-admin@ogrenci.example", "admin")
	d, _ := readOverview(t, admin)
	overviewCounts(t, d.Businesses, map[string]int64{"total": 0, "active": 0, "in_progress": 0, "ended": 0})
	overviewCounts(t, d.Benefits, map[string]int64{"total": 0, "published": 0, "active": 0, "scheduled": 0, "expired": 0, "suspended": 0, "draft": 0, "ending_soon": 0})
	overviewCounts(t, d.Students, map[string]int64{"total": 0, "active": 0, "pending_review": 0, "pending_email": 0, "review_ready": 0})
	if len(d.Expiring) != 0 || len(d.Activity) != 0 {
		t.Fatal("empty database returned activity or benefits")
	}
	student, _ := h.account("overview-student@ogrenci.example", "student")
	student.send("GET", "/api/admin/overview", nil).denied(t)
	overviewExec(t, h, "UPDATE accounts SET role='student' WHERE id=$1", adminID)
	admin.send("GET", "/api/admin/overview", nil).denied(t)
}

func TestOverviewCountsAndIstanbulDateBoundaries(t *testing.T) {
	h := newHarness(t)
	admin, adminID := h.account("overview-admin@ogrenci.example", "admin")
	var day string
	if err := h.pool.QueryRow(context.Background(), "SELECT (now() AT TIME ZONE 'Europe/Istanbul')::date::text").Scan(&day); err != nil {
		t.Fatal(err)
	}
	overviewExec(t, h, `INSERT INTO businesses(name,category,partnership_status) SELECT 'Bulk '||n,'food','active' FROM generate_series(1,1001) n`)
	overviewExec(t, h, `INSERT INTO businesses(name,category,partnership_status) SELECT s,'food',s FROM unnest(ARRAY['contacted','discussing','awaiting','ended']) s`)
	business := h.uuid()
	overviewExec(t, h, `INSERT INTO businesses(id,name,category,partnership_status,notes,contact_info) VALUES($1,'Visible business','food','ended','PRIVATE-NOTES','private@example.test')`, business)
	// Every day offset is relative to the database's Istanbul day. Null dates remain active.
	type fixture struct {
		status      string
		from, until any
	}
	rows := []fixture{{"published", nil, nil}, {"published", day, day}, {"published", nil, dateOffset(t, day, 6)}, {"published", nil, dateOffset(t, day, 7)}, {"published", dateOffset(t, day, 1), nil}, {"published", nil, dateOffset(t, day, -1)}, {"draft", nil, dateOffset(t, day, -1)}, {"suspended", dateOffset(t, day, 1), nil}}
	for i, row := range rows {
		overviewExec(t, h, `INSERT INTO benefits(business_id,title,method,publication_status,valid_from,valid_until,description,perk,conditions,instructions,code,reviewed_at,reviewed_by,published_at) VALUES($1,$2,'code',$3,$4::date,$5::date,'PRIVATE-DESCRIPTION','perk',ARRAY['condition'],ARRAY['instruction'],'PRIVATE-CODE',now(),$6,now())`, business, fmt.Sprintf("Benefit %d", i), row.status, row.from, row.until, adminID)
	}
	overviewExec(t, h, `INSERT INTO benefits(business_id,title,method) SELECT $1,'Bulk draft '||n,'code' FROM generate_series(1,1001) n`, business)
	overviewExec(t, h, `INSERT INTO accounts(name,email,password_hash,status,eligibility_status,email_verified_at) SELECT 'Bulk student','bulk'||n||'@example.test','PRIVATE-HASH','active','approved',now() FROM generate_series(1,1001) n`)
	overviewExec(t, h, `INSERT INTO accounts(name,email,password_hash,status,eligibility_status,email_verified_at) VALUES
 ('Review ready','ready@example.test','PRIVATE-HASH','pending','pending',now()),
 ('Review email','pending@example.test','PRIVATE-HASH','pending','pending',NULL),
 ('Approved email','approved@example.test','PRIVATE-HASH','active','approved',NULL),
 ('Unapproved active','unapproved@example.test','PRIVATE-HASH','active','pending',now()),
 ('Rejected eligibility','eligibility-rejected@example.test','PRIVATE-HASH','active','rejected',NULL),
 ('Disabled','disabled@example.test','PRIVATE-HASH','disabled','pending',NULL),
 ('Rejected','rejected@example.test','PRIVATE-HASH','rejected','rejected',NULL)`)
	d, body := readOverview(t, admin)
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	if d.GeneratedAt.In(loc).Format("2006-01-02") != day {
		t.Skip("Istanbul midnight crossed while arranging fixtures")
	}
	overviewCounts(t, d.Businesses, map[string]int64{"total": 1006, "active": 1001, "in_progress": 3, "ended": 2})
	overviewCounts(t, d.Benefits, map[string]int64{"total": 1009, "published": 6, "active": 4, "scheduled": 1, "expired": 1, "suspended": 1, "draft": 1002, "ending_soon": 2})
	overviewCounts(t, d.Students, map[string]int64{"total": 1008, "active": 1001, "pending_review": 2, "pending_email": 2, "review_ready": 1})
	if len(d.Expiring) != 2 || d.Expiring[0]["valid_until"] != day || d.Expiring[1]["valid_until"] != dateOffset(t, day, 6) {
		t.Fatalf("inclusive expiration boundaries: %v", d.Expiring)
	}
	for _, secret := range []string{"PRIVATE-", "private@example.test", "password_hash", "contact_info", "instructions"} {
		if strings.Contains(string(body), secret) {
			t.Errorf("overview leaked %q", secret)
		}
	}
}
func dateOffset(t *testing.T, day string, offset int) string {
	t.Helper()
	d, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}
	return d.AddDate(0, 0, offset).Format("2006-01-02")
}

func TestOverviewActivityWhitelistAndPanelLimits(t *testing.T) {
	h := newHarness(t)
	admin, actor := h.account("private-actor@ogrenci.example", "admin")
	business := h.uuid()
	overviewExec(t, h, `INSERT INTO businesses(id,name,category) VALUES($1,'Business label','food')`, business)
	overviewExec(t, h, `INSERT INTO benefits(business_id,title,method,publication_status,valid_until,description,perk,conditions,instructions,code,reviewed_at,reviewed_by,published_at) SELECT $1,'Expiring '||n,'code','published',(now() AT TIME ZONE 'Europe/Istanbul')::date,'description','perk',ARRAY['condition'],ARRAY['instruction'],'PRIVATE-CODE',now(),$2,now() FROM generate_series(1,8) n`, business, actor)
	overviewExec(t, h, `DELETE FROM audit_events`)
	var expected []string
	for _, action := range []string{"business.save", "account.review.approved", "account.review.rejected", "account.role.admin", "account.role.student", "business.save", "business.save"} {
		entity, id := "account", actor
		if action == "business.save" {
			entity, id = "business", business
		}
		var eventID string
		if err := h.pool.QueryRow(context.Background(), `INSERT INTO audit_events(actor_id,action,entity_type,entity_id,created_at) VALUES($1,$2,$3,$4,'2026-01-01T00:00:00Z') RETURNING id::text`, actor, action, entity, id).Scan(&eventID); err != nil {
			t.Fatal(err)
		}
		expected = append(expected, eventID)
	}
	for _, action := range []string{"auth.login", "password.reset", "token.created", "business.save"} {
		// Even an allowed action on the wrong entity type must be excluded.
		overviewExec(t, h, `INSERT INTO audit_events(actor_id,action,entity_type,entity_id,created_at) VALUES($1,$2,'session',$1,now())`, actor, action)
	}
	d, body := readOverview(t, admin)
	if len(d.Expiring) != 5 || len(d.Activity) != 6 {
		t.Fatalf("panel limits: benefits=%d activity=%d", len(d.Expiring), len(d.Activity))
	}
	for i, item := range d.Expiring {
		if len(item) != 4 || item["business_name"] != "Business label" {
			t.Fatalf("benefit projection: %v", item)
		}
		if i > 0 && d.Expiring[i-1]["id"] >= item["id"] {
			t.Fatal("equal-date expiration rows must sort by UUID")
		}
	}
	for i, item := range d.Activity {
		if len(item) != 7 || item["id"] != expected[len(expected)-1-i] {
			t.Fatalf("activity order/projection: %v", item)
		}
		if _, err := strconv.ParseInt(item["id"], 10, 64); err != nil {
			t.Fatal(err)
		}
		if _, err := time.Parse(time.RFC3339Nano, item["created_at"]); err != nil {
			t.Fatal(err)
		}
		if item["entity_label"] == "" || item["actor_name"] == "" {
			t.Fatalf("missing safe labels: %v", item)
		}
	}
	for _, secret := range []string{"private-actor@", "PRIVATE-CODE", "auth.login", "password.reset", "token.created"} {
		if strings.Contains(string(body), secret) {
			t.Errorf("overview leaked %q", secret)
		}
	}
	var count int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM audit_events").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 11 {
		t.Fatalf("overview mutated audit history: %d", count)
	}
}
