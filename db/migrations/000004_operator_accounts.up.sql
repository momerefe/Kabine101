-- Email-free operators are provisioned privately; public registration remains email-only.
ALTER TABLE accounts ALTER COLUMN email DROP NOT NULL;
ALTER TABLE accounts ADD COLUMN username text;
CREATE UNIQUE INDEX accounts_username_lower_key ON accounts (lower(username)) WHERE username IS NOT NULL;
ALTER TABLE accounts ADD CONSTRAINT accounts_identity_check CHECK (
    (username IS NULL AND email IS NOT NULL AND length(trim(email)) > 0)
    OR
    (username IS NOT NULL AND username ~ '^[A-Za-z][A-Za-z0-9_]{2,31}$'
     AND email IS NULL AND role = 'admin' AND email_verified_at IS NULL
     AND eligibility_status = 'approved' AND status IN ('active', 'disabled'))
);
