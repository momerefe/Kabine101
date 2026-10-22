// Package student exposes the eligible account's published catalogue and favorites.
package student

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/momerefe/Kabine101/internal/auth"
	"github.com/momerefe/Kabine101/internal/database"
)

type Service struct {
	pool *pgxpool.Pool
	auth *auth.Service
	q    *database.Queries
}

func New(pool *pgxpool.Pool, a *auth.Service) *Service { return &Service{pool, a, database.New(pool)} }
func (s *Service) RegisterRoutes(r chi.Router) {
	r.Get("/api/student/catalog", s.catalog)
	r.Get("/api/student/benefits/{id}", s.detail)
	r.Get("/api/student/benefits/{id}/image", s.benefitImage)
	r.Get("/api/student/benefits/{id}/redemption", s.redemption)
	r.Get("/api/student/favorites", s.favorites)
	r.Get("/api/student/settings", s.settings)
	r.Group(func(r chi.Router) {
		r.Use(s.auth.CSRF)
		r.Put("/api/student/favorites/{id}", s.favorite)
		r.Delete("/api/student/favorites/{id}", s.favorite)
		r.Post("/api/student/benefits/{id}/view", s.view)
	})
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code string) {
	reply(w, status, map[string]string{"code": code, "message": map[string]string{"invalid_request": "Geçersiz istek.", "not_found": "Avantaj bulunamadı.", "unavailable": "Avantaj şu anda kullanılamıyor.", "not_active": "Avantaj bu tarihte kullanılamıyor.", "forbidden": "Oturum geçerli değil."}[code]})
}
func dbfail(w http.ResponseWriter, e error) {
	if errors.Is(e, pgx.ErrNoRows) {
		fail(w, 404, "not_found")
	} else {
		fail(w, 503, "unavailable")
	}
}
func identifier(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	var id pgtype.UUID
	if id.Scan(chi.URLParam(r, "id")) != nil || !id.Valid {
		fail(w, 400, "invalid_request")
		return id, false
	}
	return id, true
}

var istanbul = func() *time.Location {
	l, e := time.LoadLocation("Europe/Istanbul")
	if e != nil {
		panic(e)
	}
	return l
}()

func benefitStatus(v database.Benefit, b database.Business, now time.Time) string {
	if v.PublicationStatus != "published" || b.PartnershipStatus != "active" {
		return "unavailable"
	}
	today := now.In(istanbul).Format("2006-01-02")
	if v.ValidUntil.Valid && v.ValidUntil.Time.Format("2006-01-02") < today {
		return "expired"
	}
	if v.ValidFrom.Valid && v.ValidFrom.Time.Format("2006-01-02") > today {
		return "scheduled"
	}
	return "active"
}
func date(v pgtype.Date) string {
	if v.Valid {
		return v.Time.Format("2006-01-02")
	}
	return ""
}

