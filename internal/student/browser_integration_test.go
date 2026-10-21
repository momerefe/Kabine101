//go:build integration

package student_test

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
	"sync/atomic"
	"testing"
	"time"

	"github.com/momerefe/Kabine101/internal/config"
)

// TestStudentBrowser joins the production web build to an isolated database and
// API. It is opt-in so ordinary integration runs need neither Chrome nor dist.
func TestStudentBrowser(t *testing.T) {
	if os.Getenv("KOC_STUDENT_BROWSER") != "1" {
		t.Skip("set KOC_STUDENT_BROWSER=1 after pnpm build; requires Python and Chrome")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "dist", "index.html")); err != nil {
		t.Fatal("build the web application before running student browser acceptance")
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
	const email = "browser-student@ogrenci.example"
	_, studentID := h.account(email, "student")
	ids := map[string]string{}
	for _, method := range []string{"code", "link", "qr", "membership"} {
		_, id := h.fixture(studentID)
		h.exec("UPDATE benefits SET title=$2,method=$3 WHERE id=$1", id, "Student browser "+method, method)
		ids[method] = id
	}
	payload, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python", filepath.Join(root, "tests", "run_student_real_checks.py"))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "KOC_BROWSER_URL="+web.URL, "KOC_BROWSER_STUDENT="+email, "KOC_BROWSER_PASSWORD="+testPassword, "KOC_BROWSER_BENEFITS="+string(payload))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("student browser acceptance failed: %v\n%s", err, output)
	}
	t.Log(string(output))
}
