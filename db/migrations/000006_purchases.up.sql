-- Administrator-entered receipts become savings only after an explicit review.
CREATE TABLE purchases (
    id uuid PRIMARY KEY,
    student_id uuid NOT NULL REFERENCES accounts(id),
    business_id uuid NOT NULL REFERENCES businesses(id),
    benefit_id uuid NOT NULL REFERENCES benefits(id),
    business_name text NOT NULL,
    benefit_title text NOT NULL,
    purchase_date date NOT NULL,
    original_amount bigint NOT NULL CHECK (original_amount BETWEEN 1 AND 999999999),
    paid_amount bigint NOT NULL CHECK (paid_amount >= 0 AND paid_amount <= original_amount),
    currency text NOT NULL DEFAULT 'TRY' CHECK (currency = 'TRY'),
    reference text NOT NULL CHECK (length(trim(reference)) BETWEEN 1 AND 160),
    reference_normalized text NOT NULL CHECK (length(trim(reference_normalized)) BETWEEN 1 AND 160),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','revoked')),
    review_reason text NOT NULL DEFAULT '' CHECK (length(review_reason) <= 2000),
    created_by uuid NOT NULL REFERENCES accounts(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    reviewed_by uuid REFERENCES accounts(id),
    reviewed_at timestamptz,
    UNIQUE (business_id, reference_normalized),
    CHECK ((status = 'pending' AND reviewed_by IS NULL AND reviewed_at IS NULL)
        OR (status <> 'pending' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL)),
    CHECK (status NOT IN ('rejected','revoked') OR length(trim(review_reason)) > 0)
);
CREATE INDEX purchases_student_approved_date_idx ON purchases(student_id,purchase_date DESC,id) WHERE status='approved';
CREATE INDEX purchases_admin_created_idx ON purchases(created_at DESC,id);

-- Preserve every decision and its reason even when approval is later revoked.
CREATE TABLE purchase_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    purchase_id uuid NOT NULL REFERENCES purchases(id),
    actor_id uuid NOT NULL REFERENCES accounts(id),
    from_status text NOT NULL,
    to_status text NOT NULL CHECK (to_status IN ('pending','approved','rejected','revoked')),
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION protect_purchase_facts() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF ROW(NEW.id,NEW.student_id,NEW.business_id,NEW.benefit_id,NEW.business_name,NEW.benefit_title,
           NEW.purchase_date,NEW.original_amount,NEW.paid_amount,NEW.currency,NEW.reference,
           NEW.reference_normalized,NEW.created_by,NEW.created_at)
       IS DISTINCT FROM
       ROW(OLD.id,OLD.student_id,OLD.business_id,OLD.benefit_id,OLD.business_name,OLD.benefit_title,
           OLD.purchase_date,OLD.original_amount,OLD.paid_amount,OLD.currency,OLD.reference,
           OLD.reference_normalized,OLD.created_by,OLD.created_at) THEN
        RAISE EXCEPTION 'purchase facts are immutable' USING ERRCODE='23514';
    END IF;
    IF NOT ((OLD.status='pending' AND NEW.status IN ('approved','rejected'))
         OR (OLD.status='approved' AND NEW.status='revoked')) THEN
        RAISE EXCEPTION 'invalid purchase transition' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER purchases_immutable_facts BEFORE UPDATE ON purchases
FOR EACH ROW EXECUTE FUNCTION protect_purchase_facts();
