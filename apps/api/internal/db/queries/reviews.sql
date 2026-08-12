-- name: InsertReview :one
INSERT INTO reviews (id, booking_id, villa_slug, author_name, rating, body, status, meta, source, featured)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListReviewsByVilla :many
SELECT * FROM reviews
WHERE villa_slug = $1
ORDER BY created_at DESC;

-- name: ListReviewsByVillaAndStatus :many
SELECT * FROM reviews
WHERE villa_slug = sqlc.arg('villa_slug')
  AND (sqlc.narg('status')::review_status IS NULL OR status = sqlc.narg('status')::review_status)
ORDER BY featured DESC, created_at DESC;

-- name: GetReview :one
SELECT * FROM reviews WHERE id = $1;

-- name: UpdateReview :one
UPDATE reviews
SET status     = COALESCE(sqlc.narg('status')::review_status, status),
    featured   = COALESCE(sqlc.narg('featured'), featured),
    meta       = COALESCE(sqlc.narg('meta'), meta),
    source     = COALESCE(sqlc.narg('source'), source),
    body       = COALESCE(sqlc.narg('body'), body),
    rating     = COALESCE(sqlc.narg('rating'), rating),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteReview :execrows
DELETE FROM reviews WHERE id = $1;

-- name: GetReviewMeta :one
SELECT * FROM villa_review_meta WHERE villa_slug = $1;

-- name: UpsertReviewMeta :one
INSERT INTO villa_review_meta (
    villa_slug, display_avg, display_count, cleanliness, comfort, location, host, value
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (villa_slug) DO UPDATE
SET display_avg   = EXCLUDED.display_avg,
    display_count = EXCLUDED.display_count,
    cleanliness   = EXCLUDED.cleanliness,
    comfort       = EXCLUDED.comfort,
    location      = EXCLUDED.location,
    host          = EXCLUDED.host,
    value         = EXCLUDED.value,
    updated_at    = NOW()
RETURNING *;
