-- name: ManagementBenefitImage :one
SELECT content_type,data FROM benefit_images JOIN benefits ON benefits.id=benefit_images.benefit_id
WHERE benefit_id=$1 AND benefits.has_image;

-- name: StudentBenefitImage :one
SELECT content_type,data FROM benefit_images JOIN benefits ON benefits.id=benefit_images.benefit_id
JOIN businesses ON businesses.id=benefits.business_id
WHERE benefit_id=$1 AND benefits.has_image AND benefits.publication_status='published' AND businesses.partnership_status='active';

-- name: SaveBenefitImage :exec
INSERT INTO benefit_images(benefit_id,content_type,data) VALUES($1,$2,$3)
ON CONFLICT (benefit_id) DO UPDATE SET content_type=EXCLUDED.content_type,data=EXCLUDED.data;

-- name: DeleteBenefitImage :exec
DELETE FROM benefit_images WHERE benefit_id=$1;

-- name: SetBenefitHasImage :exec
UPDATE benefits SET has_image=$2 WHERE id=$1;
