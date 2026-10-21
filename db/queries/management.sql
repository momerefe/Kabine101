-- name: ManagementGetBusiness :one
SELECT * FROM businesses WHERE id=$1;

-- name: ManagementLockBusiness :one
SELECT * FROM businesses WHERE id=$1 FOR UPDATE;

-- name: ManagementListBusiness :many
SELECT * FROM businesses
WHERE (sqlc.arg(q)::text='' OR strpos(lower(name||' '||description),lower(sqlc.arg(q)::text))>0)
AND (sqlc.arg(category)::text='' OR category=sqlc.arg(category)::text)
AND (sqlc.arg(status)::text='' OR partnership_status=sqlc.arg(status)::text)
ORDER BY updated_at DESC,id LIMIT 1001;

-- name: ManagementCreateBusiness :one
INSERT INTO businesses(id,name,category,description,image_path,contact_person,contact_info,branches,partnership_status,last_contact,notes) VALUES(sqlc.arg(id)::uuid,sqlc.arg(name)::text,sqlc.arg(category)::text,sqlc.arg(description)::text,nullif(sqlc.arg(image_path)::text, '')::text,nullif(sqlc.arg(contact_person)::text, '')::text,nullif(sqlc.arg(contact_info)::text, '')::text,nullif(sqlc.arg(branches)::text, '')::text,sqlc.arg(partnership_status)::text,nullif(sqlc.arg(last_contact)::text, '')::date,sqlc.arg(notes)::text) RETURNING *;

-- name: ManagementUpdateBusiness :one
UPDATE businesses SET name=sqlc.arg(name)::text,category=sqlc.arg(category)::text,description=sqlc.arg(description)::text,image_path=nullif(sqlc.arg(image_path)::text, '')::text,contact_person=nullif(sqlc.arg(contact_person)::text, '')::text,contact_info=nullif(sqlc.arg(contact_info)::text, '')::text,branches=nullif(sqlc.arg(branches)::text, '')::text,partnership_status=sqlc.arg(partnership_status)::text,last_contact=nullif(sqlc.arg(last_contact)::text, '')::date,notes=sqlc.arg(notes)::text,revision=revision+1,updated_at=now() WHERE id=sqlc.arg(id)::uuid AND revision=sqlc.arg(revision)::integer RETURNING *;

-- name: ManagementGetBenefit :one
SELECT * FROM benefits WHERE id=$1;

-- name: ManagementLockBenefit :one
SELECT * FROM benefits WHERE id=$1 FOR UPDATE;

-- name: ManagementListBenefit :many
SELECT benefits.* FROM benefits JOIN businesses ON businesses.id=benefits.business_id
WHERE (sqlc.arg(q)::text='' OR strpos(lower(title||' '||benefits.description||' '||perk||' '||businesses.name),lower(sqlc.arg(q)::text))>0)
AND (sqlc.arg(category)::text='' OR businesses.category=sqlc.arg(category)::text)
AND (sqlc.arg(status)::text='' OR
CASE sqlc.arg(status)::text
WHEN 'published' THEN publication_status='published'
WHEN 'archived' THEN publication_status='suspended' OR (publication_status='published' AND valid_until < (now() AT TIME ZONE 'Europe/Istanbul')::date)
ELSE sqlc.arg(status)::text=CASE WHEN publication_status<>'published' THEN publication_status WHEN valid_until<(now() AT TIME ZONE 'Europe/Istanbul')::date THEN 'expired' WHEN valid_from>(now() AT TIME ZONE 'Europe/Istanbul')::date THEN 'scheduled' ELSE 'active' END END)
ORDER BY benefits.updated_at DESC,benefits.id LIMIT 1001;

-- name: ManagementCreateBenefit :one
INSERT INTO benefits(id,business_id,title,description,perk,conditions,instructions,products,branches,exceptions,minimum_spend,currency,verification_requirement,valid_from,valid_until,method,code,target_url,qr_content,membership_label,support_contact,featured) VALUES(sqlc.arg(id)::uuid,sqlc.arg(business_id)::uuid,sqlc.arg(title)::text,sqlc.arg(description)::text,sqlc.arg(perk)::text,sqlc.arg(conditions)::text[],sqlc.arg(instructions)::text[],nullif(sqlc.arg(products)::text, '')::text,nullif(sqlc.arg(branches)::text, '')::text,nullif(sqlc.arg(exceptions)::text, '')::text,nullif(sqlc.arg(minimum_spend)::text, '')::numeric,sqlc.arg(currency)::text,nullif(sqlc.arg(verification_requirement)::text, '')::text,nullif(sqlc.arg(valid_from)::text, '')::date,nullif(sqlc.arg(valid_until)::text, '')::date,sqlc.arg(method)::text,nullif(sqlc.arg(code)::text, '')::text,nullif(sqlc.arg(target_url)::text, '')::text,nullif(sqlc.arg(qr_content)::text, '')::text,nullif(sqlc.arg(membership_label)::text, '')::text,nullif(sqlc.arg(support_contact)::text, '')::text,sqlc.arg(featured)::boolean) RETURNING *;

-- name: ManagementUpdateBenefit :one
UPDATE benefits SET business_id=sqlc.arg(business_id)::uuid,title=sqlc.arg(title)::text,description=sqlc.arg(description)::text,perk=sqlc.arg(perk)::text,conditions=sqlc.arg(conditions)::text[],instructions=sqlc.arg(instructions)::text[],products=nullif(sqlc.arg(products)::text, '')::text,branches=nullif(sqlc.arg(branches)::text, '')::text,exceptions=nullif(sqlc.arg(exceptions)::text, '')::text,minimum_spend=nullif(sqlc.arg(minimum_spend)::text, '')::numeric,currency=sqlc.arg(currency)::text,verification_requirement=nullif(sqlc.arg(verification_requirement)::text, '')::text,valid_from=nullif(sqlc.arg(valid_from)::text, '')::date,valid_until=nullif(sqlc.arg(valid_until)::text, '')::date,method=sqlc.arg(method)::text,code=nullif(sqlc.arg(code)::text, '')::text,target_url=nullif(sqlc.arg(target_url)::text, '')::text,qr_content=nullif(sqlc.arg(qr_content)::text, '')::text,membership_label=nullif(sqlc.arg(membership_label)::text, '')::text,support_contact=nullif(sqlc.arg(support_contact)::text, '')::text,featured=sqlc.arg(featured)::boolean,publication_status='draft',suspension_reason=NULL,reviewed_at=NULL,reviewed_by=NULL,published_at=NULL,revision=revision+1,updated_at=now() WHERE id=sqlc.arg(id)::uuid AND revision=sqlc.arg(revision)::integer RETURNING *;

-- name: ManagementPublish :one
UPDATE benefits SET publication_status='published',suspension_reason=NULL,reviewed_at=now(),reviewed_by=$2,published_at=now(),updated_at=now(),revision=revision+1 WHERE id=$1 RETURNING *;

-- name: ManagementSuspend :one
UPDATE benefits SET publication_status='suspended',suspension_reason=$2,reviewed_at=NULL,reviewed_by=NULL,updated_at=now(),revision=revision+1 WHERE id=$1 RETURNING *;

-- name: ManagementSuspendBusiness :many
UPDATE benefits SET publication_status='suspended',suspension_reason='İş ortaklığı aktif değil',reviewed_at=NULL,reviewed_by=NULL,updated_at=now(),revision=revision+1 WHERE business_id=$1 AND publication_status='published' RETURNING id;

-- name: ManagementAudit :exec
INSERT INTO audit_events(actor_id,action,entity_type,entity_id) VALUES($1,$2,$3,$4);
