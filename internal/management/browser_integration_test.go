//go:build integration

package management_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/momerefe/Kabine101/internal/config"
)

// This optional acceptance check uses the same isolated PostgreSQL harness as
// the API tests, and the built web application. It never opens development data.
func TestManagementBrowser(t *testing.T) {
	if os.Getenv("KOC_BROWSER_TEST") != "1" {
		t.Skip("set KOC_BROWSER_TEST=1 after pnpm build; requires Python and Chrome")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "dist", "index.html")); err != nil {
		t.Fatal("build the web application before running browser acceptance")
	}
	var proxy atomic.Pointer[httputil.ReverseProxy]
	files := http.FileServer(http.Dir(filepath.Join(root, "dist")))
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			p := proxy.Load()
			if p == nil {
				http.Error(w, "starting", 503)
				return
			}
			p.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
	t.Cleanup(web.Close)
	h := newHarness(t, func(cfg *config.Config) { cfg.AppOrigin = web.URL })
	target, _ := url.Parse(h.server.URL)
	proxy.Store(httputil.NewSingleHostReverseProxy(target))
	const username = "BrowserOperator"
	if err := h.service.BootstrapOperator(context.Background(), "Test yöneticisi", username, testPassword); err != nil {
		t.Fatal("cannot provision isolated browser operator")
	}
	const student = "browser-student@ogrenci.example"
	h.account(student, "student")
	script := "run_management_checks.py"
	if os.Getenv("KOC_RESPONSIVE_TEST") == "1" {
		script = "run_responsive_checks.py"
		h.browser().send("POST", "/api/auth/register", map[string]any{"name": "Doğrulama bekleyen test öğrencisi", "email": "pending-layout@ogrenci.example", "password": testPassword}).want(t, 202)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python", filepath.Join(root, "tests", script))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "KOC_BROWSER_URL="+web.URL, "KOC_BROWSER_USERNAME="+username, "KOC_BROWSER_PASSWORD="+testPassword, "KOC_BROWSER_STUDENT="+student)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser acceptance failed: %v\n%s", err, output)
	}
	t.Log(string(output))
}
