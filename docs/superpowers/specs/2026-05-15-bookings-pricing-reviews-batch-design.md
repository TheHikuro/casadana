# Casa Dana — Bookings + Pricing + Reviews Batch Design

**Date:** 2026-05-15
**Status:** Approved — ready for implementation plan

## 1. Purpose & Scope

Add a batch of admin / data endpoints unblocking moderation and content workflows:

- List + delete bookings
- Bulk-upsert price overrides for a villa
- Reviews/testimonials module (linked to bookings)

All endpoints unauthenticated for now; the admin-auth gate will be added in the future Plan 2 by attaching `auth.AdminOnly` middleware to the relevant routes — paths stay the same.

**In scope:**

- `GET /api/bookings?page=&limit=&status=` — paginated booking list with optional status filter.
- `DELETE /api/bookings/{id}` — hard delete (cascades to linked review).
- `POST /api/villas/{slug}/pricing` — bulk upsert price overrides (`{price_cents, dates[]}`).
- New `internal/review/` module with table + 3 endpoints:
  - `POST /api/reviews` — submit (linked to a booking).
  - `GET /api/villas/{slug}/reviews` — list all reviews for a villa.
  - `DELETE /api/reviews/{id}` — hard delete.
- Migration `0003_reviews.up.sql` adding `reviews` table + `review_status` enum, with FK + cascade to bookings.
- OpenAPI spec extension + Orval client regeneration.

**Out of scope:**

- Auth on admin endpoints (Plan 2).
- PATCH review status (approve/reject via API).
- DELETE single price override by date.
- Pagination on reviews.
- Frontend wiring for any of this batch — generated hooks will sit unused in `@casa-dana/api` until the UI is built.
- Per-date pricing in a single request (one price applies to all dates in the array).

## 2. Architecture Decisions

### 2.1 Hard delete on bookings, with cascade to reviews

`DELETE /api/bookings/{id}` removes the row outright (`DELETE FROM bookings WHERE id = $1`). The review FK uses `ON DELETE CASCADE` so the linked review (if any) is removed atomically. For "soft cancel" semantics, the existing `PATCH /api/bookings/{id}` with `{status: "cancelled"}` is the right tool.

### 2.2 Pricing upsert: one price per request, multiple dates

The POST body is `{price_cents, dates[]}`. All dates in the array get the same price. To set different prices, the client makes multiple POST requests. Keeps the schema simple and matches the most common workflow ("high season Sat 1 → Sun 30 all at €250").

Implementation: loop in Go over the dates, call a single-row upsert per date, all wrapped in `pg.WithTx`.

### 2.3 Reviews are linked to bookings via FK + UNIQUE

`reviews.booking_id` is `NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE`. Effects:

- One review max per booking (UNIQUE).
- Submitting a review requires a real booking_id; `villa_slug` is derived server-side from the booking lookup (single source of truth).
- Deleting a booking removes the review automatically (cascade).

### 2.4 Module isolation: `review` consumes `booking` via a port

`review/ports.go` defines `BookingReader { GetVillaSlug(ctx, id) (string, error) }`. The `cmd/server/casadana.go` composition root wires `bookingSvc` behind a tiny adapter satisfying this interface. No direct module-to-module import.

`booking.Service` gains a `Get(ctx, id) (*Booking, error)` method that delegates to its repo so the adapter has something to call.

### 2.5 Reviews list returns all statuses for now

`GET /api/villas/{slug}/reviews` returns every review for the villa regardless of status (pending / approved / rejected). The future auth gate will split this: public callers see only `approved`, admin sees all. For now, no filter — the consumer is admin tooling.

### 2.6 Pagination shape: offset/limit + total count

`GET /api/bookings?page=1&limit=20` returns `{bookings, page, limit, total}`. `page` is 1-based, `limit` is clamped to `[1, 100]`, default 20. `total` lets the client compute total pages.

### 2.7 No frontend wiring this batch

