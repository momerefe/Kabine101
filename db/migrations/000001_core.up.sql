CREATE TABLE accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 120),
    email text NOT NULL UNIQUE CHECK (email = lower(email)),
    password_hash text NOT NULL,
    role text NOT NULL DEFAULT 'student' CHECK (role IN ('student', 'admin')),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'rejected', 'disabled')),
    email_verified_at timestamptz,
    eligibility_status text NOT NULL DEFAULT 'pending' CHECK (eligibility_status IN ('pending', 'approved', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Token material is never stored in plaintext. Authentication arrives in 3B.
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);
CREATE TABLE account_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    purpose text NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);

CREATE TABLE businesses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 160),
    category text NOT NULL CHECK (category IN ('food', 'fashion', 'tech', 'sport', 'culture', 'other')),
    description text NOT NULL DEFAULT '',
    image_path text,
    contact_person text,
    contact_info text,
    branches text,
    partnership_status text NOT NULL DEFAULT 'contacted' CHECK (partnership_status IN ('discussing', 'contacted', 'awaiting', 'active', 'ended')),
    last_contact date,
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE benefits (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id uuid NOT NULL REFERENCES businesses(id),
    title text NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 200),
    description text NOT NULL DEFAULT '',
    perk text NOT NULL DEFAULT '',
    conditions text[] NOT NULL DEFAULT '{}',
    instructions text[] NOT NULL DEFAULT '{}',
    products text,
    branches text,
    exceptions text,
    minimum_spend numeric(12,2) CHECK (minimum_spend >= 0),
    currency text NOT NULL DEFAULT 'TRY' CHECK (currency = 'TRY'),
    verification_requirement text,
    valid_from date,
    valid_until date,
    method text NOT NULL CHECK (method IN ('code', 'link', 'qr', 'membership')),
    code text,
    target_url text,
    qr_content text,
    membership_label text,
    support_contact text,
    publication_status text NOT NULL DEFAULT 'draft' CHECK (publication_status IN ('draft', 'published', 'suspended')),
    suspension_reason text,
    reviewed_at timestamptz,
    reviewed_by uuid REFERENCES accounts(id),
    published_at timestamptz,
    featured boolean NOT NULL DEFAULT false,
    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (valid_from IS NULL OR valid_until IS NULL OR valid_from <= valid_until),
    CHECK (target_url IS NULL OR target_url ~ '^https?://'),
    CHECK (publication_status <> 'published' OR (
        length(trim(description)) > 0 AND length(trim(perk)) > 0
        AND cardinality(conditions) > 0 AND cardinality(instructions) > 0
        AND reviewed_at IS NOT NULL AND reviewed_by IS NOT NULL AND published_at IS NOT NULL
        AND CASE method
            WHEN 'code' THEN length(trim(coalesce(code, ''))) > 0
            WHEN 'link' THEN length(trim(coalesce(target_url, ''))) > 0
            WHEN 'qr' THEN length(trim(coalesce(qr_content, ''))) > 0
            WHEN 'membership' THEN length(trim(coalesce(membership_label, ''))) > 0
        END
    ))
);
CREATE TABLE favorites (
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    benefit_id uuid NOT NULL REFERENCES benefits(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, benefit_id)
);
CREATE TABLE audit_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id uuid REFERENCES accounts(id) ON DELETE SET NULL,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE platform_settings (
    id smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    platform_name text NOT NULL DEFAULT 'KoçBenefits',
    support_contact text,
    show_featured boolean NOT NULL DEFAULT true
);
INSERT INTO platform_settings (id) VALUES (1);
-- Allowed email domain is deployment configuration, not a client-writable setting.
-- No redemption history table until actual business verification is agreed.
