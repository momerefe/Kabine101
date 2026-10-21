package management

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

func TestBenefitStatusIstanbulMidnight(t *testing.T) {
	date := func(s string) pgtype.Date {
		v, e := time.Parse("2006-01-02", s)
		if e != nil {
			t.Fatal(e)
		}
		return pgtype.Date{Time: v, Valid: true}
	}
	v := database.Benefit{PublicationStatus: "published", ValidFrom: date("2026-10-21"), ValidUntil: date("2026-10-21")}
	for _, tc := range []struct{ at, want string }{
		{"2026-10-20T20:59:59Z", "scheduled"},
		{"2026-10-20T21:00:00Z", "active"},
		{"2026-10-21T20:59:59Z", "active"},
		{"2026-10-21T21:00:00Z", "expired"},
	} {
		t.Run(tc.at, func(t *testing.T) {
			now, e := time.Parse(time.RFC3339, tc.at)
			if e != nil {
				t.Fatal(e)
			}
			if got := benefitStatus(v, now); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
	v.PublicationStatus = "suspended"
	if got := benefitStatus(v, time.Now()); got != "suspended" {
		t.Fatalf("suspension must override dates, got %s", got)
	}
	v.PublicationStatus = "draft"
	if got := benefitStatus(v, time.Now()); got != "draft" {
		t.Fatalf("draft must override dates, got %s", got)
	}
}

func TestOptionalAmountAndMethodValidation(t *testing.T) {
	for _, value := range []string{"0", "125.50", "9999999999.99"} {
		if !amountPattern.MatchString(value) {
			t.Errorf("valid amount rejected: %q", value)
		}
	}
	for _, value := range []string{"-1", "1.001", "1e2", "10000000000", "NaN"} {
		if amountPattern.MatchString(value) {
			t.Errorf("invalid amount accepted: %q", value)
		}
	}
	rev := int32(0)
	v := BenefitWrite{Revision: &rev, BusinessID: "00000000-0000-4000-8000-000000000001", Title: "Benefit", Method: "code", Currency: "TRY", Code: "CODE", QRContent: "hidden", MembershipLabel: "hidden", TargetURL: "https://example.com"}
	if !v.valid() {
		t.Fatal("valid draft rejected")
	}
	if v.QRContent != "" || v.MembershipLabel != "" || v.TargetURL != "" || v.Code != "CODE" {
		t.Fatal("irrelevant method payload was retained")
	}
}
