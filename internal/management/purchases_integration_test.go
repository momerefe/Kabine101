//go:build integration

package management_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/momerefe/Kabine101/internal/config"
)

// These checks use the existing isolated-schema harness. They are intended for
// CI with TEST_DATABASE_URL, not a running development database or API.
type savingsDocument struct {
	Month         string `json:"month"`
	Currency      string `json:"currency"`
	TotalSavings  string `json:"total_savings"`
	PurchaseCount int    `json:"purchase_count"`
	Businesses    []struct {
		BusinessID    string `json:"business_id"`
		BusinessName  string `json:"business_name"`
		PurchaseCount int    `json:"purchase_count"`
		SavingsAmount string `json:"savings_amount"`
	} `json:"businesses"`
}

func purchaseExec(t *testing.T, h *managementHarness, query string, args ...any) {
	t.Helper()
	if _, err := h.pool.Exec(context.Background(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func purchaseDay(t *testing.T, h *managementHarness) string {
	t.Helper()
	var day string
	if err := h.pool.QueryRow(context.Background(), "SELECT (now() AT TIME ZONE 'Europe/Istanbul')::date::text").Scan(&day); err != nil {
		t.Fatal(err)
	}
	return day
}

func purchaseFixture(t *testing.T, h *managementHarness, admin *browser, name string) (string, string) {
	t.Helper()
	business, benefit := h.uuid(), h.uuid()
	body := businessBody(0)
	body["name"], body["notes"], body["contact_info"] = name, "PRIVATE-BUSINESS-NOTES", "PRIVATE-CONTACT"
	admin.send("PUT", "/api/admin/businesses/"+business, body).want(t, 200)
	body = benefitBody(business, 0)
	body["title"], body["code"] = "Purchase snapshot benefit", "PRIVATE-REDEMPTION-CODE"
	rev := revision(admin.send("PUT", "/api/admin/benefits/"+benefit, body).want(t, 200), "benefit")
	admin.send("POST", "/api/admin/benefits/"+benefit+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200)
	return business, benefit
}

func purchaseBody(h *managementHarness, email, benefit, date, original, paid, reference string) map[string]any {
	return map[string]any{"id": h.uuid(), "student_email": email, "benefit_id": benefit, "purchase_date": date, "original_amount": original, "paid_amount": paid, "reference": reference}
}

func recordPurchase(t *testing.T, h *managementHarness, admin *browser, email, benefit, date, original, paid, reference string) string {
	t.Helper()
	body := purchaseBody(h, email, benefit, date, original, paid, reference)
	r := admin.send("POST", "/api/admin/purchases", body).want(t, 201)
	p := entity(r, "purchase")
	if p["id"] != body["id"] || p["status"] != "pending" || p["currency"] != "TRY" {
		t.Fatalf("new purchase contract: %s", r.Body)
	}
	return body["id"].(string)
}

func reviewPurchase(t *testing.T, admin *browser, id, decision string) response {
	t.Helper()
	r := admin.send("POST", "/api/admin/purchases/"+id+"/review", map[string]any{"decision": decision, "reason": "PRIVATE-REVIEW-REASON"}).want(t, 200)
	if entity(r, "purchase")["status"] != decision {
		t.Fatalf("review did not persist %s: %s", decision, r.Body)
	}
	return r
}

func purchaseList(t *testing.T, b *browser, path string) ([]map[string]any, response) {
	t.Helper()
	r := b.send("GET", path, nil).want(t, 200)
	var document struct {
		Purchases []map[string]any `json:"purchases"`
		Truncated *bool            `json:"truncated"`
	}
	if err := json.Unmarshal(r.Body, &document); err != nil {
		t.Fatal(err)
	}
	if document.Purchases == nil || document.Truncated == nil {
		t.Fatalf("purchase list must have non-null purchases and truncated: %s", r.Body)
	}
	if *document.Truncated {
		t.Fatalf("small fixture unexpectedly truncated: %s", r.Body)
	}
	return document.Purchases, r
}

func readSavings(t *testing.T, b *browser) savingsDocument {
	t.Helper()
	r := b.send("GET", "/api/student/savings", nil).want(t, 200)
	var d savingsDocument
	if err := json.Unmarshal(r.Body, &d); err != nil {
		t.Fatal(err)
	}
	if d.Currency != "TRY" || d.Businesses == nil || len(d.Month) != 7 {
		t.Fatalf("savings contract: %s", r.Body)
	}
	var projection map[string]json.RawMessage
	if err := json.Unmarshal(r.Body, &projection); err != nil {
		t.Fatal(err)
	}
	if len(projection) != 5 {
		t.Fatalf("unexpected savings fields: %s", r.Body)
	}
	var businesses []map[string]json.RawMessage
	if err := json.Unmarshal(projection["businesses"], &businesses); err != nil {
		t.Fatal(err)
	}
	for _, business := range businesses {
		if len(business) != 4 || business["business_id"] == nil || business["business_name"] == nil || business["purchase_count"] == nil || business["savings_amount"] == nil {
			t.Fatalf("unexpected savings business projection: %s", r.Body)
		}
	}
	purchaseNoSecrets(t, r)
	return d
}

func wantSavings(t *testing.T, b *browser, total string, count int) savingsDocument {
	t.Helper()
	d := readSavings(t, b)
	if d.TotalSavings != total || d.PurchaseCount != count {
		t.Fatalf("savings=%s/%d, want %s/%d", d.TotalSavings, d.PurchaseCount, total, count)
	}
	if count == 0 && len(d.Businesses) != 0 {
		t.Fatalf("zero savings must have no business rows: %+v", d)
	}
	return d
}

func purchaseNoSecrets(t *testing.T, r response) {
	t.Helper()
	for _, secret := range []string{"PRIVATE-", "student_email", "account_id", "student_id", "created_by", "reviewed_by", "review_reason", "reference", "contact_info", "password_hash", "eligibility_status"} {
		if strings.Contains(string(r.Body), secret) {
			t.Fatalf("student response exposed %q: %s", secret, r.Body)
		}
	}
}

func TestPurchasesAuthorizationCSRFAndInputValidation(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("purchase-admin@ogrenci.example", "admin")
	student, _ := h.account("purchase-student@ogrenci.example", "student")
	_, benefit := purchaseFixture(t, h, admin, "Purchase business")
	today := purchaseDay(t, h)
	body := purchaseBody(h, "purchase-student@ogrenci.example", benefit, today, "100.00", "80.00", "PRIVATE-REFERENCE")
	for _, b := range []*browser{h.browser(), student} {
		b.send("GET", "/api/admin/purchases", nil).denied(t)
		b.send("POST", "/api/admin/purchases", body).denied(t)
		b.send("POST", "/api/admin/purchases/"+body["id"].(string)+"/review", map[string]any{"decision": "approved"}).denied(t)
	}
	for _, path := range []string{"/api/student/savings", "/api/student/purchases"} {
		h.browser().send("GET", path, nil).denied(t)
	}
	for _, auth := range []struct{ csrf, origin string }{{"", h.origin}, {admin.csrf, "https://hostile.example"}} {
		admin.request("POST", "/api/admin/purchases", body, auth.csrf, auth.origin).want(t, 403)
	}
	for _, tc := range []struct{ key, value string }{
		{"original_amount", "0.00"}, {"original_amount", "-1.00"}, {"original_amount", "1.001"},
		{"original_amount", "NaN"}, {"original_amount", "1e2"}, {"original_amount", "99999999999999999999999.00"},
		{"original_amount", "10000000.00"},
		{"paid_amount", "-0.01"}, {"paid_amount", "100.01"}, {"paid_amount", "0.001"},
		{"purchase_date", "2026-02-30"}, {"purchase_date", dateOffset(t, today, 1)},
		{"purchase_date", today + "T00:00:00+03:00"}, {"reference", ""},
		{"benefit_id", "not-a-uuid"}, {"id", "not-a-uuid"}, {"student_email", "missing@ogrenci.example"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			bad := purchaseBody(h, "purchase-student@ogrenci.example", benefit, today, "100.00", "80.00", "PRIVATE-REFERENCE")
			bad[tc.key] = tc.value
			admin.send("POST", "/api/admin/purchases", bad).denied(t)
		})
	}
	badType := purchaseBody(h, "purchase-student@ogrenci.example", benefit, today, "100.00", "80.00", "typed-amount")
	badType["original_amount"] = 100
	admin.send("POST", "/api/admin/purchases", badType).want(t, 400)
	items, _ := purchaseList(t, admin, "/api/admin/purchases")
	if len(items) != 0 {
		t.Fatal("denied or invalid requests persisted purchases")
	}
	id := recordPurchase(t, h, admin, "purchase-student@ogrenci.example", benefit, today, "100.00", "80.00", "PRIVATE-REFERENCE")
	reviewPath := "/api/admin/purchases/" + id + "/review"
	admin.request("POST", reviewPath, map[string]any{"decision": "approved"}, "", h.origin).want(t, 403)
	admin.request("POST", reviewPath, map[string]any{"decision": "approved"}, admin.csrf, "https://hostile.example").want(t, 403)
	admin.send("POST", reviewPath, map[string]any{"decision": "unknown"}).want(t, 400)
	admin.send("POST", reviewPath, map[string]any{"decision": "rejected", "reason": "   "}).want(t, 400)
	admin.send("GET", "/api/admin/purchases?status=unknown", nil).want(t, 400)
	wantSavings(t, student, "0.00", 0)
}

func TestPurchaseStudentEligibilityIsCheckedFromDatabase(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("eligible-admin@ogrenci.example", "admin")
	student, studentID := h.account("eligible-student@ogrenci.example", "student")
	_, benefit := purchaseFixture(t, h, admin, "Eligibility business")
	today := purchaseDay(t, h)
	for _, update := range []string{"status='pending'", "status='disabled'", "status='rejected'", "eligibility_status='pending'", "eligibility_status='rejected'", "email_verified_at=NULL", "role='admin'"} {
		purchaseExec(t, h, "UPDATE accounts SET status='active',eligibility_status='approved',email_verified_at=now(),role='student' WHERE id=$1", studentID)
		purchaseExec(t, h, "UPDATE accounts SET "+update+" WHERE id=$1", studentID)
		admin.send("POST", "/api/admin/purchases", purchaseBody(h, "eligible-student@ogrenci.example", benefit, today, "50.00", "40.00", "eligibility-"+update)).denied(t)
		if update != "role='admin'" {
			student.send("GET", "/api/student/savings", nil).denied(t)
			student.send("GET", "/api/student/purchases", nil).denied(t)
		}
	}
	// Domain policy must apply even to a verified, manually approved database row.
	purchaseExec(t, h, "UPDATE accounts SET status='active',eligibility_status='approved',email_verified_at=now(),role='student',email='outsider@example.test' WHERE id=$1", studentID)
	admin.send("POST", "/api/admin/purchases", purchaseBody(h, "outsider@example.test", benefit, today, "50.00", "40.00", "wrong-school-domain")).want(t, 400)
	items, _ := purchaseList(t, admin, "/api/admin/purchases")
	if len(items) != 0 {
		t.Fatal("ineligible accounts acquired purchases")
	}
}

func TestPurchaseReviewLifecycleExactCentsIsolationAndSnapshots(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("lifecycle-admin@ogrenci.example", "admin")
	student, studentID := h.account("lifecycle-student@ogrenci.example", "student")
	other, _ := h.account("other-student@ogrenci.example", "student")
	business, benefit := purchaseFixture(t, h, admin, "Snapshot business")
	today := purchaseDay(t, h)
	first := recordPurchase(t, h, admin, "lifecycle-student@ogrenci.example", benefit, today, "0.30", "0.10", "PRIVATE-FIRST-REFERENCE")
	second := recordPurchase(t, h, admin, "lifecycle-student@ogrenci.example", benefit, today, "0.20", "0.10", "PRIVATE-SECOND-REFERENCE")
	rejected := recordPurchase(t, h, admin, "lifecycle-student@ogrenci.example", benefit, today, "999.99", "0.00", "PRIVATE-REJECTED-REFERENCE")
	wantSavings(t, student, "0.00", 0)
	if items, _ := purchaseList(t, student, "/api/student/purchases"); len(items) != 0 {
		t.Fatal("pending purchase exposed to student")
	}
	reviewPurchase(t, admin, rejected, "rejected")
	wantSavings(t, student, "0.00", 0)
	reviewPurchase(t, admin, first, "approved")
	reviewPurchase(t, admin, second, "approved")
	d := wantSavings(t, student, "0.30", 2)
	if len(d.Businesses) != 1 || d.Businesses[0].BusinessID != business || d.Businesses[0].SavingsAmount != "0.30" || d.Businesses[0].PurchaseCount != 2 {
		t.Fatalf("per-business aggregation: %+v", d.Businesses)
	}
	// Saved labels and amounts remain historical facts after catalogue changes.
	purchaseExec(t, h, "UPDATE businesses SET name='Later business name',notes='PRIVATE-LATER-NOTES' WHERE id=$1", business)
	purchaseExec(t, h, "UPDATE benefits SET title='Later benefit title',perk='99 percent',publication_status='draft' WHERE id=$1", benefit)
	items, r := purchaseList(t, student, "/api/student/purchases")
	purchaseNoSecrets(t, r)
	if len(items) != 2 {
		t.Fatalf("approved history length: %s", r.Body)
	}
	allowed := map[string]bool{"id": true, "business_name": true, "benefit_title": true, "purchase_date": true, "original_amount": true, "paid_amount": true, "savings_amount": true, "currency": true, "status": true}
	for _, item := range items {
		if len(item) != len(allowed) {
			t.Fatalf("unexpected student purchase fields: %v", item)
		}
		for key := range item {
			if !allowed[key] {
				t.Fatalf("private purchase field %q exposed", key)
			}
		}
		if item["business_name"] != "Snapshot business" || item["benefit_title"] != "Purchase snapshot benefit" || item["currency"] != "TRY" || item["status"] != "approved" || item["purchase_date"] != today {
			t.Fatalf("purchase snapshot changed: %v", item)
		}
		if item["id"] == first && (item["original_amount"] != "0.30" || item["paid_amount"] != "0.10" || item["savings_amount"] != "0.20") {
			t.Fatalf("cent arithmetic changed: %v", item)
		}
	}
	d = wantSavings(t, student, "0.30", 2)
	if d.Businesses[0].BusinessName != "Snapshot business" {
		t.Fatalf("savings lost saved business label: %+v", d.Businesses)
	}
	wantSavings(t, other, "0.00", 0)
	if items, _ := purchaseList(t, other, "/api/student/purchases?account_id="+url.QueryEscape(studentID)); len(items) != 0 {
		t.Fatal("student history can be redirected to another account")
	}
	r = other.send("GET", "/api/student/savings?account_id="+url.QueryEscape(studentID), nil).want(t, 200)
	var isolated savingsDocument
	if err := json.Unmarshal(r.Body, &isolated); err != nil || isolated.TotalSavings != "0.00" || isolated.PurchaseCount != 0 {
		t.Fatalf("student savings can be redirected to another account: %s", r.Body)
	}
	// Reversal remains possible after the student and source offer become inactive.
	purchaseExec(t, h, "UPDATE accounts SET status='disabled' WHERE id=$1", studentID)
	purchaseExec(t, h, "UPDATE businesses SET partnership_status='ended' WHERE id=$1", business)
	reviewPurchase(t, admin, first, "revoked")
	purchaseExec(t, h, "UPDATE accounts SET status='active' WHERE id=$1", studentID)
	wantSavings(t, student, "0.10", 1)
	if items, _ := purchaseList(t, student, "/api/student/purchases"); len(items) != 1 || items[0]["id"] != second {
		t.Fatal("revoked purchase remains in approved history")
	}
	rows, err := h.pool.Query(context.Background(), "SELECT from_status,to_status FROM purchase_events WHERE purchase_id=$1 ORDER BY id", first)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var transitions []string
	for rows.Next() {
		var from, to string
		if err := rows.Scan(&from, &to); err != nil {
			t.Fatal(err)
		}
		transitions = append(transitions, from+">"+to)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(transitions, ",") != ">pending,pending>approved,approved>revoked" {
		t.Fatalf("decision history was overwritten: %v", transitions)
	}
	for state, wantCount := range map[string]int{"pending": 0, "approved": 1, "rejected": 1, "revoked": 1} {
		items, _ := purchaseList(t, admin, "/api/admin/purchases?status="+state)
		if len(items) != wantCount {
			t.Fatalf("%s filter count=%d want %d", state, len(items), wantCount)
		}
		for _, item := range items {
			if item["status"] != state {
				t.Fatalf("status filter leaked another state: %v", item)
			}
		}
	}
}

func TestPurchaseDuplicateReferenceAndReviewConflicts(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("conflict-admin@ogrenci.example", "admin")
	secondAdmin, _ := h.account("conflict-second-admin@ogrenci.example", "admin")
	student, _ := h.account("conflict-student@ogrenci.example", "student")
	business, benefit := purchaseFixture(t, h, admin, "Reference business")
	_, otherBenefit := purchaseFixture(t, h, admin, "Other reference business")
	secondBenefit := h.uuid()
	p := "/api/admin/benefits/" + secondBenefit
	rev := revision(admin.send("PUT", p, benefitBody(business, 0)).want(t, 200), "benefit")
	admin.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200)
	today := purchaseDay(t, h)
	first := recordPurchase(t, h, admin, "conflict-student@ogrenci.example", benefit, today, "10.00", "8.00", "same-receipt")
	admin.send("POST", "/api/admin/purchases", purchaseBody(h, "conflict-student@ogrenci.example", secondBenefit, today, "10.00", "8.00", "same-receipt")).want(t, 409)
	admin.send("POST", "/api/admin/purchases", purchaseBody(h, "conflict-student@ogrenci.example", secondBenefit, today, "10.00", "8.00", "  SAME-RECEIPT  ")).want(t, 409)
	recordPurchase(t, h, admin, "conflict-student@ogrenci.example", otherBenefit, today, "10.00", "8.00", "same-receipt")
	admin.send("POST", "/api/admin/purchases/"+first+"/review", map[string]any{"decision": "revoked", "reason": "invalid transition"}).want(t, 409)
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, reviewer := range []*browser{admin, secondAdmin} {
		wg.Add(1)
		go func(b *browser) {
			defer wg.Done()
			statuses <- b.send("POST", "/api/admin/purchases/"+first+"/review", map[string]any{"decision": "approved", "reason": "concurrent review"}).Status
		}(reviewer)
	}
	wg.Wait()
	close(statuses)
	wins := 0
	for status := range statuses {
		if status == 200 {
			wins++
		} else if status != 409 {
			t.Fatalf("concurrent review returned HTTP %d", status)
		}
	}
	if wins == 0 {
		t.Fatal("no review succeeded")
	}
	wantSavings(t, student, "2.00", 1)
	// Repeated approval may succeed idempotently or conflict; neither may count twice.
	r := admin.send("POST", "/api/admin/purchases/"+first+"/review", map[string]any{"decision": "approved", "reason": "repeat"})
	if r.Status != 200 && r.Status != 409 {
		t.Fatalf("repeated approval HTTP %d: %s", r.Status, r.Body)
	}
	wantSavings(t, student, "2.00", 1)
	admin.send("POST", "/api/admin/purchases/"+first+"/review", map[string]any{"decision": "rejected", "reason": "invalid transition"}).want(t, 409)
	reviewPurchase(t, admin, first, "revoked")
	admin.send("POST", "/api/admin/purchases/"+first+"/review", map[string]any{"decision": "approved", "reason": "invalid transition"}).want(t, 409)
	wantSavings(t, student, "0.00", 0)
	items, _ := purchaseList(t, admin, "/api/admin/purchases?q="+url.QueryEscape("conflict-student@ogrenci.example")+"&status=revoked")
	if len(items) != 1 || items[0]["id"] != first {
		t.Fatalf("combined search/status filters: %v", items)
	}
}

func TestPurchaseSavingsUsesIstanbulPurchaseMonth(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("month-admin@ogrenci.example", "admin")
	student, _ := h.account("month-student@ogrenci.example", "student")
	_, benefit := purchaseFixture(t, h, admin, "Month business")
	today := purchaseDay(t, h)
	day, err := time.Parse("2006-01-02", today)
	if err != nil {
		t.Fatal(err)
	}
	monthStart := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
	previousMonthLast := monthStart.AddDate(0, 0, -1).Format("2006-01-02")
	for _, row := range []struct{ date, original, paid, reference string }{
		{previousMonthLast, "1000.00", "0.00", "previous-month"},
		{monthStart.Format("2006-01-02"), "0.30", "0.10", "first-day"},
		{today, "0.20", "0.10", "today"},
	} {
		id := recordPurchase(t, h, admin, "month-student@ogrenci.example", benefit, row.date, row.original, row.paid, row.reference)
		reviewPurchase(t, admin, id, "approved")
	}
	d := readSavings(t, student)
	if d.Month != today[:7] {
		if purchaseDay(t, h)[:7] != today[:7] {
			t.Skip("Istanbul month changed while arranging fixtures")
		}
		t.Fatalf("month=%q, want Istanbul month %q", d.Month, today[:7])
	}
	if d.TotalSavings != "0.30" || d.PurchaseCount != 2 {
		t.Fatalf("purchase month boundaries: %+v", d)
	}
	if items, _ := purchaseList(t, student, "/api/student/purchases"); len(items) != 3 {
		t.Fatal("approved history incorrectly limited to current month")
	}
}

func TestPurchaseCreateReplayDoesNotDuplicateFacts(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("replay-admin@ogrenci.example", "admin")
	student, _ := h.account("replay-student@ogrenci.example", "student")
	_, benefit := purchaseFixture(t, h, admin, "Replay business")
	body := purchaseBody(h, "replay-student@ogrenci.example", benefit, purchaseDay(t, h), "100.00", "80.00", "Replay receipt")
	admin.send("POST", "/api/admin/purchases", body).want(t, 201)
	admin.send("POST", "/api/admin/purchases", body).want(t, 200)
	id := body["id"].(string)
	reviewPurchase(t, admin, id, "approved")
	admin.send("POST", "/api/admin/purchases", body).want(t, 200)
	body["paid_amount"] = "70.00"
	admin.send("POST", "/api/admin/purchases", body).want(t, 409)
	items, _ := purchaseList(t, admin, "/api/admin/purchases")
	if len(items) != 1 || items[0]["status"] != "approved" {
		t.Fatalf("creation replay duplicated or reset reviewed purchase: %v", items)
	}
	wantSavings(t, student, "20.00", 1)
}

func TestPurchaseApprovalRechecksEligibilityAndOfferAvailability(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("recheck-admin@ogrenci.example", "admin")
	student, studentID := h.account("recheck-student@ogrenci.example", "student")
	business, benefit := purchaseFixture(t, h, admin, "Recheck business")
	today := purchaseDay(t, h)
	id := recordPurchase(t, h, admin, "recheck-student@ogrenci.example", benefit, today, "100.00", "80.00", "recheck-receipt")
	path := "/api/admin/purchases/" + id + "/review"
	decision := map[string]any{"decision": "approved", "reason": "receipt reviewed"}
	for _, update := range []string{"status='disabled'", "eligibility_status='pending'", "email_verified_at=NULL", "role='admin'"} {
		purchaseExec(t, h, "UPDATE accounts SET status='active',eligibility_status='approved',email_verified_at=now(),role='student' WHERE id=$1", studentID)
		purchaseExec(t, h, "UPDATE accounts SET "+update+" WHERE id=$1", studentID)
		admin.send("POST", path, decision).want(t, 400)
	}
	purchaseExec(t, h, "UPDATE accounts SET status='active',eligibility_status='approved',email_verified_at=now(),role='student' WHERE id=$1", studentID)
	for _, update := range []string{"publication_status='draft'", "publication_status='suspended'", "valid_from='" + dateOffset(t, today, 1) + "'::date", "valid_until='" + dateOffset(t, today, -1) + "'::date"} {
		purchaseExec(t, h, "UPDATE benefits SET publication_status='published',valid_from=NULL,valid_until=NULL WHERE id=$1", benefit)
		purchaseExec(t, h, "UPDATE benefits SET "+update+" WHERE id=$1", benefit)
		admin.send("POST", path, decision).want(t, 400)
		admin.send("POST", "/api/admin/purchases", purchaseBody(h, "recheck-student@ogrenci.example", benefit, today, "1.00", "0.00", "unavailable-offer")).want(t, 400)
	}
	purchaseExec(t, h, "UPDATE benefits SET publication_status='published',valid_from=NULL,valid_until=NULL WHERE id=$1", benefit)
	purchaseExec(t, h, "UPDATE businesses SET partnership_status='ended' WHERE id=$1", business)
	admin.send("POST", path, decision).want(t, 400)
	wantSavings(t, student, "0.00", 0)
	items, _ := purchaseList(t, admin, "/api/admin/purchases")
	if len(items) != 1 || items[0]["status"] != "pending" {
		t.Fatalf("failed approval mutated purchase: %v", items)
	}
	purchaseExec(t, h, "UPDATE businesses SET partnership_status='active' WHERE id=$1", business)
	reviewPurchase(t, admin, id, "approved")
	wantSavings(t, student, "20.00", 1)
}

func TestPurchaseSavingsAggregationExceedsBoundedHistory(t *testing.T) {
	h := newHarness(t)
	admin, adminID := h.account("aggregate-admin@ogrenci.example", "admin")
	student, studentID := h.account("aggregate-student@ogrenci.example", "student")
	business, benefit := purchaseFixture(t, h, admin, "Complete totals business")
	purchaseExec(t, h, `INSERT INTO purchases(id,student_id,business_id,benefit_id,business_name,benefit_title,purchase_date,original_amount,paid_amount,reference,reference_normalized,status,created_by,reviewed_by,reviewed_at)
 SELECT gen_random_uuid(),$1,$2,$3,'Complete totals business','Bulk saved benefit',(now() AT TIME ZONE 'Europe/Istanbul')::date,1,0,'bulk-'||n,'bulk-'||n,'approved',$4,$4,now() FROM generate_series(1,1001) n`, studentID, business, benefit, adminID)
	d := wantSavings(t, student, "10.01", 1001)
	if len(d.Businesses) != 1 || d.Businesses[0].PurchaseCount != 1001 || d.Businesses[0].SavingsAmount != "10.01" {
		t.Fatalf("business aggregate capped: %+v", d)
	}
	for _, request := range []struct {
		browser *browser
		path    string
	}{{student, "/api/student/purchases"}, {admin, "/api/admin/purchases"}} {
		r := request.browser.send("GET", request.path, nil).want(t, 200)
		var list struct {
			Purchases []map[string]any `json:"purchases"`
			Truncated bool             `json:"truncated"`
		}
		if err := json.Unmarshal(r.Body, &list); err != nil {
			t.Fatal(err)
		}
		if len(list.Purchases) != 1000 || !list.Truncated {
			t.Fatalf("history bound: count=%d truncated=%v", len(list.Purchases), list.Truncated)
		}
		if request.browser == student {
			purchaseNoSecrets(t, r)
		}
	}
}

func TestPurchaseAuditFailureRollsBackFactsAndReview(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("audit-purchase-admin@ogrenci.example", "admin")
	student, _ := h.account("audit-purchase-student@ogrenci.example", "student")
	_, benefit := purchaseFixture(t, h, admin, "Audit business")
	today := purchaseDay(t, h)
	id := recordPurchase(t, h, admin, "audit-purchase-student@ogrenci.example", benefit, today, "100.00", "80.00", "audit-existing")
	purchaseExec(t, h, `CREATE FUNCTION reject_purchase_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action LIKE 'purchase.%' THEN RAISE EXCEPTION 'synthetic purchase audit failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_purchase_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_purchase_audit()`)
	admin.send("POST", "/api/admin/purchases", purchaseBody(h, "audit-purchase-student@ogrenci.example", benefit, today, "100.00", "80.00", "audit-new")).want(t, 503)
	admin.send("POST", "/api/admin/purchases/"+id+"/review", map[string]any{"decision": "approved"}).want(t, 503)
	items, _ := purchaseList(t, admin, "/api/admin/purchases")
	if len(items) != 1 || items[0]["id"] != id || items[0]["status"] != "pending" {
		t.Fatalf("audit failure left partial purchase mutation: %v", items)
	}
	var events int
	if err := h.pool.QueryRow(context.Background(), "SELECT count(*) FROM purchase_events WHERE purchase_id=$1", id).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("failed approval left a decision event: %d", events)
	}
	wantSavings(t, student, "0.00", 0)
}

// This browser test is opt-in for CI and uses the production build with the
// same isolated schema/API as the integration tests above.
func TestSavingsBrowser(t *testing.T) {
	if os.Getenv("KOC_SAVINGS_BROWSER") != "1" {
		t.Skip("set KOC_SAVINGS_BROWSER=1 in CI after pnpm build; requires Python and Chrome")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "dist", "index.html")); err != nil {
		t.Fatal("build the web application before savings browser acceptance")
	}
	var proxy atomic.Pointer[httputil.ReverseProxy]
	files := http.FileServer(http.Dir(filepath.Join(root, "dist")))
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			p := proxy.Load()
			if p == nil {
				http.Error(w, "starting", http.StatusServiceUnavailable)
				return
			}
			p.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
	t.Cleanup(web.Close)
	h := newHarness(t, func(cfg *config.Config) { cfg.AppOrigin = web.URL })
	target, err := url.Parse(h.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy.Store(httputil.NewSingleHostReverseProxy(target))
	const username = "SavingsBrowserOperator"
	if err := h.service.BootstrapOperator(context.Background(), "Savings test operator", username, testPassword); err != nil {
		t.Fatal(err)
	}
	admin, _ := h.account("savings-browser-fixture@ogrenci.example", "admin")
	_, benefit := purchaseFixture(t, h, admin, "Browser savings business")
	const student = "savings-browser-student@ogrenci.example"
	h.account(student, "student")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python", filepath.Join(root, "tests", "run_savings_checks.py"))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "KOC_BROWSER_URL="+web.URL, "KOC_BROWSER_USERNAME="+username, "KOC_BROWSER_PASSWORD="+testPassword, "KOC_BROWSER_STUDENT="+student, "KOC_BROWSER_BENEFIT="+benefit, "KOC_BROWSER_PURCHASE_DAY="+purchaseDay(t, h))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("savings browser acceptance failed: %v\n%s", err, output)
	}
	t.Log(string(output))
}
