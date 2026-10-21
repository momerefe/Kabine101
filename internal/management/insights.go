package management

import (
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

func settingsDTO(v database.PlatformSetting) map[string]any {
	return map[string]any{"platform_name": v.PlatformName, "support_contact": v.SupportContact.String, "show_featured": v.ShowFeatured, "revision": v.Revision}
}
func (s *Service) settings(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	v, err := s.q.ManagementSettings(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	reply(w, 200, map[string]any{"settings": settingsDTO(v), "school_domain": s.auth.SchoolDomain()})
}

var phoneContact = regexp.MustCompile(`^\+?[0-9][0-9 ()-]{5,30}$`)

func validSupportContact(v string) bool {
	if !validText(v, 254, false) || strings.ContainsAny(v, "\r\n") {
		return false
	}
	if v == "" {
		return true
	}
	if a, err := mail.ParseAddress(v); err == nil && a.Address == v {
		return true
	}
	if u, err := url.Parse(v); err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil {
		return true
	}
	if !phoneContact.MatchString(v) {
		return false
	}
	digits := 0
	for _, char := range v {
		if char >= '0' && char <= '9' {
			digits++
		}
	}
	return digits >= 6 && digits <= 20
}

func (s *Service) putSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PlatformName   *string `json:"platform_name"`
		SupportContact *string `json:"support_contact"`
		ShowFeatured   *bool   `json:"show_featured"`
		Revision       *int32  `json:"revision"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.PlatformName == nil || in.SupportContact == nil || in.ShowFeatured == nil || in.Revision == nil || *in.Revision < 1 {
		fail(w, 400, "invalid_request")
		return
	}
	name, contact := strings.TrimSpace(*in.PlatformName), strings.TrimSpace(*in.SupportContact)
	if !validText(name, 80, true) || strings.ContainsAny(name, "\r\n") || !validSupportContact(contact) {
		fail(w, 400, "invalid_request")
		return
	}
	tx, q, actor, ok := s.begin(w, r)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context())
	v, err := q.ManagementUpdateSettings(r.Context(), database.ManagementUpdateSettingsParams{PlatformName: name, SupportContact: contact, ShowFeatured: *in.ShowFeatured, Revision: *in.Revision})
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "conflict")
		return
	}
	if err == nil {
		err = audit(q, r, actor, "platform.settings", "platform", pgtype.UUID{})
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		handleError(w, err)
		return
	}
	reply(w, 200, map[string]any{"settings": settingsDTO(v), "school_domain": s.auth.SchoolDomain()})
}

func filterDate(v string) pgtype.Date {
	if v == "" {
		return pgtype.Date{}
	}
	t, _ := time.Parse("2006-01-02", v)
	return pgtype.Date{Time: t, Valid: true}
}

func (s *Service) auditEvents(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	values := r.URL.Query()
	query, action := strings.TrimSpace(values.Get("q")), values.Get("action")
	from, until := values.Get("from"), values.Get("until")
	page := 1
	var err error
	if p := values.Get("page"); p != "" {
		page, err = strconv.Atoi(p)
	}
	if err != nil || page < 1 || page > 1000000 || !validText(query, 100, false) || !validText(action, 100, false) || !validDate(from) || !validDate(until) || (from != "" && until != "" && from > until) {
		fail(w, 400, "invalid_request")
		return
	}
	ctx := r.Context()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		handleError(w, err)
		return
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	total, err := q.ManagementAuditCount(ctx, database.ManagementAuditCountParams{Q: query, Action: action, FromDate: filterDate(from), UntilDate: filterDate(until)})
	if err != nil {
		handleError(w, err)
		return
	}
	rows, err := q.ManagementAuditList(ctx, database.ManagementAuditListParams{Q: query, Action: action, FromDate: filterDate(from), UntilDate: filterDate(until), OffsetRows: int32((page - 1) * 50)})
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		handleError(w, err)
		return
	}
	events := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		events = append(events, object(row))
	}
	reply(w, 200, map[string]any{"events": events, "total": total, "page": page, "page_size": 50})
}

func (s *Service) analytics(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	ctx := r.Context()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		handleError(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var generatedAt time.Time
	if err = tx.QueryRow(ctx, "SELECT transaction_timestamp()").Scan(&generatedAt); err != nil {
		handleError(w, err)
		return
	}
	end := generatedAt.In(istanbul).Format("2006-01-02")
	start := generatedAt.In(istanbul).AddDate(0, 0, -29).Format("2006-01-02")
	q := s.q.WithTx(tx)
	totals, err := q.ManagementAnalyticsTotals(ctx, database.ManagementAnalyticsTotalsParams{PeriodStart: filterDate(start), PeriodEnd: filterDate(end)})
	if err != nil {
		handleError(w, err)
		return
	}
	rows, err := q.ManagementAnalyticsBenefits(ctx, database.ManagementAnalyticsBenefitsParams{PeriodStart: filterDate(start), PeriodEnd: filterDate(end)})
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		handleError(w, err)
		return
	}
	benefits := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		benefits = append(benefits, object(row))
	}
	reply(w, 200, map[string]any{"period_start": start, "period_end": end, "generated_at": generatedAt, "totals": totals, "benefits": benefits, "limit": 20})
}
