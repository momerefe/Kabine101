-- name: GetPlatformName :one
SELECT platform_name FROM platform_settings WHERE id = 1;

-- name: ManagementSettings :one
SELECT * FROM platform_settings WHERE id=1;

-- name: ManagementUpdateSettings :one
UPDATE platform_settings SET platform_name=sqlc.arg(platform_name)::text,
 support_contact=nullif(sqlc.arg(support_contact)::text,''),show_featured=sqlc.arg(show_featured)::boolean,
 revision=revision+1 WHERE id=1 AND revision=sqlc.arg(revision)::integer RETURNING *;