Orval regenerates and exposes new hooks (`useListBookings`, `useDeleteBooking`, `useUpsertVillaPricing`, `useListVillaReviews`, `useSubmitReview`, `useDeleteReview`) via `@casa-dana/api`. None are imported by `apps/web` in this plan. UI integration happens later.

## 3. Database

### 3.1 Migration `0003_reviews.up.sql`

```sql
CREATE TYPE review_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE reviews (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id   UUID NOT NULL UNIQUE
                   REFERENCES bookings(id) ON DELETE CASCADE,
    villa_slug   TEXT NOT NULL,
    author_name  TEXT NOT NULL,
    rating       SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    body         TEXT NOT NULL DEFAULT '',
    status       review_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX reviews_villa_slug_idx ON reviews (villa_slug, status);
```

### 3.2 Migration `0003_reviews.down.sql`

```sql
DROP TABLE IF EXISTS reviews;
DROP TYPE  IF EXISTS review_status;
```

### 3.3 sqlc additions to `bookings.sql`

```sql
-- name: ListBookingsPaged :many
SELECT * FROM bookings
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBookingsPagedByStatus :many
SELECT * FROM bookings
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountBookings :one
SELECT COUNT(*) FROM bookings;

-- name: CountBookingsByStatus :one
SELECT COUNT(*) FROM bookings WHERE status = $1;

-- name: DeleteBooking :execrows
DELETE FROM bookings WHERE id = $1;
```

### 3.4 sqlc additions to `pricing.sql`

```sql
-- name: UpsertPriceOverride :exec
INSERT INTO price_overrides (villa_slug, date, price_cents)
VALUES ($1, $2, $3)
ON CONFLICT (villa_slug, date) DO UPDATE
SET price_cents = EXCLUDED.price_cents,
    updated_at = NOW();
```

### 3.5 New sqlc query file `reviews.sql`

```sql
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
```

## 4. Backend modules

### 4.1 `booking` additions

- `Repository` gains:
  - `List(ctx, status *Status, limit, offset int) ([]Booking, error)`
  - `Count(ctx, status *Status) (int, error)`
  - `Delete(ctx, id string) error`
- `Service` gains:
  - `Get(ctx, id string) (*Booking, error)` (used by review module via adapter)
  - `List(ctx, status *Status, page, limit int) (bookings []Booking, total int, err error)` — clamps `page>=1`, `limit ∈ [1,100]`, default 20
  - `Delete(ctx, id string) error` — returns `ErrNotFound` if delete affected 0 rows
- HTTP handlers:
  - `listBookingsHandler` — parses query, calls service, returns `{bookings, page, limit, total}`
  - `deleteBookingHandler` — 204 on success, 404 if `ErrNotFound`
- Mount adds `r.Get("/api/bookings", ...)` and `r.Delete("/api/bookings/{id}", ...)`

### 4.2 `pricing` additions

- `Repository` gains: `UpsertMany(ctx, villaSlug string, priceCents int, dates []time.Time) error`. Implementation iterates dates inside a `pg.WithTx` and calls `db.UpsertPriceOverride` per row.
- `Service` gains: `UpsertOverrides(ctx, villaSlug string, priceCents int, dates []time.Time) (count int, err error)`. Validates `priceCents >= 0` and `len(dates) > 0` (else `ErrInvalidPayload`).
- New sentinel: `ErrInvalidPayload = errors.New("invalid payload")`. Mapped to 422 `INVALID_PAYLOAD`.
- HTTP handler: `upsertPricingHandler` — decodes body, validates struct, returns `{count}` 201.
- Mount adds `r.Post("/api/villas/{slug}/pricing", ...)`.

### 4.3 New `review` module

Layout:

```
internal/review/
├── domain.go         # Review entity, NewReview constructor, sentinel errors
├── ports.go          # Repository, BookingReader, Clock
├── service.go        # Submit, ListByVilla, Delete
├── http.go           # Mount + 3 handlers + DTOs
├── postgres.go       # adapter
├── fakes_test.go     # fakeRepo, fakeBookingReader, fixedClock
├── service_test.go   # unit tests with fakes
├── http_test.go      # httptest with fake service
└── postgres_test.go  # //go:build integration
```

