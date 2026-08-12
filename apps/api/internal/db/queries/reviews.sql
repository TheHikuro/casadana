-- name: InsertReview :one
INSERT INTO reviews (
    id, booking_id, villa_slug, author_name, rating, body, status, meta, source, featured,
    rating_cleanliness, rating_comfort, rating_location, rating_host, rating_value
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
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
SET status             = COALESCE(sqlc.narg('status')::review_status, status),
    featured           = COALESCE(sqlc.narg('featured'), featured),
    meta               = COALESCE(sqlc.narg('meta'), meta),
    source             = COALESCE(sqlc.narg('source'), source),
    body               = COALESCE(sqlc.narg('body'), body),
    rating             = COALESCE(sqlc.narg('rating'), rating),
    rating_cleanliness = COALESCE(sqlc.narg('rating_cleanliness'), rating_cleanliness),
    rating_comfort     = COALESCE(sqlc.narg('rating_comfort'), rating_comfort),
    rating_location    = COALESCE(sqlc.narg('rating_location'), rating_location),
    rating_host        = COALESCE(sqlc.narg('rating_host'), rating_host),
    rating_value       = COALESCE(sqlc.narg('rating_value'), rating_value),
    updated_at         = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteReview :execrows
DELETE FROM reviews WHERE id = $1;

-- name: GetVillaReviewAggregate :one
-- The villa's published rating, computed from its approved reviews only: a
-- review that is pending or hidden contributes nothing, and approving one folds
-- it straight in. The per-category AVGs skip NULLs, so each category is averaged
-- over the reviews that actually scored it and reads back NULL when none did.
SELECT
    COUNT(*)                          AS review_count,
    AVG(rating)::numeric              AS avg_rating,
    AVG(rating_cleanliness)::numeric  AS avg_cleanliness,
    AVG(rating_comfort)::numeric      AS avg_comfort,
    AVG(rating_location)::numeric     AS avg_location,
    AVG(rating_host)::numeric         AS avg_host,
    AVG(rating_value)::numeric        AS avg_value
FROM reviews
WHERE villa_slug = $1 AND status = 'approved';
