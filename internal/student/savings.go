package student

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

func savingsAmount(value int64) string { return fmt.Sprintf("%d.%02d", value/100, value%100) }

// Every response uses one complete aggregate, independent of capped purchase history.
func (s *Service) savings(w http.ResponseWriter, r *http.Request) {
	account, _, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		dbfail(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var now time.Time
	if err = tx.QueryRow(ctx, "SELECT transaction_timestamp()").Scan(&now); err != nil {
		dbfail(w, err)
		return
	}
	month := now.In(istanbul).Format("2006-01")
	start, _ := time.Parse("2006-01-02", month+"-01")
	rows, err := s.q.WithTx(tx).StudentSavingsBusinesses(ctx, database.StudentSavingsBusinessesParams{
		StudentID: account, MonthStart: pgtype.Date{Time: start, Valid: true},
		MonthEnd: pgtype.Date{Time: start.AddDate(0, 1, 0), Valid: true},
	})
	if err != nil {
		dbfail(w, err)
		return
	}
	var total, count int64
	businesses := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row.SavingsAmount > math.MaxInt64-total || row.PurchaseCount > math.MaxInt64-count {
			fail(w, 503, "unavailable")
			return
		}
		total += row.SavingsAmount
		count += row.PurchaseCount
		businesses = append(businesses, map[string]any{
			"business_id": row.BusinessID, "business_name": row.BusinessName,
			"purchase_count": row.PurchaseCount, "savings_amount": savingsAmount(row.SavingsAmount),
		})
	}
	if err = tx.Commit(ctx); err != nil {
		dbfail(w, err)
		return
	}
	reply(w, 200, map[string]any{
		"month": month, "currency": "TRY", "total_savings": savingsAmount(total),
		"purchase_count": count, "businesses": businesses,
	})
}

func (s *Service) purchases(w http.ResponseWriter, r *http.Request) {
	account, _, ok := s.auth.RequireAccount(w, r)
	if !ok {
		return
	}
	rows, err := s.q.StudentPurchases(r.Context(), account)
	if err != nil {
		dbfail(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for i, row := range rows {
		if i == 1000 {
			break
		}
		// Deliberate owner-only allowlist: no references, review notes or student details.
		out = append(out, map[string]any{
			"id": row.ID, "business_name": row.BusinessName, "benefit_title": row.BenefitTitle,
			"purchase_date": date(row.PurchaseDate), "original_amount": savingsAmount(row.OriginalAmount),
			"paid_amount": savingsAmount(row.PaidAmount), "savings_amount": savingsAmount(row.OriginalAmount - row.PaidAmount),
			"currency": row.Currency, "status": row.Status,
		})
	}
	reply(w, 200, map[string]any{"purchases": out, "truncated": len(rows) > 1000})
}