Domain:

```go
type Status string

const (
    StatusPending  Status = "pending"
    StatusApproved Status = "approved"
    StatusRejected Status = "rejected"
)

type Review struct {
    ID         string
    BookingID  string
    VillaSlug  string
    AuthorName string
    Rating     int
    Body       string
    Status     Status
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type NewReviewInput struct {
    BookingID  string
    VillaSlug  string
    AuthorName string
    Rating     int
    Body       string
    Now        time.Time
}

var (
    ErrBookingNotFound = errors.New("booking not found")
    ErrAlreadyReviewed = errors.New("review already exists for this booking")
    ErrNotFound        = errors.New("review not found")
    ErrInvalidPayload  = errors.New("invalid review payload")
)

func NewReview(in NewReviewInput) (*Review, error) {
    // trim, validate rating 1..5, name non-empty, body <= 2000
    // returns *Review with StatusPending
}
```

Ports:

```go
type Repository interface {
    Save(ctx context.Context, r *Review) error           // returns ErrAlreadyReviewed on UNIQUE conflict
    ListByVillaSlug(ctx context.Context, slug string) ([]Review, error)
    Delete(ctx context.Context, id string) error         // returns ErrNotFound if 0 rows affected
}

type BookingReader interface {
    GetVillaSlug(ctx context.Context, bookingID string) (string, error)  // returns ErrBookingNotFound
}

type Clock interface { Now() time.Time }
```

Service:

```go
type SubmitCommand struct {
    BookingID  string
    AuthorName string
    Rating     int
    Body       string
}

func (s *Service) Submit(ctx context.Context, cmd SubmitCommand) (*Review, error) {
    villaSlug, err := s.bookings.GetVillaSlug(ctx, cmd.BookingID)
    if err != nil {
        return nil, err  // ErrBookingNotFound bubbles
    }
    r, err := NewReview(NewReviewInput{
        BookingID: cmd.BookingID,
        VillaSlug: villaSlug,
        AuthorName: cmd.AuthorName,
        Rating: cmd.Rating,
        Body: cmd.Body,
        Now: s.clock.Now(),
    })
    if err != nil {
        return nil, err  // wraps as ErrInvalidPayload
    }
    if err := s.repo.Save(ctx, r); err != nil {
        return nil, err
    }
    return r, nil
}

func (s *Service) ListByVilla(ctx, slug string) ([]Review, error) { ... }
func (s *Service) Delete(ctx, id string) error { ... }
```

HTTP:

```go
func init() {
    httpserver.Register(ErrBookingNotFound, http.StatusNotFound, "BOOKING_NOT_FOUND")
    httpserver.Register(ErrAlreadyReviewed, http.StatusConflict, "ALREADY_REVIEWED")
    httpserver.Register(ErrNotFound, http.StatusNotFound, "REVIEW_NOT_FOUND")
    httpserver.Register(ErrInvalidPayload, http.StatusUnprocessableEntity, "INVALID_PAYLOAD")
}

func Mount(r chi.Router, svc *Service) {
    r.Post("/api/reviews", submitHandler(svc))
    r.Get("/api/villas/{slug}/reviews", listByVillaHandler(svc))
    r.Delete("/api/reviews/{id}", deleteHandler(svc))
}
```

Postgres adapter handles UNIQUE conflict by checking for `pgconn.PgError` with code `23505` and mapping to `ErrAlreadyReviewed`.

### 4.4 Cross-module wiring in `cmd/server/casadana.go`

```go
// Adapter so review.Service can fetch the villa_slug for a booking_id
// without importing the booking package directly.
type bookingReaderAdapter struct{ svc *booking.Service }
func (a bookingReaderAdapter) GetVillaSlug(ctx context.Context, id string) (string, error) {
    b, err := a.svc.Get(ctx, id)
    if err != nil {
        if errors.Is(err, booking.ErrNotFound) {
            return "", review.ErrBookingNotFound
        }
        return "", err
    }
    return b.VillaSlug, nil
}

reviewSvc := review.NewService(review.NewPgRepo(pool), bookingReaderAdapter{svc: bookingSvc}, realClock{})
review.Mount(r, reviewSvc)
```

