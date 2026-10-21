// Package management owns administrator-only partnership and benefit workflows.
package management

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	r.Get("/api/admin/overview", s.overview)
	r.Get("/api/admin/businesses", s.businesses)
	r.Get("/api/admin/businesses/{id}", s.business)
	r.Get("/api/admin/benefits", s.benefits)
	r.Get("/api/admin/benefits/{id}", s.benefit)
	r.Get("/api/admin/purchases", s.purchases)
	r.Group(func(r chi.Router) {
		r.Use(s.auth.CSRF)
		r.Put("/api/admin/businesses/{id}", s.putBusiness)
		r.Put("/api/admin/benefits/{id}", s.putBenefit)
		r.Post("/api/admin/benefits/{id}/publish", s.publish)
		r.Post("/api/admin/benefits/{id}/suspend", s.suspend)
		r.Post("/api/admin/purchases", s.createPurchase)
		r.Post("/api/admin/purchases/{id}/review", s.reviewPurchase)
	})
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code string) {
	reply(w, status, map[string]string{"code": code, "message": map[string]string{"invalid_request": "Alanları kontrol edin.", "conflict": "Kayıt değişti. Güncel kaydı açıp yeniden deneyin.", "not_found": "Kayıt bulunamadı.", "forbidden": "Yönetici yetkisi veya oturum geçerli değil.", "unavailable": "İşlem tamamlanamadı. Yeniden deneyin."}[code]})
}
func handleError(w http.ResponseWriter, e error) {
	var pe *pgconn.PgError
	switch {
	case errors.Is(e, pgx.ErrNoRows):
		fail(w, 404, "not_found")
	case errors.As(e, &pe) && pe.Code == "23505":
		fail(w, 409, "conflict")
	case errors.As(e, &pe) && (strings.HasPrefix(pe.Code, "22") || strings.HasPrefix(pe.Code, "23")):
		fail(w, 400, "invalid_request")
	default:
		fail(w, 503, "unavailable")
	}
}
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	b, e := io.ReadAll(r.Body)
	if e != nil || len(bytes.TrimSpace(b)) == 0 || bytes.TrimSpace(b)[0] != '{' {
		fail(w, 400, "invalid_request")
		return false
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(dst) != nil || d.Decode(new(any)) != io.EOF {
		fail(w, 400, "invalid_request")
		return false
	}
	return true
}
func identifier(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	var id pgtype.UUID
	e := id.Scan(chi.URLParam(r, "id"))
	if e != nil || !id.Valid {
		fail(w, 400, "invalid_request")
		return id, false
	}
	return id, true
}
func copyJSON(src, dst any) { b, _ := json.Marshal(src); _ = json.Unmarshal(b, dst) }
func object(v any) map[string]any {
	m := map[string]any{}
	copyJSON(v, &m)
	for k, v := range m {
		if v == nil {
			m[k] = ""
		}
	}
	return m
}
func fields(v any, keys string) map[string]any {
	src := object(v)
	out := map[string]any{}
	for _, key := range strings.Fields(keys) {
		out[key] = src[key]
	}
	return out
}
func businessDTO(v database.Business) map[string]any {
	return fields(v, "id revision name category description image_path contact_person contact_info branches partnership_status last_contact notes created_at updated_at")
}

var istanbul = func() *time.Location {
	location, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		panic(err)
	}
	return location
}()

