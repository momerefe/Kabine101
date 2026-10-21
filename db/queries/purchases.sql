-- name: ManagementListPurchases :many
SELECT sqlc.embed(p),a.name AS student_name,a.email AS student_email
FROM purchases p JOIN accounts a ON a.id=p.student_id
WHERE (sqlc.arg(status)::text='' OR p.status=sqlc.arg(status)::text)
 AND (sqlc.arg(q)::text='' OR strpos(lower(a.name||' '||coalesce(a.email,'')||' '||p.business_name||' '||p.benefit_title||' '||p.reference),lower(sqlc.arg(q)::text))>0)
ORDER BY p.created_at DESC,p.id LIMIT 1001;

-- name: ManagementGetPurchase :one
SELECT sqlc.embed(p),a.name AS student_name,a.email AS student_email
FROM purchases p JOIN accounts a ON a.id=p.student_id WHERE p.id=$1;

-- name: ManagementLockPurchase :one
SELECT * FROM purchases WHERE id=$1 FOR UPDATE;

-- name: ManagementLockPurchaseStudent :one
SELECT * FROM accounts WHERE email=sqlc.arg(email)::text AND role='student' FOR UPDATE;

-- name: ManagementLockPurchaseStudentID :one
SELECT * FROM accounts WHERE id=$1 AND role='student' FOR UPDATE;

-- name: ManagementSerializePurchaseID :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(id)::text::uuid::text,0));

-- name: ManagementCreatePurchase :one
INSERT INTO purchases(id,student_id,business_id,benefit_id,business_name,benefit_title,purchase_date,original_amount,paid_amount,reference,reference_normalized,created_by)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING *;

-- name: ManagementReviewPurchase :one
UPDATE purchases SET status=sqlc.arg(status)::text,review_reason=sqlc.arg(reason)::text,
 reviewed_by=sqlc.arg(actor_id)::uuid,reviewed_at=now()
WHERE id=sqlc.arg(id)::uuid AND status=sqlc.arg(prior_status)::text
RETURNING *;

-- name: ManagementPurchaseEvent :exec
INSERT INTO purchase_events(purchase_id,actor_id,from_status,to_status,reason) VALUES($1,$2,$3,$4,$5);

-- name: StudentPurchases :many
SELECT id,business_name,benefit_title,purchase_date,original_amount,paid_amount,currency,status
FROM purchases WHERE student_id=$1 AND status='approved'
ORDER BY purchase_date DESC,created_at DESC,id LIMIT 1001;

-- name: StudentSavingsBusinesses :many
SELECT business_id, (array_agg(business_name ORDER BY purchase_date DESC,created_at DESC,id))[1]::text AS business_name,
 count(*)::bigint AS purchase_count,sum(original_amount-paid_amount)::bigint AS savings_amount
FROM purchases
WHERE student_id=sqlc.arg(student_id)::uuid AND status='approved'
 AND purchase_date>=sqlc.arg(month_start)::date AND purchase_date<sqlc.arg(month_end)::date
GROUP BY business_id ORDER BY savings_amount DESC,business_id;
