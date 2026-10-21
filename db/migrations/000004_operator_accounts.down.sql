-- An operator has no email to restore. Refuse rather than delete an account
-- or invent an identity while rolling back this schema.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM accounts WHERE username IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove operator account support while operator accounts exist';
    END IF;
END $$;
ALTER TABLE accounts DROP CONSTRAINT accounts_identity_check;
DROP INDEX accounts_username_lower_key;
ALTER TABLE accounts DROP COLUMN username;
ALTER TABLE accounts ALTER COLUMN email SET NOT NULL;
