-- name: OverviewBusinesses :one
SELECT count(*) AS total,
 count(*) FILTER (WHERE partnership_status='active') AS active,
 count(*) FILTER (WHERE partnership_status IN ('contacted','discussing','awaiting')) AS in_progress,
 count(*) FILTER (WHERE partnership_status='ended') AS ended FROM businesses;

-- name: OverviewBenefits :one
WITH classified AS (
 SELECT publication_status,valid_until, CASE WHEN publication_status<>'published' THEN publication_status
 WHEN valid_until<sqlc.arg(today)::date THEN 'expired'
 WHEN valid_from>sqlc.arg(today)::date THEN 'scheduled' ELSE 'active' END AS state FROM benefits
)
SELECT count(*) AS total,
 count(*) FILTER (WHERE publication_status='published') AS published,
 count(*) FILTER (WHERE state='active') AS active,
 count(*) FILTER (WHERE state='scheduled') AS scheduled,
 count(*) FILTER (WHERE state='expired') AS expired,
 count(*) FILTER (WHERE state='suspended') AS suspended,
 count(*) FILTER (WHERE state='draft') AS draft,
 count(*) FILTER (WHERE state='active' AND valid_until BETWEEN sqlc.arg(today)::date AND sqlc.arg(today)::date+6) AS ending_soon
FROM classified;

-- name: OverviewStudents :one
SELECT count(*) AS total,
 count(*) FILTER (WHERE status='active' AND eligibility_status='approved' AND email_verified_at IS NOT NULL) AS active,
 count(*) FILTER (WHERE status='pending' AND eligibility_status='pending') AS pending_review,
 count(*) FILTER (WHERE status NOT IN ('rejected','disabled') AND eligibility_status<>'rejected' AND email_verified_at IS NULL) AS pending_email,
 count(*) FILTER (WHERE status='pending' AND eligibility_status='pending' AND email_verified_at IS NOT NULL) AS review_ready
FROM accounts WHERE role='student';

-- name: OverviewExpiringBenefits :many
SELECT b.id,b.title,businesses.name AS business_name,b.valid_until
FROM benefits b JOIN businesses ON businesses.id=b.business_id
WHERE b.publication_status='published' AND (b.valid_from IS NULL OR b.valid_from<=sqlc.arg(today)::date)
 AND b.valid_until BETWEEN sqlc.arg(today)::date AND sqlc.arg(today)::date+6
ORDER BY b.valid_until,b.id LIMIT 5;

-- name: OverviewRecentActivity :many
SELECT e.id::text AS id,e.action,coalesce(actor.name,'Sistem')::text AS actor_name,e.entity_type,
 e.entity_id::text AS entity_id,coalesce(b.name,v.title,target.name,p.business_name,'')::text AS entity_label,e.created_at
FROM audit_events e
LEFT JOIN accounts actor ON actor.id=e.actor_id
LEFT JOIN businesses b ON e.entity_type='business' AND b.id=e.entity_id
LEFT JOIN benefits v ON e.entity_type='benefit' AND v.id=e.entity_id
LEFT JOIN accounts target ON e.entity_type='account' AND target.id=e.entity_id
LEFT JOIN purchases p ON e.entity_type='purchase' AND p.id=e.entity_id
WHERE e.entity_id IS NOT NULL AND (
 (e.entity_type='business' AND e.action='business.save') OR
 (e.entity_type='benefit' AND e.action IN ('benefit.save','benefit.publish','benefit.suspend')) OR
 (e.entity_type='account' AND e.action IN ('account.review.approved','account.review.rejected','account.role.admin','account.role.student')) OR
 (e.entity_type='purchase' AND e.action IN ('purchase.create','purchase.approved','purchase.rejected','purchase.revoked')))
ORDER BY e.created_at DESC,e.id DESC LIMIT 6;
