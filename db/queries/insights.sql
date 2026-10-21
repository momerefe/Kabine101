-- name: ManagementAuditList :many
SELECT e.id::text AS id,e.action,coalesce(actor.name,'Sistem')::text AS actor_name,e.entity_type,
 coalesce(e.entity_id::text,'')::text AS entity_id,coalesce(b.name,v.title,target.name,'')::text AS entity_label,e.created_at
FROM audit_events e
LEFT JOIN accounts actor ON actor.id=e.actor_id
LEFT JOIN businesses b ON e.entity_type='business' AND b.id=e.entity_id
LEFT JOIN benefits v ON e.entity_type='benefit' AND v.id=e.entity_id
LEFT JOIN accounts target ON e.entity_type='account' AND target.id=e.entity_id
WHERE (sqlc.arg(q)::text='' OR strpos(lower(e.action||' '||coalesce(actor.name,'Sistem')||' '||coalesce(b.name,v.title,target.name,'')),lower(sqlc.arg(q)::text))>0)
AND (sqlc.arg(action)::text='' OR e.action=sqlc.arg(action)::text)
AND (sqlc.narg(from_date)::date IS NULL OR e.created_at >= (sqlc.narg(from_date)::date::timestamp AT TIME ZONE 'Europe/Istanbul'))
AND (sqlc.narg(until_date)::date IS NULL OR e.created_at < ((sqlc.narg(until_date)::date+1)::timestamp AT TIME ZONE 'Europe/Istanbul'))
ORDER BY e.created_at DESC,e.id DESC LIMIT 50 OFFSET sqlc.arg(offset_rows)::integer;

-- name: ManagementAuditCount :one
SELECT count(*) FROM audit_events e
LEFT JOIN accounts actor ON actor.id=e.actor_id
LEFT JOIN businesses b ON e.entity_type='business' AND b.id=e.entity_id
LEFT JOIN benefits v ON e.entity_type='benefit' AND v.id=e.entity_id
LEFT JOIN accounts target ON e.entity_type='account' AND target.id=e.entity_id
WHERE (sqlc.arg(q)::text='' OR strpos(lower(e.action||' '||coalesce(actor.name,'Sistem')||' '||coalesce(b.name,v.title,target.name,'')),lower(sqlc.arg(q)::text))>0)
AND (sqlc.arg(action)::text='' OR e.action=sqlc.arg(action)::text)
AND (sqlc.narg(from_date)::date IS NULL OR e.created_at >= (sqlc.narg(from_date)::date::timestamp AT TIME ZONE 'Europe/Istanbul'))
AND (sqlc.narg(until_date)::date IS NULL OR e.created_at < ((sqlc.narg(until_date)::date+1)::timestamp AT TIME ZONE 'Europe/Istanbul'));

-- name: ManagementAnalyticsTotals :one
SELECT (SELECT count(*) FROM benefit_daily_views WHERE view_date BETWEEN sqlc.arg(period_start)::date AND sqlc.arg(period_end)::date) AS views,
 (SELECT count(*) FROM favorites) AS favorites,(SELECT count(*) FROM benefits) AS benefits;

-- name: ManagementAnalyticsBenefits :many
SELECT b.id,b.title,businesses.name AS business_name,
 (CASE WHEN b.publication_status<>'published' THEN b.publication_status WHEN businesses.partnership_status<>'active' THEN 'unavailable' WHEN b.valid_until<sqlc.arg(period_end)::date THEN 'expired' WHEN b.valid_from>sqlc.arg(period_end)::date THEN 'scheduled' ELSE 'active' END)::text AS status,
 coalesce(v.view_count,0)::bigint AS view_count,coalesce(f.favorite_count,0)::bigint AS favorite_count
FROM benefits b JOIN businesses ON businesses.id=b.business_id
LEFT JOIN (SELECT benefit_id,count(*) AS view_count FROM benefit_daily_views WHERE view_date BETWEEN sqlc.arg(period_start)::date AND sqlc.arg(period_end)::date GROUP BY benefit_id) v ON v.benefit_id=b.id
LEFT JOIN (SELECT benefit_id,count(*) AS favorite_count FROM favorites GROUP BY benefit_id) f ON f.benefit_id=b.id
ORDER BY view_count DESC,favorite_count DESC,b.id LIMIT 20;
