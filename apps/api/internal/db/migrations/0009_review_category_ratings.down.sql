-- Recreated empty: the curated figures that lived here were replaced by values
-- computed from the reviews, and rolling back cannot invent them again. A villa
-- reads back as zeros until someone re-enters them.
CREATE TABLE villa_review_meta (
    villa_slug    TEXT PRIMARY KEY,
    display_avg   NUMERIC(3, 2) NOT NULL DEFAULT 0 CHECK (display_avg >= 0 AND display_avg <= 5),
    display_count INTEGER       NOT NULL DEFAULT 0 CHECK (display_count >= 0),
    cleanliness   NUMERIC(3, 2) NOT NULL DEFAULT 0 CHECK (cleanliness >= 0 AND cleanliness <= 5),
    comfort       NUMERIC(3, 2) NOT NULL DEFAULT 0 CHECK (comfort >= 0 AND comfort <= 5),
    location      NUMERIC(3, 2) NOT NULL DEFAULT 0 CHECK (location >= 0 AND location <= 5),
    host          NUMERIC(3, 2) NOT NULL DEFAULT 0 CHECK (host >= 0 AND host <= 5),
    value         NUMERIC(3, 2) NOT NULL DEFAULT 0 CHECK (value >= 0 AND value <= 5),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

ALTER TABLE reviews
    DROP COLUMN rating_cleanliness,
    DROP COLUMN rating_comfort,
    DROP COLUMN rating_location,
    DROP COLUMN rating_host,
    DROP COLUMN rating_value;
