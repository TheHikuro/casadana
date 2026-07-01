-- name: ListPriceOverrides :many
SELECT villa_slug, date, price_cents
FROM price_overrides
WHERE villa_slug = sqlc.arg('villa_slug')
  AND date >= sqlc.arg('from')
  AND date < sqlc.arg('to')
ORDER BY date;

-- name: UpsertPriceOverride :exec
INSERT INTO price_overrides (villa_slug, date, price_cents)
VALUES ($1, $2, $3)
ON CONFLICT (villa_slug, date) DO UPDATE
SET price_cents = EXCLUDED.price_cents,
    updated_at = NOW();
