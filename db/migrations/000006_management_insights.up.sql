ALTER TABLE platform_settings ADD COLUMN revision integer NOT NULL DEFAULT 1 CHECK (revision > 0);

-- One eligible student per benefit per Istanbul calendar day. No request metadata.
CREATE TABLE benefit_daily_views (
    account_id uuid NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    benefit_id uuid NOT NULL REFERENCES benefits(id) ON DELETE CASCADE,
    view_date date NOT NULL,
    PRIMARY KEY (account_id, benefit_id, view_date)
);
CREATE INDEX benefit_daily_views_date_benefit_idx ON benefit_daily_views(view_date, benefit_id);
CREATE INDEX benefit_daily_views_benefit_idx ON benefit_daily_views(benefit_id);
CREATE INDEX audit_events_created_id_idx ON audit_events(created_at DESC, id DESC);
