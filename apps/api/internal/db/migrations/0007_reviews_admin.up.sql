-- Admin-authored reviews (the dashboard's "Add review" form) have no booking
-- behind them. Postgres allows many NULLs under a UNIQUE constraint, so the
-- existing one-review-per-booking rule still holds for guest submissions.
ALTER TABLE reviews ALTER COLUMN booking_id DROP NOT NULL;

ALTER TABLE reviews
    ADD COLUMN featured BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN meta     TEXT    NOT NULL DEFAULT '',
    ADD COLUMN source   TEXT    NOT NULL DEFAULT '';

CREATE INDEX reviews_featured_idx ON reviews (villa_slug, featured);

-- Editorial figures shown on the public villa page. Kept separate from the
-- computed average so the owner can present a curated rating and per-category
-- breakdown independently of what is stored in `reviews`.
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
