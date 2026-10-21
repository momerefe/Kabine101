package management

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// overview reads every panel from the same database snapshot and Istanbul day.
func (s *Service) overview(w http.ResponseWriter, r *http.Request) {
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
	// PostgreSQL transaction time is stable across every query in this snapshot.
	var generatedAt time.Time
	if err = tx.QueryRow(ctx, "SELECT transaction_timestamp()").Scan(&generatedAt); err != nil {
		handleError(w, err)
		return
	}
	day, err := time.Parse("2006-01-02", generatedAt.In(istanbul).Format("2006-01-02"))
	if err != nil {
		handleError(w, err)
		return
	}
	today := pgtype.Date{Time: day, Valid: true}
	q := s.q.WithTx(tx)
	businesses, err := q.OverviewBusinesses(ctx)
	if err != nil {
		handleError(w, err)
		return
	}
	benefits, err := q.OverviewBenefits(ctx, today)
	if err != nil {
		handleError(w, err)
		return
	}
	students, err := q.OverviewStudents(ctx)
	if err != nil {
		handleError(w, err)
		return
	}
	expiring, err := q.OverviewExpiringBenefits(ctx, today)
	if err != nil {
		handleError(w, err)
		return
	}
	activity, err := q.OverviewRecentActivity(ctx)
	if err != nil {
		handleError(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		handleError(w, err)
		return
	}
	// sqlc returns nil for empty result sets; the response contract uses arrays.
	expiringJSON := make([]map[string]any, 0, len(expiring))
	for _, row := range expiring {
		expiringJSON = append(expiringJSON, object(row))
	}
	activityJSON := make([]map[string]any, 0, len(activity))
	for _, row := range activity {
		activityJSON = append(activityJSON, object(row))
	}
	reply(w, http.StatusOK, map[string]any{"generated_at": generatedAt, "businesses": businesses, "benefits": benefits, "students": students, "expiring_benefits": expiringJSON, "recent_activity": activityJSON})
}
