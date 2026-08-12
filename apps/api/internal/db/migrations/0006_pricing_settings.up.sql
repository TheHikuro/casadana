-- Per-villa base rate and fees. The admin dashboard edits these; the public
-- quote falls back to base_price_cents whenever no season rule or per-date
-- override matches.
CREATE TABLE villa_pricing_settings (
    villa_slug          TEXT PRIMARY KEY,
    base_price_cents    INTEGER NOT NULL DEFAULT 0 CHECK (base_price_cents >= 0),
    min_nights          SMALLINT NOT NULL DEFAULT 1 CHECK (min_nights >= 1),
    cleaning_fee_cents  INTEGER NOT NULL DEFAULT 0 CHECK (cleaning_fee_cents >= 0),
    concierge_fee_cents INTEGER NOT NULL DEFAULT 0 CHECK (concierge_fee_cents >= 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Named date-range rules ("Summer peak"). Kept as ranges rather than expanded
-- into price_overrides so the admin can still edit or delete a rule as a unit.
CREATE TABLE season_rules (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    villa_slug  TEXT NOT NULL,
    label       TEXT NOT NULL,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    price_cents INTEGER NOT NULL CHECK (price_cents >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT season_rules_range_valid CHECK (end_date >= start_date)
);

CREATE INDEX season_rules_villa_idx ON season_rules (villa_slug, start_date);
