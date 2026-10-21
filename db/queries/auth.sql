-- name: FindAccountByEmail :one
SELECT * FROM accounts WHERE email = $1;

-- name: FindAccountByUsername :one
SELECT * FROM accounts WHERE lower(username) = lower(sqlc.arg(username)::text);

-- name: FindAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: LockAccount :one
SELECT * FROM accounts WHERE id = $1 FOR UPDATE;

-- name: CreateAccount :one
INSERT INTO accounts (name,email,password_hash,role,status,eligibility_status)
VALUES ($1,$2,$3,$4,$5,$6) RETURNING *;

-- name: CreateOperator :one
INSERT INTO accounts (name,username,password_hash,role,status,eligibility_status)
VALUES ($1,$2,$3,'admin','active','approved') RETURNING *;

-- name: ListAccounts :many
SELECT * FROM accounts ORDER BY created_at DESC;

-- name: CreateSession :exec
INSERT INTO sessions(token_hash,account_id,expires_at) VALUES ($1,$2,$3);

-- name: SessionAccount :one
SELECT a.* FROM accounts a JOIN sessions s ON s.account_id=a.id
WHERE s.token_hash=$1 AND s.expires_at>now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash=$1;

-- name: RevokeAccountSessions :exec
DELETE FROM sessions WHERE account_id=$1;

-- name: CreateAccountToken :exec
INSERT INTO account_tokens(token_hash,account_id,purpose,expires_at) VALUES ($1,$2,$3,$4);

-- name: FindAccountToken :one
SELECT * FROM account_tokens WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>now();

-- name: ConsumeAccountToken :one
UPDATE account_tokens SET consumed_at=now() WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>now() RETURNING *;

-- name: InvalidateAccountTokens :exec
UPDATE account_tokens SET consumed_at=now() WHERE account_id=$1 AND purpose=$2 AND consumed_at IS NULL;

-- name: VerifyAccountEmail :exec
UPDATE accounts SET email_verified_at=now(),updated_at=now() WHERE id=$1;

-- name: ChangePassword :exec
UPDATE accounts SET password_hash=$2,updated_at=now() WHERE id=$1;

-- name: ReviewAccount :exec
UPDATE accounts SET status=$2,eligibility_status=$3,updated_at=now() WHERE id=$1;

-- name: ChangeRole :exec
UPDATE accounts SET role=$2,updated_at=now() WHERE id=$1;

-- name: CountActiveAdmins :one
SELECT count(*) FROM accounts WHERE role='admin' AND status='active' AND eligibility_status='approved'
AND ((username IS NOT NULL AND email IS NULL AND email_verified_at IS NULL)
     OR (username IS NULL AND email_verified_at IS NOT NULL AND split_part(email,'@',2) = sqlc.arg(school_domain)::text));

-- name: FindSession :one
SELECT * FROM sessions WHERE token_hash=$1;

-- name: InsertAuthAudit :exec
INSERT INTO audit_events(actor_id,action,entity_type,entity_id) VALUES ($1,$2,'account',$3);
