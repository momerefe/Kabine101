package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndReadiness(t *testing.T) {
	for _, tc := range []struct {
		path    string
		dbError error
		status  int
		body    string
	}{
		{"/api/health", errors.New("database down"), 200, "ok"},
		{"/api/ready", nil, 200, "ok"},
		{"/api/ready", errors.New("password=secret"), 503, "unavailable"},
		{"/api/accounts", nil, 404, ""},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			router := Router(func(context.Context) error { return tc.dbError })
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("got %d, want %d", w.Code, tc.status)
			}
			if tc.body != "" && !strings.Contains(w.Body.String(), tc.body) {
				t.Fatal(w.Body.String())
			}
			if strings.Contains(w.Body.String(), "secret") {
				t.Fatal("database error leaked")
			}
		})
	}
}