## 5. OpenAPI

New tag: `reviews`. New operations: 6.

| Operation ID | Method | Path | Tag |
|---|---|---|---|
| `listBookings` | GET | `/api/bookings` | bookings |
| `deleteBooking` | DELETE | `/api/bookings/{id}` | bookings |
| `upsertVillaPricing` | POST | `/api/villas/{slug}/pricing` | pricing |
| `submitReview` | POST | `/api/reviews` | reviews |
| `listVillaReviews` | GET | `/api/villas/{slug}/reviews` | reviews |
| `deleteReview` | DELETE | `/api/reviews/{id}` | reviews |

New schemas:
- `ListBookingsResponse` — `{bookings: [BookingResponse], page, limit, total}`
- `UpsertPricingRequest` — `{price_cents: int (≥0), dates: [date] (min 1 item)}`
- `UpsertPricingResponse` — `{count: int}`
- `ReviewStatus` — enum `pending|approved|rejected`
- `Review` — entity shape
- `ListReviewsResponse` — `{reviews: [Review]}`
- `SubmitReviewRequest` — `{booking_id (uuid), author_name (1..120), rating (1..5), body (≤2000)}`

All responses use the existing `ErrorResponse` envelope for errors.

## 6. Orval regen

After OpenAPI update, `bun --filter @casa-dana/api generate` produces:
- New `packages/api/src/generated/reviews/reviews.ts` with `useSubmitReview`, `useListVillaReviews`, `useDeleteReview`.
- Existing files updated with `useListBookings`, `useDeleteBooking`, `useUpsertVillaPricing`.

The barrel `packages/api/src/index.ts` adds `export * from "./generated/reviews/reviews"`.

No frontend code consumes these in this plan.

## 7. Tests

- **Unit** (fast, no IO): each module's `service_test.go` with fakes for ports. Cases per endpoint:
  - bookings list: pagination clamps, status filter routing, total count
  - bookings delete: not-found → `ErrNotFound`
  - pricing upsert: empty dates / negative price → `ErrInvalidPayload`, unknown villa → `ErrUnknownVilla`
  - review submit: unknown booking → `ErrBookingNotFound`, duplicate → `ErrAlreadyReviewed`, bad rating → `ErrInvalidPayload`
  - review delete: not-found → `ErrNotFound`
- **HTTP** (`http_test.go`): one test per status code path per endpoint
- **Integration** (`postgres_test.go`, build-tagged): one happy-path test per module that exercises the real SQL (bookings list paginated, pricing upsert + read-back, review insert + cascade-delete-on-booking-delete)

## 8. Composition + final wiring

`cmd/server/casadana.go` ends with:

```go
bookingSvc := booking.NewService(booking.NewPgRepo(pool), booking.NewResendMailer(mailer), slugAllowlist{}, realClock{})
pricingSvc := pricing.NewService(pricing.NewPgRepo(pool), slugAllowlist{})
reviewSvc  := review.NewService(review.NewPgRepo(pool), bookingReaderAdapter{svc: bookingSvc}, realClock{})

r := httpserver.NewRouter(log, cfg.WebOrigin)
openapi.Mount(r)
booking.Mount(r, bookingSvc)
pricing.Mount(r, pricingSvc)
review.Mount(r, reviewSvc)
```

## 9. Out-of-Scope (explicit)

- Auth on admin endpoints — Plan 2.
- PATCH review status (admin moderation API) — when needed.
- DELETE single price override by date.
- Pagination on reviews — small volume, defer.
- Per-date prices in one request — POST one price per call; multiple calls for varying prices.
- Frontend UI for any of these endpoints — when product needs them.
- Verifying booking status before allowing review (e.g. only `paid` bookings can be reviewed) — server accepts review for any booking status now.
- Booking timestamps in `Review` (we link via FK; the booking row carries its own timestamps).
- Soft-delete of reviews — DELETE = hard delete.
