ALTER TABLE benefits ADD COLUMN has_image boolean NOT NULL DEFAULT false;
CREATE TABLE benefit_images (
    benefit_id uuid PRIMARY KEY REFERENCES benefits(id) ON DELETE CASCADE,
    content_type text NOT NULL CHECK (content_type IN ('image/png', 'image/jpeg')),
    data bytea NOT NULL CHECK (octet_length(data) BETWEEN 1 AND 5242880)
);