// This allowlist is deliberately independent of administrative serialization.
func benefitDTO(v database.Benefit, b database.Business, favorite bool) map[string]any {
	status := benefitStatus(v, b, time.Now())
	m := map[string]any{"id": v.ID, "business_id": "", "business_name": "", "image_path": "", "category": "", "title": "", "description": "", "perk": "", "conditions": []string{}, "instructions": []string{}, "products": "", "branches": "", "exceptions": "", "minimum_spend": "", "currency": "", "verification_requirement": "", "valid_from": "", "valid_until": "", "method": "", "featured": false, "status": status, "is_favorite": favorite, "support_contact": ""}
	if status == "unavailable" {
		return m
	}
	m["business_id"] = v.BusinessID
	m["business_name"] = b.Name
	m["image_path"] = b.ImagePath.String
	if v.HasImage {
		m["image_path"] = "/api/student/benefits/" + v.ID.String() + "/image?v=" + strconv.Itoa(int(v.Revision))
	}
	m["category"] = b.Category
	m["title"] = v.Title
	m["description"] = v.Description
	m["perk"] = v.Perk
	if v.Conditions != nil {
		m["conditions"] = v.Conditions
	}
	if v.Instructions != nil {
		m["instructions"] = v.Instructions
	}
	m["products"] = v.Products.String
	m["branches"] = v.Branches.String
	m["exceptions"] = v.Exceptions.String
	m["currency"] = v.Currency
	m["verification_requirement"] = v.VerificationRequirement.String
	m["valid_from"] = date(v.ValidFrom)
	m["valid_until"] = date(v.ValidUntil)
	m["method"] = v.Method
	m["featured"] = v.Featured
	m["support_contact"] = v.SupportContact.String
	if v.MinimumSpend.Valid {
		x, _ := v.MinimumSpend.Value()
		m["minimum_spend"] = x
	}
	return m
}
func (s *Service) catalog(w http.ResponseWriter, r *http.Request) {
	id, _, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	rows, e := s.q.StudentCatalog(r.Context(), database.StudentCatalogParams{AccountID: id, Q: strings.TrimSpace(r.URL.Query().Get("q")), Category: r.URL.Query().Get("category")})
	if e != nil {
		dbfail(w, e)
		return
	}
	out := []map[string]any{}
	for i, v := range rows {
		if i == 1000 {
			break
		}
		out = append(out, benefitDTO(v.Benefit, v.Business, v.IsFavorite))
	}
	reply(w, 200, map[string]any{"benefits": out, "truncated": len(rows) > 1000})
}
func (s *Service) favorites(w http.ResponseWriter, r *http.Request) {
	id, _, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	rows, e := s.q.StudentFavorites(r.Context(), id)
	if e != nil {
		dbfail(w, e)
		return
	}
	out := []map[string]any{}
	for i, v := range rows {
		if i == 1000 {
			break
		}
		out = append(out, benefitDTO(v.Benefit, v.Business, true))
	}
	reply(w, 200, map[string]any{"benefits": out, "truncated": len(rows) > 1000})
}
func (s *Service) detail(w http.ResponseWriter, r *http.Request) {
	account, _, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	v, e := s.q.ManagementGetBenefit(r.Context(), id)
	if e != nil {
		dbfail(w, e)
		return
	}
	b, e := s.q.ManagementGetBusiness(r.Context(), v.BusinessID)
	if e != nil {
		dbfail(w, e)
		return
	}
	if benefitStatus(v, b, time.Now()) == "unavailable" {
		fail(w, 404, "not_found")
		return
	}
	f, e := s.q.StudentHasFavorite(r.Context(), database.StudentHasFavoriteParams{AccountID: account, BenefitID: id})
	if e != nil {
		dbfail(w, e)
		return
	}
	reply(w, 200, map[string]any{"benefit": benefitDTO(v, b, f)})
}
func (s *Service) settings(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAccount(w, r); !ok {
		return
	}
	v, e := s.q.StudentSettings(r.Context())
	if e != nil {
		dbfail(w, e)
		return
	}
	reply(w, 200, map[string]any{"platform_name": v.PlatformName, "support_contact": v.SupportContact.String, "show_featured": v.ShowFeatured})
}