func benefitStatus(v database.Benefit, now time.Time) string {
	if v.PublicationStatus != "published" {
		return v.PublicationStatus
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
func benefitDTO(v database.Benefit, b database.Business) map[string]any {
	m := fields(v, "id revision business_id title description perk conditions instructions products branches exceptions minimum_spend currency verification_requirement valid_from valid_until method code target_url qr_content membership_label support_contact featured publication_status suspension_reason reviewed_at published_at created_at updated_at")
	m["status"] = benefitStatus(v, time.Now())
	m["business_name"] = b.Name
	m["category"] = b.Category
	m["minimum_spend"] = ""
	if v.MinimumSpend.Valid {
		x, _ := v.MinimumSpend.Value()
		m["minimum_spend"] = x
	}
	return m
}
func (s *Service) businesses(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	rows, e := s.q.ManagementListBusiness(r.Context(), database.ManagementListBusinessParams{Q: strings.TrimSpace(r.URL.Query().Get("q")), Category: r.URL.Query().Get("category"), Status: r.URL.Query().Get("status")})
	if e != nil {
		handleError(w, e)
		return
	}
	out := []map[string]any{}
	for _, v := range rows {
		m := businessDTO(v)
		out = append(out, m)
	}
	truncated := len(rows) > 1000
	if len(out) > 1000 {
		out = out[:1000]
	}
	reply(w, 200, map[string]any{"businesses": out, "truncated": truncated})
}
func (s *Service) business(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	v, e := s.q.ManagementGetBusiness(r.Context(), id)
	if e != nil {
		handleError(w, e)
		return
	}
	reply(w, 200, map[string]any{"business": businessDTO(v)})
}
func (s *Service) benefits(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	rows, e := s.q.ManagementListBenefit(r.Context(), database.ManagementListBenefitParams{Q: strings.TrimSpace(r.URL.Query().Get("q")), Category: r.URL.Query().Get("category"), Status: r.URL.Query().Get("status")})
	if e != nil {
		handleError(w, e)
		return
	}
	byID := map[pgtype.UUID]database.Business{}
	out := []map[string]any{}
	for _, v := range rows {
		b, ok := byID[v.BusinessID]
		if !ok {
			b, e = s.q.ManagementGetBusiness(r.Context(), v.BusinessID)
			if e != nil {
				handleError(w, e)
				return
			}
			byID[v.BusinessID] = b
		}
		m := benefitDTO(v, b)
		out = append(out, m)
	}
	truncated := len(rows) > 1000
	if len(out) > 1000 {
		out = out[:1000]
	}
	reply(w, 200, map[string]any{"benefits": out, "truncated": truncated})
}
func (s *Service) benefit(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	v, e := s.q.ManagementGetBenefit(r.Context(), id)
	if e != nil {
		handleError(w, e)
		return
	}
	b, e := s.q.ManagementGetBusiness(r.Context(), v.BusinessID)
	if e != nil {
		handleError(w, e)
		return
	}
	reply(w, 200, map[string]any{"benefit": benefitDTO(v, b)})
}

// Every mutation locks the canonical actor account and rechecks the current session.
func (s *Service) begin(w http.ResponseWriter, r *http.Request) (pgx.Tx, *database.Queries, pgtype.UUID, bool) {
	actor, hash, ok := s.auth.RequireAdmin(w, r)
	if !ok {
		return nil, nil, actor, false
	}
	tx, e := s.pool.Begin(r.Context())
	if e != nil {
		handleError(w, e)
		return nil, nil, actor, false
	}
	if !s.auth.RecheckAdmin(r, tx, actor, hash) {
		_ = tx.Rollback(r.Context())
		fail(w, 403, "forbidden")
		return nil, nil, actor, false
	}
	return tx, database.New(tx), actor, true
}
func audit(q *database.Queries, r *http.Request, actor pgtype.UUID, action, kind string, id pgtype.UUID) error {
	return q.ManagementAudit(r.Context(), database.ManagementAuditParams{ActorID: actor, Action: action, EntityType: kind, EntityID: id})
}
func validDate(s string) bool {
	if s == "" {
		return true
	}
	d, e := time.Parse("2006-01-02", s)
	return e == nil && d.Format("2006-01-02") == s
}
func oneOf(v string, values ...string) bool {
	for _, x := range values {
		if v == x {
			return true
		}
	}
	return false
}
func validText(v string, max int, required bool) bool {
	return utf8.ValidString(v) && utf8.RuneCountInString(v) <= max && (!required || strings.TrimSpace(v) != "") && !strings.ContainsRune(v, 0)
}

var amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?$`)

func safeURL(s string) bool {
	u, e := url.Parse(s)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil
}

type BusinessWrite struct {
	Revision          *int32 `json:"revision"`
	Name              string `json:"name"`
	Category          string `json:"category"`
	Description       string `json:"description"`
	ImagePath         string `json:"image_path"`
	ContactPerson     string `json:"contact_person"`
	ContactInfo       string `json:"contact_info"`
	Branches          string `json:"branches"`
	PartnershipStatus string `json:"partnership_status"`
	LastContact       string `json:"last_contact"`
	Notes             string `json:"notes"`
}
type BenefitWrite struct {
	Revision                *int32   `json:"revision"`
	BusinessID              string   `json:"business_id"`
	Title                   string   `json:"title"`
	Description             string   `json:"description"`
	Perk                    string   `json:"perk"`
	Conditions              []string `json:"conditions"`
	Instructions            []string `json:"instructions"`
	Products                string   `json:"products"`
	Branches                string   `json:"branches"`
	Exceptions              string   `json:"exceptions"`
	MinimumSpend            string   `json:"minimum_spend"`
	Currency                string   `json:"currency"`
	VerificationRequirement string   `json:"verification_requirement"`
	ValidFrom               string   `json:"valid_from"`
	ValidUntil              string   `json:"valid_until"`
	Method                  string   `json:"method"`
	Code                    string   `json:"code"`
	TargetURL               string   `json:"target_url"`
	QRContent               string   `json:"qr_content"`
	MembershipLabel         string   `json:"membership_label"`
	SupportContact          string   `json:"support_contact"`
	Featured                bool     `json:"featured"`
}

func (v BusinessWrite) valid() bool {
	if v.Revision == nil || *v.Revision < 0 || !validText(v.Name, 160, true) || !oneOf(v.Category, "food", "fashion", "tech", "sport", "culture", "other") || !oneOf(v.PartnershipStatus, "discussing", "contacted", "awaiting", "active", "ended") || !validDate(v.LastContact) {
		return false
	}
	if v.ImagePath != "" && !oneOf(v.ImagePath, "assets/books.svg", "assets/cafe.svg", "assets/culture.svg", "assets/demo-qr.svg", "assets/fashion.svg", "assets/favicon.svg", "assets/fitness.svg", "assets/placeholder.svg", "assets/tech.svg") {
		return false
	}
	for _, s := range []string{v.Description, v.ContactPerson, v.ContactInfo, v.Branches, v.Notes} {
		if !validText(s, 10000, false) {
			return false
		}
	}
	return true
}
func (v *BenefitWrite) valid() bool {
	var id pgtype.UUID
	if id.Scan(v.BusinessID) != nil || !id.Valid || v.Revision == nil || *v.Revision < 0 || !validText(v.Title, 200, true) || !oneOf(v.Method, "code", "link", "qr", "membership") || v.Currency != "TRY" || !validDate(v.ValidFrom) || !validDate(v.ValidUntil) || (v.ValidFrom != "" && v.ValidUntil != "" && v.ValidFrom > v.ValidUntil) || (v.MinimumSpend != "" && !amountPattern.MatchString(v.MinimumSpend)) {
		return false
	}
	if v.TargetURL != "" && !safeURL(v.TargetURL) {
		return false
	}
	for _, s := range []string{v.Description, v.Perk, v.Products, v.Branches, v.Exceptions, v.VerificationRequirement, v.Code, v.TargetURL, v.QRContent, v.MembershipLabel, v.SupportContact} {
		if !validText(s, 10000, false) {
			return false
		}
	}
	for _, ss := range [][]string{v.Conditions, v.Instructions} {
		if len(ss) > 100 {
			return false
		}
		for _, s := range ss {
			if !validText(s, 2000, true) {
				return false
			}
		}
	}
	if v.Conditions == nil {
		v.Conditions = []string{}
	}
	if v.Instructions == nil {
		v.Instructions = []string{}
	}
	if v.Method != "code" {
		v.Code = ""
	}
	if v.Method != "link" {
		v.TargetURL = ""
	}
	if v.Method != "qr" {
		v.QRContent = ""
	}
	if v.Method != "membership" {
		v.MembershipLabel = ""
	}
	return true
}
func (s *Service) putBusiness(w http.ResponseWriter, r *http.Request) {
	var in BusinessWrite
	if !decode(w, r, &in) {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	if !in.valid() {
		fail(w, 400, "invalid_request")
		return
	}
	tx, q, actor, ok := s.begin(w, r)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context())
	old, e := q.ManagementLockBusiness(r.Context(), id)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		handleError(w, e)
		return
	}
	exists := e == nil
	if (!exists && *in.Revision != 0) || (exists && old.Revision != *in.Revision) {
		fail(w, 409, "conflict")
		return
	}
	var v database.Business
	if !exists {
		var p database.ManagementCreateBusinessParams
		copyJSON(in, &p)
		p.ID = id
		v, e = q.ManagementCreateBusiness(r.Context(), p)
	} else {
		var p database.ManagementUpdateBusinessParams
		copyJSON(in, &p)
		p.ID = id
		v, e = q.ManagementUpdateBusiness(r.Context(), p)
	}
	if e != nil {
		handleError(w, e)
		return
	}
	if exists && old.PartnershipStatus == "active" && v.PartnershipStatus != "active" {
		ids, err := q.ManagementSuspendBusiness(r.Context(), id)
		if err != nil {
			handleError(w, err)
			return
		}
		for _, benefitID := range ids {
			if err = audit(q, r, actor, "benefit.suspend", "benefit", benefitID); err != nil {
				handleError(w, err)
				return
			}
		}
	}
	if e = audit(q, r, actor, "business.save", "business", id); e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		handleError(w, e)
		return
	}
	reply(w, 200, map[string]any{"business": businessDTO(v)})
}
func (s *Service) putBenefit(w http.ResponseWriter, r *http.Request) {
	var in BenefitWrite
	if !decode(w, r, &in) {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	if !in.valid() {
		fail(w, 400, "invalid_request")
		return
	}
	tx, q, actor, ok := s.begin(w, r)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context())
	var businessID pgtype.UUID
	_ = businessID.Scan(in.BusinessID)
	// A benefit's business is immutable. Lock its parent first for all state changes.
	prior, e := q.ManagementGetBenefit(r.Context(), id)
	exists := e == nil
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		handleError(w, e)
		return
	}
	if exists && prior.BusinessID != businessID {
		reply(w, 400, map[string]string{"code": "invalid_request", "message": "Kayıtlı avantajın işletmesi değiştirilemez. Diğer işletme için yeni bir avantaj oluşturun."})
		return
	}
	b, e := q.ManagementLockBusiness(r.Context(), businessID)
	if e != nil {
		handleError(w, e)
		return
	}
	old, e := q.ManagementLockBenefit(r.Context(), id)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		handleError(w, e)
		return
	}
	exists = e == nil
	if (!exists && *in.Revision != 0) || (exists && old.Revision != *in.Revision) {
		fail(w, 409, "conflict")
		return
	}
	var v database.Benefit
	if !exists {
		var p database.ManagementCreateBenefitParams
		copyJSON(in, &p)
		p.ID = id
		v, e = q.ManagementCreateBenefit(r.Context(), p)
	} else {
		var p database.ManagementUpdateBenefitParams
		copyJSON(in, &p)
		p.ID = id
		v, e = q.ManagementUpdateBenefit(r.Context(), p)
	}
	if e != nil {
		handleError(w, e)
		return
	}
	if e = audit(q, r, actor, "benefit.save", "benefit", id); e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		handleError(w, e)
		return
	}
	reply(w, 200, map[string]any{"benefit": benefitDTO(v, b)})
}
func (s *Service) publish(w http.ResponseWriter, r *http.Request) { s.transition(w, r, true) }
func (s *Service) suspend(w http.ResponseWriter, r *http.Request) { s.transition(w, r, false) }
func (s *Service) transition(w http.ResponseWriter, r *http.Request, publish bool) {
	var revision *int32
	var reviewed bool
	var reason string
	if publish {
		var in struct {
			Revision           *int32 `json:"revision"`
			ConditionsReviewed bool   `json:"conditions_reviewed"`
		}
		if !decode(w, r, &in) {
			return
		}
		revision = in.Revision
		reviewed = in.ConditionsReviewed
	} else {
		var in struct {
			Revision *int32 `json:"revision"`
			Reason   string `json:"reason"`
		}
		if !decode(w, r, &in) {
			return
		}
		revision = in.Revision
		reason = strings.TrimSpace(in.Reason)
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	if revision == nil || *revision < 1 || (publish && !reviewed) || (!publish && !validText(reason, 2000, true)) {
		fail(w, 400, "invalid_request")
		return
	}
	tx, q, actor, ok := s.begin(w, r)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context())
	prior, e := q.ManagementGetBenefit(r.Context(), id)
	if e != nil {
		handleError(w, e)
		return
	}
	b, e := q.ManagementLockBusiness(r.Context(), prior.BusinessID)
	if e != nil {
		handleError(w, e)
		return
	}
	v, e := q.ManagementLockBenefit(r.Context(), id)
	if e != nil {
		handleError(w, e)
		return
	}
	if v.Revision != *revision {
		fail(w, 409, "conflict")
		return
	}
	action := "benefit.suspend"
	if publish {
		if v.PublicationStatus == "published" {
			fail(w, 409, "conflict")
			return
		}
		today := time.Now().In(istanbul).Format("2006-01-02")
		payload := map[string]string{"code": v.Code.String, "link": v.TargetUrl.String, "qr": v.QrContent.String, "membership": v.MembershipLabel.String}[v.Method]
		if b.PartnershipStatus != "active" || (v.ValidUntil.Valid && v.ValidUntil.Time.Format("2006-01-02") < today) || strings.TrimSpace(v.Description) == "" || strings.TrimSpace(v.Perk) == "" || len(v.Conditions) == 0 || len(v.Instructions) == 0 || strings.TrimSpace(payload) == "" || (v.Method == "link" && !safeURL(payload)) {
			reply(w, 400, map[string]string{"code": "invalid_request", "message": "Yayın için aktif iş ortaklığı, geçerli tarihler, açıklama, ayrıcalık, koşullar, kullanım adımları ve seçilen yöntemin kullanım bilgisi gerekiyor."})
			return
		}
		v, e = q.ManagementPublish(r.Context(), database.ManagementPublishParams{ID: id, ReviewedBy: actor})
		action = "benefit.publish"
	} else {
		if v.PublicationStatus != "published" {
			fail(w, 409, "conflict")
			return
		}
		v, e = q.ManagementSuspend(r.Context(), database.ManagementSuspendParams{ID: id, SuspensionReason: pgtype.Text{String: reason, Valid: true}})
	}
	if e != nil {
		handleError(w, e)
		return
	}
	if e = audit(q, r, actor, action, "benefit", id); e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		handleError(w, e)
		return
	}
	reply(w, 200, map[string]any{"benefit": benefitDTO(v, b)})
}
