-- name: StudentCatalog :many
SELECT sqlc.embed(benefits), sqlc.embed(businesses), EXISTS(SELECT 1 FROM favorites f WHERE f.account_id=sqlc.arg(account_id)::uuid AND f.benefit_id=benefits.id) AS is_favorite
FROM benefits JOIN businesses ON businesses.id=benefits.business_id
WHERE benefits.publication_status='published' AND businesses.partnership_status='active'
AND (sqlc.arg(q)::text='' OR strpos(lower(benefits.title||' '||benefits.description||' '||benefits.perk||' '||businesses.name),lower(sqlc.arg(q)::text))>0)
AND (sqlc.arg(category)::text='' OR businesses.category=sqlc.arg(category)::text)
ORDER BY (benefits.featured AND (SELECT show_featured FROM platform_settings WHERE id=1)) DESC, benefits.updated_at DESC, benefits.id LIMIT 1001;

-- name: StudentFavorites :many
SELECT sqlc.embed(benefits), sqlc.embed(businesses)
FROM favorites JOIN benefits ON benefits.id=favorites.benefit_id JOIN businesses ON businesses.id=benefits.business_id
WHERE favorites.account_id=$1 ORDER BY favorites.created_at DESC, benefits.id LIMIT 1001;

-- name: StudentHasFavorite :one
SELECT EXISTS(SELECT 1 FROM favorites WHERE account_id=$1 AND benefit_id=$2);

-- name: StudentAddFavorite :exec
INSERT INTO favorites(account_id,benefit_id) VALUES($1,$2) ON CONFLICT DO NOTHING;

-- name: StudentRemoveFavorite :exec
DELETE FROM favorites WHERE account_id=$1 AND benefit_id=$2;

-- name: StudentSettings :one
SELECT platform_name,support_contact,show_featured FROM platform_settings WHERE id=1;

-- name: StudentRecordView :exec
INSERT INTO benefit_daily_views(account_id,benefit_id,view_date)
VALUES($1,$2,(now() AT TIME ZONE 'Europe/Istanbul')::date) ON CONFLICT DO NOTHING;