func (s *Service) view(w http.ResponseWriter, r *http.Request) {
	account, hash, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	body, err := io.ReadAll(r.Body)
	var in map[string]json.RawMessage
	if err != nil || json.Unmarshal(body, &in) != nil || in == nil || len(in) != 0 || len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '{' {
		fail(w, 400, "invalid_request")
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		dbfail(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	if !s.auth.RecheckAccount(r, tx, account, hash) {
		fail(w, 403, "forbidden")
		return
	}
	q := database.New(tx)
	a, err := q.FindAccountByID(r.Context(), account)
	if err != nil {
		dbfail(w, err)
		return
	}
	prior, err := q.ManagementGetBenefit(r.Context(), id)
	if err != nil {
		dbfail(w, err)
		return
	}
	b, err := q.ManagementLockBusiness(r.Context(), prior.BusinessID)
	if err != nil {
		dbfail(w, err)
		return
	}
	v, err := q.ManagementLockBenefit(r.Context(), id)
	if err != nil {
		dbfail(w, err)
		return
	}
	if benefitStatus(v, b, time.Now()) == "unavailable" {
		fail(w, 404, "not_found")
		return
	}
	if a.Role == "student" {
		if err = q.StudentRecordView(r.Context(), database.StudentRecordViewParams{AccountID: account, BenefitID: id}); err != nil {
			dbfail(w, err)
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		dbfail(w, err)
		return
	}
	reply(w, 200, map[string]string{"message": "Görüntüleme kaydedildi."})
}
func (s *Service) favorite(w http.ResponseWriter, r *http.Request) {
	account, hash, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	tx, e := s.pool.Begin(r.Context())
	if e != nil {
		dbfail(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	if !s.auth.RecheckAccount(r, tx, account, hash) {
		fail(w, 403, "forbidden")
		return
	}
	q := database.New(tx)
	add := r.Method == http.MethodPut
	// Deletion never requires the benefit to remain visible or published.
	if add {
		prior, e := q.ManagementGetBenefit(r.Context(), id)
		if e != nil {
			dbfail(w, e)
			return
		}
		b, e := q.ManagementLockBusiness(r.Context(), prior.BusinessID)
		if e != nil {
			dbfail(w, e)
			return
		}
		v, e := q.ManagementLockBenefit(r.Context(), id)
		if e != nil {
			dbfail(w, e)
			return
		}
		if benefitStatus(v, b, time.Now()) == "unavailable" {
			fail(w, 409, "unavailable")
			return
		}
		e = q.StudentAddFavorite(r.Context(), database.StudentAddFavoriteParams{AccountID: account, BenefitID: id})
		if e != nil {
			dbfail(w, e)
			return
		}
	} else {
		// Preserve parent-before-benefit lock ordering even for unavailable rows.
		prior, err := q.ManagementGetBenefit(r.Context(), id)
		if err == nil {
			if _, err = q.ManagementLockBusiness(r.Context(), prior.BusinessID); err != nil {
				dbfail(w, err)
				return
			}
			if _, err = q.ManagementLockBenefit(r.Context(), id); err != nil {
				dbfail(w, err)
				return
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			dbfail(w, err)
			return
		}
		e = q.StudentRemoveFavorite(r.Context(), database.StudentRemoveFavoriteParams{AccountID: account, BenefitID: id})
		if e != nil {
			dbfail(w, e)
			return
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		dbfail(w, e)
		return
	}
	reply(w, 200, map[string]bool{"is_favorite": add})
}
func (s *Service) redemption(w http.ResponseWriter, r *http.Request) {
	account, hash, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	tx, e := s.pool.Begin(r.Context())
	if e != nil {
		dbfail(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	if !s.auth.RecheckAccount(r, tx, account, hash) {
		fail(w, 403, "forbidden")
		return
	}
	q := database.New(tx)
	prior, e := q.ManagementGetBenefit(r.Context(), id)
	if e != nil {
		dbfail(w, e)
		return
	}
	b, e := q.ManagementLockBusiness(r.Context(), prior.BusinessID)
	if e != nil {
		dbfail(w, e)
		return
	}
	v, e := q.ManagementLockBenefit(r.Context(), id)
	if e != nil {
		dbfail(w, e)
		return
	}
	if v.PublicationStatus == "draft" {
		fail(w, 404, "not_found")
		return
	}
	status := benefitStatus(v, b, time.Now())
	if status == "unavailable" {
		fail(w, 409, "unavailable")
		return
	}
	if status != "active" {
		fail(w, 409, "not_active")
		return
	}
	out := map[string]string{"method": v.Method}
	switch v.Method {
	case "code":
		out["code"] = v.Code.String
	case "link":
		out["target_url"] = v.TargetUrl.String
	case "qr":
		out["qr_content"] = v.QrContent.String
	case "membership":
		out["membership_label"] = v.MembershipLabel.String
	default:
		fail(w, 409, "unavailable")
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		dbfail(w, e)
		return
	}
	reply(w, 200, out)
}
