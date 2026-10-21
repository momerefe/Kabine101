-- Bound token lookups and make account-wide revocation inexpensive.
CREATE INDEX account_tokens_account_purpose_idx ON account_tokens(account_id, purpose);
CREATE INDEX accounts_pending_review_idx ON accounts(created_at) WHERE eligibility_status = 'pending';
