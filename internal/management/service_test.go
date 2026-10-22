package management

import (
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

func publishableBenefit() database.Benefit {
	text := func(v string) pgtype.Text { return pgtype.Text{String: v, Valid: true} }
	return database.Benefit{Description: "Açıklama", Perk: "Yüzde 10 indirim", Conditions: []string{"Öğrenci olun"}, Instructions: []string{"Kodu gösterin"}, Method: "code", Code: text("KOD"), TargetUrl: text("https://example.com/benefit"), QrContent: text("qr-payload"), MembershipLabel: text("Öğrenci kartı")}
}

func TestPublishIssuesMissingAndMalformedRequirements(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		change func(*database.Benefit, *database.Business)
		want   []string
	}{
		{"description", func(v *database.Benefit, _ *database.Business) { v.Description = " \n" }, []string{"Açıklama alanını doldurun."}},
		{"perk", func(v *database.Benefit, _ *database.Business) { v.Perk = " " }, []string{"İndirim / ayrıcalık alanını doldurun."}},
		{"conditions", func(v *database.Benefit, _ *database.Business) { v.Conditions = nil }, []string{"En az bir koşul ekleyin."}},
		{"instructions", func(v *database.Benefit, _ *database.Business) { v.Instructions = []string{} }, []string{"En az bir kullanım adımı ekleyin."}},
		{"blank condition", func(v *database.Benefit, _ *database.Business) { v.Conditions = []string{"Öğrenci", " "} }, []string{"Koşullar boş satır içermemelidir."}},
		{"blank instruction", func(v *database.Benefit, _ *database.Business) { v.Instructions = []string{" "} }, []string{"Kullanım adımları boş satır içermemelidir."}},
		{"multiple issues", func(v *database.Benefit, _ *database.Business) { v.Perk = ""; v.Code = pgtype.Text{} }, []string{"İndirim / ayrıcalık alanını doldurun.", "İndirim kodunu girin."}},
		{"code", func(v *database.Benefit, _ *database.Business) { v.Code = pgtype.Text{} }, []string{"İndirim kodunu girin."}},
		{"link", func(v *database.Benefit, _ *database.Business) { v.Method = "link"; v.TargetUrl = pgtype.Text{} }, []string{"Hedef bağlantıyı girin."}},
		{"malformed link", func(v *database.Benefit, _ *database.Business) { v.Method = "link"; v.TargetUrl.String = "https://" }, []string{"Hedef bağlantı geçerli bir http:// veya https:// adresi olmalıdır."}},
		{"unsafe link", func(v *database.Benefit, _ *database.Business) {
			v.Method = "link"
			v.TargetUrl.String = "javascript:alert(1)"
		}, []string{"Hedef bağlantı geçerli bir http:// veya https:// adresi olmalıdır."}},
		{"qr", func(v *database.Benefit, _ *database.Business) { v.Method = "qr"; v.QrContent = pgtype.Text{} }, []string{"QR içeriğini girin."}},
		{"membership", func(v *database.Benefit, _ *database.Business) {
			v.Method = "membership"
			v.MembershipLabel = pgtype.Text{}
		}, []string{"Üyelik kartı metnini girin."}},
		{"unknown method", func(v *database.Benefit, _ *database.Business) { v.Method = "invalid" }, []string{"Geçerli bir kullanım yöntemi seçin."}},
		{"business ended", func(_ *database.Benefit, b *database.Business) { b.PartnershipStatus = "ended" }, []string{"İş ortaklığı aktif olmalıdır (mevcut durum: Sonlandırıldı)."}},
		{"business discussing", func(_ *database.Benefit, b *database.Business) { b.PartnershipStatus = "discussing" }, []string{"İş ortaklığı aktif olmalıdır (mevcut durum: Görüşülüyor)."}},
		{"business contacted", func(_ *database.Benefit, b *database.Business) { b.PartnershipStatus = "contacted" }, []string{"İş ortaklığı aktif olmalıdır (mevcut durum: İletişim kuruldu)."}},
		{"business awaiting", func(_ *database.Benefit, b *database.Business) { b.PartnershipStatus = "awaiting" }, []string{"İş ortaklığı aktif olmalıdır (mevcut durum: Anlaşma bekleniyor)."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, b := publishableBenefit(), database.Business{PartnershipStatus: "active"}
			tc.change(&v, &b)
			if got := publishIssues(v, b, now); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("issues=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestPublishIssuesMethodsAndInclusiveIstanbulEndDate(t *testing.T) {
	for _, method := range []string{"code", "link", "qr", "membership"} {
		t.Run(method, func(t *testing.T) {
			v := publishableBenefit()
			v.Method = method
			v.ValidUntil = pgtype.Date{Time: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), Valid: true}
			b := database.Business{PartnershipStatus: "active"}
			for _, at := range []string{"2026-10-20T21:00:00Z", "2026-10-21T20:59:59Z"} {
				now, _ := time.Parse(time.RFC3339, at)
				if got := publishIssues(v, b, now); len(got) != 0 {
					t.Fatalf("end date must allow publication at %s: %q", at, got)
				}
			}
			now, _ := time.Parse(time.RFC3339, "2026-10-21T21:00:00Z")
			want := []string{"Bitiş tarihi geçmiş; bugünü veya sonraki bir tarihi seçin."}
			if got := publishIssues(v, b, now); !reflect.DeepEqual(got, want) {
				t.Fatalf("issues=%q want=%q", got, want)
			}
			v.ValidUntil = pgtype.Date{}
			v.ValidFrom = pgtype.Date{Time: time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), Valid: true}
			if got := publishIssues(v, b, now); len(got) != 0 {
				t.Fatalf("scheduled publication must remain allowed: %q", got)
			}
		})
	}
}

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
