//go:build integration

package management_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/momerefe/Kabine101/internal/benefitimage"
)

func imageFixture(t *testing.T, format string) string {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 4, 3))
	im.SetNRGBA(2, 1, color.NRGBA{R: 255, B: 150, A: 255})
	var data bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&data, im)
	} else {
		err = jpeg.Encode(&data, im, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(data.Bytes())
}

func fetchImage(t *testing.T, b *browser, path string, status int, mime string) []byte {
	t.Helper()
	r, err := b.client.Get(b.h.server.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != status {
		t.Fatalf("image HTTP %d want %d: %s", r.StatusCode, status, data)
	}
	if r.Header.Get("Cache-Control") != "no-store" || r.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("private image security headers missing")
	}
	if status == 200 {
		if r.Header.Get("Content-Type") != mime || r.Header.Get("Content-Length") != strconv.Itoa(len(data)) {
			t.Fatal("image content headers incorrect")
		}
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			t.Fatal("image endpoint returned invalid raster data")
		}
	}
	return data
}

func TestBenefitImagesAtomicLifecycleAndPrivateVisibility(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("admin@ogrenci.example", "admin")
	student, studentID := h.account("student@ogrenci.example", "student")
	anon := h.browser()
	business, id := h.uuid(), h.uuid()
	bp, p := "/api/admin/businesses/"+business, "/api/admin/benefits/"+id
	adminImage, studentImage := p+"/image", "/api/student/benefits/"+id+"/image"
	businessWrite := businessBody(0)
	businessWrite["image_path"] = "assets/cafe.svg"
	businessRevision := revision(admin.send("PUT", bp, businessWrite).want(t, 200), "business")
	body := benefitBody(business, 0)
	body["image_data"] = imageFixture(t, "png")
	anon.send("PUT", p, body).want(t, 401)
	student.send("PUT", p, body).want(t, 403)
	admin.request("PUT", p, body, "", h.origin).want(t, 403)
	admin.request("PUT", p, body, admin.csrf, "https://other.example").want(t, 403)
	fetchImage(t, anon, adminImage, 401, "")
	fetchImage(t, anon, studentImage, 401, "")
	fetchImage(t, student, adminImage, 403, "")
	created := admin.send("PUT", p, body).want(t, 200)
	rev := revision(created, "benefit")
	if entity(created, "benefit")["image_path"] != adminImage+"?v="+strconv.Itoa(rev) {
		t.Fatalf("administrator image path missing: %s", created.Body)
	}
	before := fetchImage(t, admin, adminImage, 200, "image/png")
	fetchImage(t, student, studentImage, 404, "")
	fetchImage(t, admin, studentImage, 404, "")
	var hasImage bool
	var stored int
	if err := h.pool.QueryRow(context.Background(), "SELECT has_image,(SELECT count(*) FROM benefit_images WHERE benefit_id=$1) FROM benefits WHERE id=$1", id).Scan(&hasImage, &stored); err != nil || !hasImage || stored != 1 {
		t.Fatalf("image marker inconsistent: %t %d %v", hasImage, stored, err)
	}
	rev = revision(admin.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200), "benefit")
	fetchImage(t, student, studentImage, 200, "image/png")
	// Canonical parent visibility also holds when a legacy database row has not
	// yet gone through the normal business-close workflow that suspends benefits.
	if _, err := h.pool.Exec(context.Background(), "UPDATE businesses SET partnership_status='ended' WHERE id=$1", business); err != nil {
		t.Fatal(err)
	}
	fetchImage(t, student, studentImage, 404, "")
	fetchImage(t, admin, adminImage, 200, "image/png")
	if _, err := h.pool.Exec(context.Background(), "UPDATE businesses SET partnership_status='active' WHERE id=$1", business); err != nil {
		t.Fatal(err)
	}
	wantPath := studentImage + "?v=" + strconv.Itoa(rev)
	detail := student.send("GET", "/api/student/benefits/"+id, nil).want(t, 200)
	if entity(detail, "benefit")["image_path"] != wantPath {
		t.Fatalf("student image path incorrect: %s", detail.Body)
	}
	catalog := student.send("GET", "/api/student/catalog", nil).want(t, 200)
	var list struct {
		Benefits []map[string]any `json:"benefits"`
	}
	if err := json.Unmarshal(catalog.Body, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Benefits) != 1 || list.Benefits[0]["image_path"] != wantPath || strings.Contains(string(catalog.Body), "image_data") {
		t.Fatal("catalogue must expose the private image URL only")
	}
	// Benefit edits preserve omitted/null image data, and return to private drafts.
	for _, nullImage := range []bool{false, true} {
		body = benefitBody(business, rev)
		if nullImage {
			body["image_data"] = nil
		}
		updated := admin.send("PUT", p, body).want(t, 200)
		rev = revision(updated, "benefit")
		if entity(updated, "benefit")["publication_status"] != "draft" || entity(updated, "benefit")["image_path"] == "" {
			t.Fatal("saving without image data lost the image or retained publication")
		}
		if !bytes.Equal(before, fetchImage(t, admin, adminImage, 200, "image/png")) {
			t.Fatal("omitted/null image changed stored bytes")
		}
		fetchImage(t, student, studentImage, 404, "")
	}
	stale := rev
	body = benefitBody(business, rev)
	body["image_data"] = imageFixture(t, "jpeg")
	rev = revision(admin.send("PUT", p, body).want(t, 200), "benefit")
	replacement := fetchImage(t, admin, adminImage, 200, "image/jpeg")
	body = benefitBody(business, stale)
	body["image_data"] = ""
	admin.send("PUT", p, body).want(t, 409)
	if !bytes.Equal(replacement, fetchImage(t, admin, adminImage, 200, "image/jpeg")) {
		t.Fatal("stale update changed image bytes")
	}
	// Publishing and then suspending or closing a business withdraws the same URL.
	rev = revision(admin.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200), "benefit")
	fetchImage(t, student, studentImage, 200, "image/jpeg")
	rev = revision(admin.send("POST", p+"/suspend", map[string]any{"revision": rev, "reason": "Synthetic suspension"}).want(t, 200), "benefit")
	fetchImage(t, student, studentImage, 404, "")
	rev = revision(admin.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200), "benefit")
	closed := businessBody(businessRevision)
	closed["partnership_status"] = "ended"
	businessRevision = revision(admin.send("PUT", bp, closed).want(t, 200), "business")
	fetchImage(t, student, studentImage, 404, "")
	reopened := businessBody(businessRevision)
	reopened["image_path"] = "assets/cafe.svg"
	admin.send("PUT", bp, reopened).want(t, 200)
	rev = revision(admin.send("GET", p, nil).want(t, 200), "benefit")
	rev = revision(admin.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200), "benefit")
	// Scheduled and expired published benefits retain the catalogue's image visibility.
	if _, err := h.pool.Exec(context.Background(), "UPDATE benefits SET valid_from=(now() AT TIME ZONE 'Europe/Istanbul')::date+1 WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	fetchImage(t, student, studentImage, 200, "image/jpeg")
	if _, err := h.pool.Exec(context.Background(), "UPDATE benefits SET valid_from=NULL,valid_until=(now() AT TIME ZONE 'Europe/Istanbul')::date-1 WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	fetchImage(t, student, studentImage, 200, "image/jpeg")
	admin.send("POST", "/api/admin/accounts/"+studentID+"/status", map[string]any{"status": "disabled", "current_password": testPassword}).want(t, 200)
	fetchImage(t, student, studentImage, 401, "")
	// Removing an image deletes its bytes, clears the marker and restores business fallback.
	body = benefitBody(business, rev)
	body["image_data"] = ""
	removed := admin.send("PUT", p, body).want(t, 200)
	rev = revision(removed, "benefit")
	if entity(removed, "benefit")["image_path"] != "" {
		t.Fatal("removed image path retained")
	}
	fetchImage(t, admin, adminImage, 404, "")
	if err := h.pool.QueryRow(context.Background(), "SELECT has_image,(SELECT count(*) FROM benefit_images WHERE benefit_id=$1) FROM benefits WHERE id=$1", id).Scan(&hasImage, &stored); err != nil || hasImage || stored != 0 {
		t.Fatalf("image deletion inconsistent: %t %d %v", hasImage, stored, err)
	}
	admin.send("POST", p+"/publish", map[string]any{"revision": rev, "conditions_reviewed": true}).want(t, 200)
	other, _ := h.account("other@ogrenci.example", "student")
	detail = other.send("GET", "/api/student/benefits/"+id, nil).want(t, 200)
	if entity(detail, "benefit")["image_path"] != "assets/cafe.svg" {
		t.Fatal("removing the custom image did not restore the business image")
	}
}

func TestBenefitImageInvalidUploadAndAuditRollback(t *testing.T) {
	h := newHarness(t)
	admin, _ := h.account("admin@ogrenci.example", "admin")
	business, id := h.uuid(), h.uuid()
	admin.send("PUT", "/api/admin/businesses/"+business, businessBody(0)).want(t, 200)
	p := "/api/admin/benefits/" + id
	body := benefitBody(business, 0)
	body["image_data"] = imageFixture(t, "png")
	rev := revision(admin.send("PUT", p, body).want(t, 200), "benefit")
	before := fetchImage(t, admin, p+"/image", 200, "image/png")
	for _, tc := range []struct {
		raw    string
		status int
	}{
		{"not-base64", 400},
		{base64.StdEncoding.EncodeToString([]byte("<svg><script>bad</script></svg>")), 400},
		{base64.StdEncoding.EncodeToString(make([]byte, benefitimage.MaxBytes+1)), 413},
	} {
		body = benefitBody(business, rev)
		body["image_data"] = tc.raw
		r := admin.send("PUT", p, body).want(t, tc.status)
		var issue map[string]string
		if err := json.Unmarshal(r.Body, &issue); err != nil || issue["code"] != "invalid_request" || !strings.Contains(issue["message"], "Görsel") {
			t.Fatalf("image error not actionable: %s", r.Body)
		}
		if revision(admin.send("GET", p, nil).want(t, 200), "benefit") != rev {
			t.Fatal("invalid upload changed revision")
		}
	}
	// Benefit uploads alone accept the larger request envelope.
	large := benefitBody(business, rev)
	large["image_data"] = strings.Repeat("A", benefitimage.MaxBodyBytes)
	admin.send("PUT", p, large).want(t, http.StatusRequestEntityTooLarge)
	ordinary := businessBody(1)
	ordinary["notes"] = strings.Repeat("x", 64*1024)
	admin.send("PUT", "/api/admin/businesses/"+business, ordinary).want(t, 400)
	_, err := h.pool.Exec(context.Background(), `CREATE FUNCTION reject_image_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic audit failure'; END $$; CREATE TRIGGER reject_image_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_image_audit()`)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"", imageFixture(t, "jpeg")} {
		body = benefitBody(business, rev)
		body["image_data"] = raw
		admin.send("PUT", p, body).want(t, 503)
		if revision(admin.send("GET", p, nil).want(t, 200), "benefit") != rev || !bytes.Equal(before, fetchImage(t, admin, p+"/image", 200, "image/png")) {
			t.Fatal("failed audit committed a partial benefit or image change")
		}
	}
}
