-- name: GetPricingSettings :one
SELECT * FROM villa_pricing_settings
WHERE villa_slug = $1;

-- name: UpsertPricingSettings :one
INSERT INTO villa_pricing_settings (
    villa_slug, base_price_cents, min_nights, cleaning_fee_cents, concierge_fee_cents
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (villa_slug) DO UPDATE
SET base_price_cents    = EXCLUDED.base_price_cents,
    min_nights          = EXCLUDED.min_nights,
    cleaning_fee_cents  = EXCLUDED.cleaning_fee_cents,
    concierge_fee_cents = EXCLUDED.concierge_fee_cents,
    updated_at          = NOW()
RETURNING *;

-- name: ListSeasonRules :many
SELECT * FROM season_rules
WHERE villa_slug = $1
ORDER BY start_date;

-- name: GetSeasonRule :one
SELECT * FROM season_rules
WHERE id = $1;

-- name: InsertSeasonRule :one
INSERT INTO season_rules (id, villa_slug, label, start_date, end_date, price_cents)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateSeasonRule :one
UPDATE season_rules
SET label       = COALESCE(sqlc.narg('label'), label),
    start_date  = COALESCE(sqlc.narg('start_date'), start_date),
    end_date    = COALESCE(sqlc.narg('end_date'), end_date),
    price_cents = COALESCE(sqlc.narg('price_cents'), price_cents),
    updated_at  = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteSeasonRule :execrows
DELETE FROM season_rules WHERE id = $1;
