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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/momerefe/Kabine101/internal/config"
)

// TestManagement3EBrowser connects the production build to a disposable schema.
// All accounts, content, views, favorites and audit fixtures belong to that schema.
func TestManagement3EBrowser(t *testing.T) {
	if os.Getenv("KOC_MANAGEMENT3E_BROWSER") != "1" {
		t.Skip("set KOC_MANAGEMENT3E_BROWSER=1 after pnpm build; requires Python and Chrome")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "dist", "index.html")); err != nil {
		t.Fatal("build the web application before running 3E browser acceptance")
	}
	var proxy atomic.Pointer[httputil.ReverseProxy]
	files := http.FileServer(http.Dir(filepath.Join(root, "dist")))
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if p := proxy.Load(); p != nil {
				p.ServeHTTP(w, r)
			} else {
				http.Error(w, "starting", http.StatusServiceUnavailable)
			}
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
	const username = "Browser3EOperator"
	if err = h.service.BootstrapOperator(context.Background(), "Synthetic 3E operator", username, testPassword); err != nil {
		t.Fatal(err)
	}
	admin := h.browser()
	admin.send("POST", "/api/auth/admin-login", map[string]any{"username": username, "password": testPassword}).want(t, 200)
	admin.session()
	const email = "browser-3e-student@ogrenci.example"
	_, studentID := h.account(email, "student")
	overviewExec(t, h, "UPDATE accounts SET name='Synthetic 3E Student' WHERE id=$1", studentID)
	benefit := insightBenefit(t, h, admin, true)
	overviewExec(t, h, "UPDATE benefits SET title='Synthetic 3E coffee' WHERE id=$1", benefit)
	// Enough synthetic history for real server pagination, without touching dev data.
	overviewExec(t, h, `INSERT INTO audit_events(actor_id,action,entity_type,entity_id) SELECT a.id,'business.save','business',b.business_id FROM accounts a CROSS JOIN benefits b CROSS JOIN generate_series(1,55) WHERE a.username=$1 AND b.id=$2`, username, benefit)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python", filepath.Join(root, "tests", "run_management3e_checks.py"))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "KOC_BROWSER_URL="+web.URL, "KOC_BROWSER_USERNAME="+username, "KOC_BROWSER_PASSWORD="+testPassword, "KOC_BROWSER_STUDENT="+email, "KOC_BROWSER_STUDENT_ID="+studentID, "KOC_BROWSER_BENEFIT="+benefit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("3E browser acceptance failed: %v\n%s", err, output)
	}
	t.Log(string(output))
}
