-- name: InsertReview :one
INSERT INTO reviews (id, booking_id, villa_slug, author_name, rating, body, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListReviewsByVilla :many
SELECT * FROM reviews
WHERE villa_slug = $1
ORDER BY created_at DESC;

-- name: DeleteReview :execrows
DELETE FROM reviews WHERE id = $1;
