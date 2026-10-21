package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/momerefe/Kabine101/internal/config"
)

func TestStrictJSONRejectsNullUnknownTrailingAndLargeBodies(t *testing.T) {
	for _, body := range []string{"null", "[]", `{"unexpected":true}`, `{} {}`, `{"known":"` + strings.Repeat("x", 16*1024) + `"}`} {
		var in struct {
			Known string `json:"known"`
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		if decode(w, r, &in) || w.Code != 400 {
			t.Fatalf("invalid body accepted: %.40s", body)
		}
	}
}

func TestCSRFRequiresExactOriginAndCookieTokenPair(t *testing.T) {
	s := &Service{cfg: config.Config{AppOrigin: "https://school.test"}}
	token, _, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	other, _, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	handler := s.csrf(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	cases := []struct {
		origin, header, cookie, content string
		status                          int
	}{
		{"https://school.test", token, token, "application/json", 204},
		{"", token, token, "application/json", 403},
		{"https://school.test.evil", token, token, "application/json", 403},
		{"https://school.test", other, token, "application/json", 403},
		{"https://school.test", token, "", "application/json", 403},
		{"https://school.test", token, token, "text/plain", 400},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("POST", "https://school.test/api/auth/logout", strings.NewReader("{}"))
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("X-CSRF-Token", tc.header)
		r.Header.Set("Content-Type", tc.content)
		if tc.cookie != "" {
			r.AddCookie(&http.Cookie{Name: CSRFCookie, Value: tc.cookie})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("origin=%q status=%d want=%d", tc.origin, w.Code, tc.status)
		}
	}
}

func TestProductionCookiesUseHostPrefixAndSecurityAttributes(t *testing.T) {
	s := &Service{cfg: config.Config{Environment: "production"}}
	w := httptest.NewRecorder()
	s.cookie(w, SessionCookie, "value", 24*time.Hour)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing cookie")
	}
	c := cookies[0]
	if c.Name != "__Host-kb_session" || !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Domain != "" {
		t.Fatal("production cookie security attributes missing")
	}
}

func TestSchoolDomainHasNoSuffixOrSubdomainBypass(t *testing.T) {
	s := &Service{cfg: config.Config{SchoolDomain: "stu.koc.k12.tr"}}
	for _, email := range []string{"student@stu.koc.k12.tr.evil", "student@other.stu.koc.k12.tr", "student@koc.k12.tr", "Name <student@stu.koc.k12.tr>", "student@stu.koc.k12.tr\r\nBcc:evil@example.com"} {
		if _, ok := s.email(email); ok {
			t.Fatal("wrong domain accepted")
		}
	}
	email, ok := s.email(" STUDENT@STU.KOC.K12.TR ")
	if !ok || email != "student@stu.koc.k12.tr" {
		t.Fatal("normalization failed")
	}
}

func TestLoginFailureLimiterResetsAfterSuccessfulPassword(t *testing.T) {
	l := limiter{entries: make(map[string]rateEntry)}
	r := httptest.NewRequest("POST", "/", nil)
	for i := 0; i < 12; i++ {
		if !l.allow(r, "login", "person@school.test") {
			t.Fatal("limited before threshold")
		}
	}
	if l.allow(r, "login", "person@school.test") {
		t.Fatal("failure burst not bounded")
	}
	l.resetEmail("login", "person@school.test")
	if !l.allow(r, "login", "person@school.test") {
		t.Fatal("successful password did not reset per-account failures")
	}
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	for i := 0; i < 600; i++ {
		l.allow(r, "login", "")
	}
	r.Header.Set("X-Forwarded-For", "5.6.7.8")
	if l.allow(r, "login", "") {
		t.Fatal("untrusted forwarded header bypassed IP limit")
	}
}
