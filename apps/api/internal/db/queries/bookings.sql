-- name: InsertBooking :one
INSERT INTO bookings (
    id, villa_slug, guest_name, guest_email, guest_phone,
    check_in, check_out, adults, children, message, status, source
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetBookingByID :one
SELECT * FROM bookings WHERE id = $1;

-- name: FindOverlappingBookings :many
SELECT * FROM bookings
WHERE villa_slug = $1
  AND status IN ('pending', 'approved', 'paid')
  AND check_in  < $3
  AND check_out > $2;

-- name: ListBookingsByStatus :many
SELECT * FROM bookings
WHERE status = $1
ORDER BY created_at DESC;

-- name: UpdateBookingStatus :exec
UPDATE bookings
SET status = $2, updated_at = $3
WHERE id = $1;

-- name: ListBookedRanges :many
SELECT check_in, check_out FROM bookings
WHERE villa_slug = $1
  AND status IN ('approved', 'paid')
  AND check_in  < $3
  AND check_out > $2
ORDER BY check_in;

-- name: ListPendingRanges :many
SELECT check_in, check_out FROM bookings
WHERE villa_slug = $1
  AND status = 'pending'
  AND check_in  < $3
  AND check_out > $2
ORDER BY check_in;

-- name: ListBookingsPaged :many
SELECT * FROM bookings
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBookingsPagedByStatus :many
SELECT * FROM bookings
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListBookingsPagedByVilla :many
SELECT * FROM bookings
WHERE villa_slug = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListBookingsPagedByVillaAndStatus :many
SELECT * FROM bookings
WHERE villa_slug = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountBookings :one
SELECT COUNT(*) FROM bookings;

-- name: CountBookingsByStatus :one
SELECT COUNT(*) FROM bookings WHERE status = $1;

-- name: CountBookingsByVilla :one
SELECT COUNT(*) FROM bookings WHERE villa_slug = $1;

-- name: CountBookingsByVillaAndStatus :one
SELECT COUNT(*) FROM bookings WHERE villa_slug = $1 AND status = $2;

-- name: DeleteBooking :execrows
DELETE FROM bookings WHERE id = $1;
