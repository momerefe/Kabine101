package management

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/momerefe/Kabine101/internal/database"
)

type PurchaseWrite struct {
	ID             string `json:"id"`
	StudentEmail   string `json:"student_email"`
	BenefitID      string `json:"benefit_id"`
	PurchaseDate   string `json:"purchase_date"`
	OriginalAmount string `json:"original_amount"`
	PaidAmount     string `json:"paid_amount"`
	Reference      string `json:"reference"`
}

var purchaseAmountPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,6})(\.[0-9]{1,2})?$`)

// Amounts cross the HTTP boundary as decimal strings and never as floating point.
func parsePurchaseAmount(value string) (int64, bool) {
	if !purchaseAmountPattern.MatchString(value) {
		return 0, false
	}
	parts := strings.SplitN(value, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	var fraction int64
	if len(parts) == 2 {
		fraction, err = strconv.ParseInt(parts[1]+strings.Repeat("0", 2-len(parts[1])), 10, 64)
	}
	return whole*100 + fraction, err == nil
}

func purchaseAmount(value int64) string { return fmt.Sprintf("%d.%02d", value/100, value%100) }

func normalizePurchaseReference(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func purchaseDTO(v database.Purchase, name, email string) map[string]any {
	reviewedAt := ""
	if v.ReviewedAt.Valid {
		reviewedAt = v.ReviewedAt.Time.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"id": v.ID, "student_id": v.StudentID, "student_name": name, "student_email": email,
		"business_id": v.BusinessID, "business_name": v.BusinessName,
		"benefit_id": v.BenefitID, "benefit_title": v.BenefitTitle,
		"purchase_date":   v.PurchaseDate.Time.Format("2006-01-02"),
		"original_amount": purchaseAmount(v.OriginalAmount), "paid_amount": purchaseAmount(v.PaidAmount),
		"savings_amount": purchaseAmount(v.OriginalAmount - v.PaidAmount), "currency": v.Currency,
		"status": v.Status, "reference": v.Reference, "review_reason": v.ReviewReason,
		"created_at": v.CreatedAt.Time.UTC().Format(time.RFC3339Nano), "reviewed_at": reviewedAt,
	}
}

func (s *Service) purchases(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.auth.RequireAdmin(w, r); !ok {
		return
	}
	status := r.URL.Query().Get("status")
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if !oneOf(status, "", "pending", "approved", "rejected", "revoked") || !validText(search, 500, false) {
		fail(w, 400, "invalid_request")
		return
	}
	rows, err := s.q.ManagementListPurchases(r.Context(), database.ManagementListPurchasesParams{Status: status, Q: search})
	if err != nil {
		handleError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for i, row := range rows {
		if i == 1000 {
			break
		}
		out = append(out, purchaseDTO(row.Purchase, row.StudentName, row.StudentEmail.String))
	}
	reply(w, 200, map[string]any{"purchases": out, "truncated": len(rows) > 1000})
}

// Locks follow account -> business -> benefit -> purchase, as in benefit management.
// The student-role predicate avoids locking another administrator as a receipt owner.
func (s *Service) purchaseEligibility(w http.ResponseWriter, r *http.Request, q *database.Queries, student database.Account, benefitID pgtype.UUID, day string) (database.Business, database.Benefit, bool) {
	var business database.Business
	var benefit database.Benefit
	if !s.auth.EligibleStudent(student) {
		fail(w, 400, "invalid_request")
		return business, benefit, false
	}
	prior, err := q.ManagementGetBenefit(r.Context(), benefitID)
	if err != nil {
		handleError(w, err)
		return business, benefit, false
	}
	business, err = q.ManagementLockBusiness(r.Context(), prior.BusinessID)
	if err != nil {
		handleError(w, err)
		return business, benefit, false
	}
	benefit, err = q.ManagementLockBenefit(r.Context(), benefitID)
	if err != nil {
		handleError(w, err)
		return business, benefit, false
	}
	// A concurrent benefit edit can move it to a different business before its lock.
	if benefit.BusinessID != business.ID {
		fail(w, 409, "conflict")
		return business, benefit, false
	}
	if business.PartnershipStatus != "active" || benefit.PublicationStatus != "published" ||
		(benefit.ValidFrom.Valid && benefit.ValidFrom.Time.Format("2006-01-02") > day) ||
		(benefit.ValidUntil.Valid && benefit.ValidUntil.Time.Format("2006-01-02") < day) {
		fail(w, 400, "invalid_request")
		return business, benefit, false
	}
	return business, benefit, true
}

func purchaseAudit(q *database.Queries, r *http.Request, actor pgtype.UUID, v database.Purchase, prior, action string) error {
	if err := q.ManagementPurchaseEvent(r.Context(), database.ManagementPurchaseEventParams{
		PurchaseID: v.ID, ActorID: actor, FromStatus: prior, ToStatus: v.Status, Reason: v.ReviewReason,
	}); err != nil {
		return err
	}
	return audit(q, r, actor, action, "purchase", v.ID)
}

func (s *Service) createPurchase(w http.ResponseWriter, r *http.Request) {
	var in PurchaseWrite
	if !decode(w, r, &in) {
		return
	}
	var id, benefitID pgtype.UUID
	original, validOriginal := parsePurchaseAmount(in.OriginalAmount)
	paid, validPaid := parsePurchaseAmount(in.PaidAmount)
	in.StudentEmail = strings.ToLower(strings.TrimSpace(in.StudentEmail))
	in.Reference = strings.TrimSpace(in.Reference)
	if id.Scan(in.ID) != nil || !id.Valid || benefitID.Scan(in.BenefitID) != nil || !benefitID.Valid ||
		!validOriginal || !validPaid || original <= 0 || paid > original ||
		in.PurchaseDate < "0001-01-01" || !validDate(in.PurchaseDate) || in.PurchaseDate > time.Now().In(istanbul).Format("2006-01-02") ||
		!validText(in.StudentEmail, 254, true) || !validText(in.Reference, 160, true) {
		fail(w, 400, "invalid_request")
		return
	}
	tx, q, actor, ok := s.begin(w, r)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize absent IDs too, so simultaneous retries cannot create duplicate events.
	if err := q.ManagementSerializePurchaseID(r.Context(), id.String()); err != nil {
		handleError(w, err)
		return
	}
	prior, err := q.ManagementGetPurchase(r.Context(), id)
	if err == nil {
		p := prior.Purchase
		if prior.StudentEmail.String != in.StudentEmail || p.BenefitID != benefitID || p.PurchaseDate.Time.Format("2006-01-02") != in.PurchaseDate ||
			p.OriginalAmount != original || p.PaidAmount != paid || p.ReferenceNormalized != normalizePurchaseReference(in.Reference) {
			fail(w, 409, "conflict")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			handleError(w, err)
			return
		}
		reply(w, 200, map[string]any{"purchase": purchaseDTO(p, prior.StudentName, prior.StudentEmail.String)})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		handleError(w, err)
		return
	}
	student, err := q.ManagementLockPurchaseStudent(r.Context(), in.StudentEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 400, "invalid_request")
		return
	}
	if err != nil {
		handleError(w, err)
		return
	}
	business, benefit, ok := s.purchaseEligibility(w, r, q, student, benefitID, in.PurchaseDate)
	if !ok {
		return
	}
	day, _ := time.Parse("2006-01-02", in.PurchaseDate)
	v, err := q.ManagementCreatePurchase(r.Context(), database.ManagementCreatePurchaseParams{
		ID: id, StudentID: student.ID, BusinessID: business.ID, BenefitID: benefit.ID,
		BusinessName: business.Name, BenefitTitle: benefit.Title, PurchaseDate: pgtype.Date{Time: day, Valid: true},
		OriginalAmount: original, PaidAmount: paid, Reference: in.Reference,
		ReferenceNormalized: normalizePurchaseReference(in.Reference), CreatedBy: actor,
	})
	if err == nil {
		err = purchaseAudit(q, r, actor, v, "", "purchase.create")
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		handleError(w, err)
		return
	}
	reply(w, 201, map[string]any{"purchase": purchaseDTO(v, student.Name, student.Email.String)})
}

func (s *Service) reviewPurchase(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	id, ok := identifier(w, r)
	if !ok {
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if !oneOf(in.Decision, "approved", "rejected", "revoked") || !validText(in.Reason, 2000, in.Decision != "approved") {
		fail(w, 400, "invalid_request")
		return
	}
	tx, q, actor, ok := s.begin(w, r)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context())
	prior, err := q.ManagementGetPurchase(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	if !((prior.Purchase.Status == "pending" && oneOf(in.Decision, "approved", "rejected")) || (prior.Purchase.Status == "approved" && in.Decision == "revoked")) {
		fail(w, 409, "conflict")
		return
	}
	if in.Decision == "approved" {
		student, err := q.ManagementLockPurchaseStudentID(r.Context(), prior.Purchase.StudentID)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 400, "invalid_request")
			return
		}
		if err != nil {
			handleError(w, err)
			return
		}
		business, _, ok := s.purchaseEligibility(w, r, q, student, prior.Purchase.BenefitID, prior.Purchase.PurchaseDate.Time.Format("2006-01-02"))
		if !ok {
			return
		}
		// Reassignment cannot silently credit a receipt to a different business.
		if business.ID != prior.Purchase.BusinessID {
			fail(w, 409, "conflict")
			return
		}
	}
	v, err := q.ManagementLockPurchase(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	if !((v.Status == "pending" && oneOf(in.Decision, "approved", "rejected")) || (v.Status == "approved" && in.Decision == "revoked")) {
		fail(w, 409, "conflict")
		return
	}
	previousStatus := v.Status
	v, err = q.ManagementReviewPurchase(r.Context(), database.ManagementReviewPurchaseParams{
		ID: id, Status: in.Decision, Reason: in.Reason, ActorID: actor, PriorStatus: previousStatus,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "conflict")
		return
	}
	if err == nil {
		err = purchaseAudit(q, r, actor, v, previousStatus, "purchase."+in.Decision)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		handleError(w, err)
		return
	}
	reply(w, 200, map[string]any{"purchase": purchaseDTO(v, prior.StudentName, prior.StudentEmail.String)})
}
