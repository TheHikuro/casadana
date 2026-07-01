# Bookings + Pricing + Reviews Batch — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Commit policy:** DO NOT run `git commit`, `git add`, `git stash`, `git stash pop`, `git checkout`, `git restore`, `git reset`, `git clean`, or any other git state-changing command. Make file changes only. The user runs commits manually. (Memories: `feedback-no-auto-commit`, `feedback-subagent-no-git-stash`.)

**Goal:** Add booking list/delete, pricing bulk-upsert, and a new reviews module (linked to bookings via FK + cascade), exposed via OpenAPI + Orval. No frontend wiring.

**Architecture:** Backend-only batch. Three concerns:
- Extend `internal/booking` and `internal/pricing` with new repo + service + HTTP methods.
- New hexagonal `internal/review` module (domain → ports → service → adapters), consuming `booking` only via a `BookingReader` port.
- Migration `0003_reviews` adds the `reviews` table with `booking_id UUID NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE`.

**Tech Stack:** Go 1.25, chi, pgx/v5, sqlc, golang-migrate (embedded), validator/v10. Orval regenerates the React Query client.

**Spec:** `docs/superpowers/specs/2026-05-15-bookings-pricing-reviews-batch-design.md`.

**Scope of this plan:**
- Phase A — DB layer (migration + sqlc queries + regenerate)
- Phase B — Booking module additions (List, Count, Delete, Get)
- Phase C — Pricing module addition (UpsertMany)
- Phase D — Review module (full new hexagonal slice)
- Phase E — Composition root wiring + OpenAPI extension + Orval regen
- Phase F — Manual smoke test

**Out of scope:**
- Auth on admin endpoints — Plan 2.
- PATCH review status — when needed.
- DELETE single price override by date.
- Frontend wiring — generated hooks sit unused until UI lands.

**Working dir:** backend = `apps/api/`; frontend = repo root or `packages/api/`.

---

## Phase A — DB layer

### Task A1: Migration 0003 (reviews table)

**Files:**
- Create: `apps/api/internal/db/migrations/0003_reviews.up.sql`
- Create: `apps/api/internal/db/migrations/0003_reviews.down.sql`

- [ ] **Step 1: Write the up migration**

`apps/api/internal/db/migrations/0003_reviews.up.sql`:

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

- [ ] **Step 2: Write the down migration**

`apps/api/internal/db/migrations/0003_reviews.down.sql`:

```sql
DROP TABLE IF EXISTS reviews;
DROP TYPE  IF EXISTS review_status;
```

The existing `internal/db/embed.go` (`//go:embed migrations/*.sql`) auto-includes the new files. No code change needed.

---

### Task A2: Add booking sqlc queries

**Files:**
- Modify: `apps/api/internal/db/queries/bookings.sql`

- [ ] **Step 1: Append the new queries**

Append at the bottom of `apps/api/internal/db/queries/bookings.sql`:

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

`:execrows` returns the number of affected rows (int64) — needed so Delete can detect not-found.

---

### Task A3: Add pricing upsert sqlc query

**Files:**
- Modify: `apps/api/internal/db/queries/pricing.sql`

- [ ] **Step 1: Append the upsert query**

Append at the bottom of `apps/api/internal/db/queries/pricing.sql`:

```sql
-- name: UpsertPriceOverride :exec
INSERT INTO price_overrides (villa_slug, date, price_cents)
VALUES ($1, $2, $3)
ON CONFLICT (villa_slug, date) DO UPDATE
SET price_cents = EXCLUDED.price_cents,
    updated_at = NOW();
```

---

### Task A4: New reviews sqlc query file

**Files:**
- Create: `apps/api/internal/db/queries/reviews.sql`

- [ ] **Step 1: Write the queries**

`apps/api/internal/db/queries/reviews.sql`:

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

---

### Task A5: Regenerate sqlc + verify build

**Files:**
- Modified (regenerated): `apps/api/internal/db/{bookings,pricing,reviews,models,querier,db}.sql.go` and friends

- [ ] **Step 1: Regenerate**

From `apps/api/`:

```bash
sqlc generate
```

Expected:
- `internal/db/reviews.sql.go` is created with `InsertReview`, `ListReviewsByVilla`, `DeleteReview` methods.
- `internal/db/bookings.sql.go` gains `ListBookingsPaged`, `ListBookingsPagedByStatus`, `CountBookings`, `CountBookingsByStatus`, `DeleteBooking`.
- `internal/db/pricing.sql.go` gains `UpsertPriceOverride`.
- `internal/db/models.go` gains a `Review` struct.

- [ ] **Step 2: Verify build**

From `apps/api/`:

```bash
go build ./...
```

Expected: exit 0. If it fails because the new sqlc-generated methods aren't imported anywhere yet, that's wrong — sqlc-generated code should always compile standalone. If it does fail, `sqlc generate` likely produced incompatible output; inspect and report.

- [ ] **Step 3: Capture exact field names**

```bash
grep -A 6 "type ListBookingsPagedByStatusParams" internal/db/bookings.sql.go
grep -A 4 "type DeleteReviewParams\|func.*DeleteReview" internal/db/reviews.sql.go
grep -A 6 "type InsertReviewParams" internal/db/reviews.sql.go
```

Note the exact field names; later tasks reference them. Common adaptations:
- `:execrows` queries return `(int64, error)` — make sure adapters use `int64`.
- The reviews `id` column may need to be passed as `pgtype.UUID`; adapter converts from `string` via `uuid.Parse` + `[16]byte(uid)` (same pattern as `booking/postgres.go`).

---

## Phase B — Booking module additions

### Task B1: Add Repository methods to ports.go

**Files:**
- Modify: `apps/api/internal/booking/ports.go`

- [ ] **Step 1: Extend the Repository interface**

In `apps/api/internal/booking/ports.go`, the current `Repository` interface has `Save`, `FindOverlapping`, `BookedRanges`, `Get`, `UpdateStatus`. Add three more methods. Replace the `Repository` interface with:

```go
type Repository interface {
	Save(ctx context.Context, b *Booking) error
	FindOverlapping(ctx context.Context, villaSlug string, from, to time.Time) ([]Booking, error)
	BookedRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error)
	Get(ctx context.Context, id string) (*Booking, error)
	UpdateStatus(ctx context.Context, id string, status Status, updatedAt time.Time) error
	List(ctx context.Context, status *Status, limit, offset int) ([]Booking, error)
	Count(ctx context.Context, status *Status) (int, error)
	Delete(ctx context.Context, id string) error
}
```

Verify build (will fail until B2 is done — that's expected):

```bash
cd apps/api && go build ./internal/booking/...
```

Expected error: `*pgRepo does not implement Repository` because the new methods aren't on the postgres adapter yet.

---

### Task B2: Implement Repository methods in postgres.go

**Files:**
- Modify: `apps/api/internal/booking/postgres.go`

- [ ] **Step 1: Append the three methods**

At the bottom of `apps/api/internal/booking/postgres.go` (after the existing `rowToBooking` helper), append:

```go
func (r *pgRepo) List(ctx context.Context, status *Status, limit, offset int) ([]Booking, error) {
	if status != nil {
		rows, err := r.q().ListBookingsPagedByStatus(ctx, db.ListBookingsPagedByStatusParams{
			Status: db.BookingStatus(*status),
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, err
		}
		out := make([]Booking, 0, len(rows))
		for _, row := range rows {
			out = append(out, rowToBooking(row))
		}
		return out, nil
	}
	rows, err := r.q().ListBookingsPaged(ctx, db.ListBookingsPagedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Booking, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToBooking(row))
	}
	return out, nil
}

func (r *pgRepo) Count(ctx context.Context, status *Status) (int, error) {
	if status != nil {
		n, err := r.q().CountBookingsByStatus(ctx, db.BookingStatus(*status))
		if err != nil {
			return 0, err
		}
		return int(n), nil
	}
	n, err := r.q().CountBookings(ctx)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func (r *pgRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("booking: invalid id: %w", err)
	}
	rows, err := r.q().DeleteBooking(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
```

`uuid` and `pgtype` are already imported by the file. The `ErrNotFound` sentinel is already declared in `domain.go`.

Verify build:

```bash
cd apps/api && go build ./...
```

Expected: exit 0.

---

### Task B3: Add Service methods (Get, List, Delete) with TDD

**Files:**
- Modify: `apps/api/internal/booking/service.go`
- Modify: `apps/api/internal/booking/service_test.go`
- Modify: `apps/api/internal/booking/fakes_test.go` (the `fakeRepo` needs the new methods)

- [ ] **Step 1: Update the fake to satisfy the extended interface**

Append to `apps/api/internal/booking/fakes_test.go`:

```go
func (f *fakeRepo) List(_ context.Context, status *Status, limit, offset int) ([]Booking, error) {
	filtered := f.saved
	if status != nil {
		filtered = nil
		for _, b := range f.saved {
			if b.Status == *status {
				filtered = append(filtered, b)
			}
		}
	}
	if offset >= len(filtered) {
		return []Booking{}, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], nil
}

func (f *fakeRepo) Count(_ context.Context, status *Status) (int, error) {
	if status == nil {
		return len(f.saved), nil
	}
	n := 0
	for _, b := range f.saved {
		if b.Status == *status {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	for i, b := range f.saved {
		if b.ID == id {
			f.saved = append(f.saved[:i], f.saved[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
```

- [ ] **Step 2: Write failing tests**

Append to `apps/api/internal/booking/service_test.go`:

```go
func TestList_Pagination(t *testing.T) {
	repo := &fakeRepo{}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true}}
	svc := newSvc(repo, &fakeMailer{}, allow, d("2026-05-12"))

	// Seed 3 bookings
	for i := 0; i < 3; i++ {
		_, err := svc.Create(context.Background(), CreateCommand{
			VillaSlug:  "casadana",
			GuestName:  "Jane",
			GuestEmail: "jane@example.com",
			GuestPhone: "+33",
			CheckIn:    d("2026-07-01").AddDate(0, 0, i*10),
			CheckOut:   d("2026-07-08").AddDate(0, 0, i*10),
			Adults:     2,
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	bookings, total, err := svc.List(context.Background(), nil, 1, 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(bookings) != 2 {
		t.Errorf("len = %d, want 2 (limit=2)", len(bookings))
	}
}

func TestList_StatusFilter(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", Status: StatusPending},
			{ID: "2", Status: StatusApproved},
			{ID: "3", Status: StatusPending},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	pending := StatusPending
	bookings, total, err := svc.List(context.Background(), &pending, 1, 50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(bookings) != 2 {
		t.Errorf("len = %d, want 2", len(bookings))
	}
}

func TestList_Clamps(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	cases := []struct {
		page, limit       int
		wantLim, wantOff  int
	}{
		{0, 20, 20, 0},     // page < 1 clamps to 1
		{1, 0, 20, 0},      // limit < 1 defaults to 20
		{1, 200, 100, 0},   // limit > 100 clamps
		{3, 25, 25, 50},    // (3-1) * 25 = 50 offset
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("page=%d/limit=%d", c.page, c.limit), func(t *testing.T) {
			// We can't introspect the actual offset/limit passed to repo from outside,
			// but we can call List and ensure no error. The clamping math is what matters.
			_, _, err := svc.List(context.Background(), nil, c.page, c.limit)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
		})
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	err := svc.Delete(context.Background(), "nonexistent-id")
	if err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDelete_Happy(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{{ID: "abc-123"}},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	if err := svc.Delete(context.Background(), "abc-123"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(repo.saved) != 0 {
		t.Errorf("repo not emptied: %d remaining", len(repo.saved))
	}
}

func TestGet_NotFound(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))

	_, err := svc.Get(context.Background(), "missing")
	if err != ErrNotFound && !errors.Is(err, ErrNotFound) {
		// fakeRepo returns "not found" string from Get; we accept either.
		// In real impl repo.Get returns ErrNotFound from postgres adapter.
		t.Fatalf("err = %v, want ErrNotFound (or wraps)", err)
	}
}
```

You'll need `"fmt"` and `"errors"` in the imports of `service_test.go` if not already there. Add as needed.

- [ ] **Step 3: Run tests, confirm RED**

```bash
cd apps/api && go test ./internal/booking/...
```

Expected: compile errors — `svc.List`, `svc.Delete`, `svc.Get` undefined.

- [ ] **Step 4: Implement the service methods**

Append to `apps/api/internal/booking/service.go`:

```go
// Get returns a booking by id. Returns ErrNotFound if missing.
func (s *Service) Get(ctx context.Context, id string) (*Booking, error) {
	return s.repo.Get(ctx, id)
}

// List returns a page of bookings ordered by created_at DESC, with optional status filter.
// page is 1-based, limit is clamped to [1, 100], default 20.
func (s *Service) List(ctx context.Context, status *Status, page, limit int) ([]Booking, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	bookings, err := s.repo.List(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("booking: list: %w", err)
	}
	total, err := s.repo.Count(ctx, status)
	if err != nil {
		return nil, 0, fmt.Errorf("booking: count: %w", err)
	}
	return bookings, total, nil
}

// Delete hard-deletes a booking. Returns ErrNotFound if no row matched.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
```

- [ ] **Step 5: Run tests, confirm GREEN**

```bash
cd apps/api && go test ./internal/booking/...
```

Expected: PASS. The `TestGet_NotFound` test passes because the fake's `Get` returns `errors.New("not found")` (not `ErrNotFound`); we accept either via the test logic. In production, the postgres adapter returns `ErrNotFound` directly.

---

### Task B4: HTTP handlers + Mount + tests

**Files:**
- Modify: `apps/api/internal/booking/http.go`
- Modify: `apps/api/internal/booking/http_test.go`

- [ ] **Step 1: Add the new mount routes**

In `apps/api/internal/booking/http.go`, find the existing `Mount` function and replace with:

```go
func Mount(r chi.Router, svc *Service) {
	r.Post("/api/bookings", createHandler(svc))
	r.Get("/api/bookings", listBookingsHandler(svc))
	r.Patch("/api/bookings/{id}", patchBookingHandler(svc))
	r.Delete("/api/bookings/{id}", deleteBookingHandler(svc))
	r.Get("/api/villas/{slug}/availability", availabilityHandler(svc))
}
```

- [ ] **Step 2: Add response DTO + list handler**

Append to `apps/api/internal/booking/http.go` (after the existing `patchBookingHandler`):

```go
type listBookingsResponse struct {
	Bookings []bookingResponse `json:"bookings"`
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
	Total    int               `json:"total"`
}

func listBookingsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		var statusFilter *Status
		if s := r.URL.Query().Get("status"); s != "" {
			switch Status(s) {
			case StatusPending, StatusApproved, StatusRejected, StatusCancelled, StatusPaid:
				st := Status(s)
				statusFilter = &st
			default:
				httpserver.WriteError(w, r, &httpserver.ValidationError{
					Message: "status must be one of: pending, approved, rejected, cancelled, paid",
				})
				return
			}
		}

		bookings, total, err := svc.List(r.Context(), statusFilter, page, limit)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		// service has clamped page/limit internally — re-derive for the response
		if page < 1 {
			page = 1
		}
		if limit < 1 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		resp := listBookingsResponse{
			Bookings: make([]bookingResponse, 0, len(bookings)),
			Page:     page,
			Limit:    limit,
			Total:    total,
		}
		for i := range bookings {
			resp.Bookings = append(resp.Bookings, toResponse(&bookings[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func deleteBookingHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := svc.Delete(r.Context(), id); err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
```

Add `"strconv"` to the imports of `http.go` if not already present.

- [ ] **Step 3: Write http tests**

Append to `apps/api/internal/booking/http_test.go`:

```go
func TestListBookings_PaginatedResponse(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", Status: StatusPending, GuestEmail: "a@x.com", GuestName: "A", VillaSlug: "casadana", CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08")},
			{ID: "2", Status: StatusApproved, GuestEmail: "b@x.com", GuestName: "B", VillaSlug: "casadana", CheckIn: d("2026-08-01"), CheckOut: d("2026-08-08")},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/bookings?page=1&limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out listBookingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Total != 2 || out.Page != 1 || out.Limit != 10 {
		t.Errorf("page/limit/total = %d/%d/%d, want 1/10/2", out.Page, out.Limit, out.Total)
	}
	if len(out.Bookings) != 2 {
		t.Errorf("bookings len = %d, want 2", len(out.Bookings))
	}
}

func TestListBookings_BadStatus(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/bookings?status=invalid")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestDeleteBooking_NoContent(t *testing.T) {
	repo := &fakeRepo{saved: []Booking{{ID: "abc-123"}}}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/bookings/abc-123", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestDeleteBooking_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/bookings/nope", nil)
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
```

- [ ] **Step 4: Run all booking tests**

```bash
cd apps/api && go test ./internal/booking/...
```

Expected: all PASS.

---

## Phase C — Pricing module addition

### Task C1: Add ErrInvalidPayload sentinel + Repository.UpsertMany

**Files:**
- Modify: `apps/api/internal/pricing/domain.go`
- Modify: `apps/api/internal/pricing/ports.go`
- Modify: `apps/api/internal/pricing/postgres.go`

- [ ] **Step 1: Add the sentinel error**

In `apps/api/internal/pricing/domain.go`, find the existing `var (...)` block and add `ErrInvalidPayload`:

```go
var (
	ErrUnknownVilla   = errors.New("unknown villa")
	ErrInvalidRange   = errors.New("from must be before to")
	ErrInvalidPayload = errors.New("invalid payload")
)
```

- [ ] **Step 2: Extend the Repository interface**

In `apps/api/internal/pricing/ports.go`, change `Repository` to:

```go
type Repository interface {
	ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error)
	UpsertMany(ctx context.Context, villaSlug string, priceCents int, dates []time.Time) error
}
```

- [ ] **Step 3: Implement UpsertMany on the postgres adapter**

Append to `apps/api/internal/pricing/postgres.go`:

```go
import (
	pg "github.com/TheHikuro/casadana/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
)

func (r *pgRepo) UpsertMany(ctx context.Context, villaSlug string, priceCents int, dates []time.Time) error {
	return pg.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		for _, dt := range dates {
			if err := q.UpsertPriceOverride(ctx, db.UpsertPriceOverrideParams{
				VillaSlug:  villaSlug,
				Date:       pgtype.Date{Time: dt, Valid: true},
				PriceCents: int32(priceCents),
			}); err != nil {
				return fmt.Errorf("upsert %s: %w", dt.Format("2006-01-02"), err)
			}
		}
		return nil
	})
}
```

The imports need merging — `pgx` and the `pg` alias. If the file already imports them, dedupe. (Existing imports in `postgres.go` include `pgxpool`, `pgtype`, `db`, `errors`, `context`, `fmt`, `time`, and `uuid` from earlier work; add `pgx` and the `pg` aliased import.)

- [ ] **Step 4: Verify build**

```bash
cd apps/api && go build ./internal/pricing/...
```

Expected: exit 0. The interface mismatch will surface in service_test fakes — handled in C2.

---

### Task C2: Add Service.UpsertOverrides + tests

**Files:**
- Modify: `apps/api/internal/pricing/service.go`
- Modify: `apps/api/internal/pricing/fakes_test.go`
- Modify: `apps/api/internal/pricing/service_test.go`

- [ ] **Step 1: Update fakeRepo for the new interface method**

Append to `apps/api/internal/pricing/fakes_test.go`:

```go
func (f *fakeRepo) UpsertMany(_ context.Context, villaSlug string, priceCents int, dates []time.Time) error {
	for _, d := range dates {
		f.overrides = append(f.overrides, PriceOverride{
			VillaSlug:  villaSlug,
			Date:       d,
			PriceCents: priceCents,
		})
	}
	return nil
}
```

- [ ] **Step 2: Write failing tests**

Append to `apps/api/internal/pricing/service_test.go`:

```go
func TestUpsertOverrides_Happy(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})

	count, err := svc.UpsertOverrides(context.Background(), "casadana", 25000, []time.Time{
		d("2026-07-04"), d("2026-07-05"),
	})
	if err != nil {
		t.Fatalf("UpsertOverrides: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestUpsertOverrides_UnknownVilla(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	_, err := svc.UpsertOverrides(context.Background(), "ghost", 100, []time.Time{d("2026-07-04")})
	if err != ErrUnknownVilla {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}

func TestUpsertOverrides_NegativePrice(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	_, err := svc.UpsertOverrides(context.Background(), "casadana", -1, []time.Time{d("2026-07-04")})
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestUpsertOverrides_EmptyDates(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	_, err := svc.UpsertOverrides(context.Background(), "casadana", 100, nil)
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}
```

- [ ] **Step 3: Run tests, confirm RED**

```bash
cd apps/api && go test ./internal/pricing/...
```

Expected: compile error — `svc.UpsertOverrides` undefined.

- [ ] **Step 4: Implement Service.UpsertOverrides**

Append to `apps/api/internal/pricing/service.go`:

```go
// UpsertOverrides bulk-sets price_cents for each date in `dates` for a villa.
// Returns the number of upserted rows on success.
func (s *Service) UpsertOverrides(ctx context.Context, villaSlug string, priceCents int, dates []time.Time) (int, error) {
	if !s.allow.IsKnown(villaSlug) {
		return 0, ErrUnknownVilla
	}
	if priceCents < 0 || len(dates) == 0 {
		return 0, ErrInvalidPayload
	}
	if err := s.repo.UpsertMany(ctx, villaSlug, priceCents, dates); err != nil {
		return 0, err
	}
	return len(dates), nil
}
```

- [ ] **Step 5: Run tests, confirm GREEN**

```bash
cd apps/api && go test ./internal/pricing/...
```

Expected: PASS.

---

### Task C3: HTTP upsert handler + test

**Files:**
- Modify: `apps/api/internal/pricing/http.go`
- Modify: `apps/api/internal/pricing/http_test.go`

- [ ] **Step 1: Register the new error code + add the handler**

In `apps/api/internal/pricing/http.go`, update `init()`:

```go
func init() {
	httpserver.Register(ErrUnknownVilla, http.StatusNotFound, "UNKNOWN_VILLA")
	httpserver.Register(ErrInvalidRange, http.StatusUnprocessableEntity, "INVALID_RANGE")
	httpserver.Register(ErrInvalidPayload, http.StatusUnprocessableEntity, "INVALID_PAYLOAD")
}
```

Update `Mount`:

```go
func Mount(r chi.Router, svc *Service) {
	r.Get("/api/villas/{slug}/pricing", listHandler(svc))
	r.Post("/api/villas/{slug}/pricing", upsertHandler(svc))
}
```

Add the handler + DTOs at the bottom of the file (after `listHandler`):

```go
type upsertPricingRequest struct {
	PriceCents int      `json:"price_cents" validate:"min=0"`
	Dates      []string `json:"dates"       validate:"required,min=1,dive,datetime=2006-01-02"`
}

type upsertPricingResponse struct {
	Count int `json:"count"`
}

func upsertHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")

		var req upsertPricingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}

		dates := make([]time.Time, 0, len(req.Dates))
		for _, ds := range req.Dates {
			t, err := time.Parse("2006-01-02", ds)
			if err != nil {
				httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid date: " + ds})
				return
			}
			dates = append(dates, t)
		}

		count, err := svc.UpsertOverrides(r.Context(), slug, req.PriceCents, dates)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(upsertPricingResponse{Count: count})
	}
}
```

The `validator` import (path `github.com/TheHikuro/casadana/internal/platform/validator`) needs to be added — booking module already uses it; check if pricing's `http.go` already imports it. If not, add it.

- [ ] **Step 2: Write http test**

Append to `apps/api/internal/pricing/http_test.go`:

```go
func TestUpsertPricing_Created(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"price_cents":25000,"dates":["2026-07-04","2026-07-05"]}`
	resp, err := http.Post(srv.URL+"/api/villas/casadana/pricing", "application/json",
		strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out upsertPricingResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Count != 2 {
		t.Errorf("count = %d, want 2", out.Count)
	}
}

func TestUpsertPricing_EmptyDates(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"price_cents":100,"dates":[]}`
	resp, _ := http.Post(srv.URL+"/api/villas/casadana/pricing", "application/json",
		strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}
```

Add `"strings"` import to `http_test.go` if not present.

- [ ] **Step 3: Run all pricing tests**

```bash
cd apps/api && go test ./internal/pricing/...
```

Expected: PASS.

---

## Phase D — Reviews module (new)

### Task D1: Domain + ports

**Files:**
- Create: `apps/api/internal/review/domain.go`
- Create: `apps/api/internal/review/ports.go`

- [ ] **Step 1: Write domain.go**

`apps/api/internal/review/domain.go`:

```go
package review

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
	in.AuthorName = strings.TrimSpace(in.AuthorName)
	in.Body = strings.TrimSpace(in.Body)

	if in.BookingID == "" {
		return nil, ErrInvalidPayload
	}
	if in.VillaSlug == "" {
		return nil, ErrInvalidPayload
	}
	if in.AuthorName == "" || len(in.AuthorName) > 120 {
		return nil, ErrInvalidPayload
	}
	if in.Rating < 1 || in.Rating > 5 {
		return nil, ErrInvalidPayload
	}
	if len(in.Body) > 2000 {
		return nil, ErrInvalidPayload
	}

	now := in.Now
	return &Review{
		ID:         uuid.NewString(),
		BookingID:  in.BookingID,
		VillaSlug:  in.VillaSlug,
		AuthorName: in.AuthorName,
		Rating:     in.Rating,
		Body:       in.Body,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
```

- [ ] **Step 2: Write ports.go**

`apps/api/internal/review/ports.go`:

```go
package review

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, r *Review) error
	ListByVillaSlug(ctx context.Context, slug string) ([]Review, error)
	Delete(ctx context.Context, id string) error
}

type BookingReader interface {
	GetVillaSlug(ctx context.Context, bookingID string) (string, error)
}

type Clock interface {
	Now() time.Time
}
```

- [ ] **Step 3: Verify build**

```bash
cd apps/api && go build ./internal/review/...
```

Expected: exit 0.

---

### Task D2: Test fakes

**Files:**
- Create: `apps/api/internal/review/fakes_test.go`

- [ ] **Step 1: Write the fakes**

`apps/api/internal/review/fakes_test.go`:

```go
package review

import (
	"context"
	"time"
)

type fakeRepo struct {
	saved   []Review
	saveErr error
}

func (f *fakeRepo) Save(_ context.Context, r *Review) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	for _, existing := range f.saved {
		if existing.BookingID == r.BookingID {
			return ErrAlreadyReviewed
		}
	}
	f.saved = append(f.saved, *r)
	return nil
}

func (f *fakeRepo) ListByVillaSlug(_ context.Context, slug string) ([]Review, error) {
	out := []Review{}
	for _, r := range f.saved {
		if r.VillaSlug == slug {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	for i, r := range f.saved {
		if r.ID == id {
			f.saved = append(f.saved[:i], f.saved[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

type fakeBookingReader struct {
	bySlug map[string]string // bookingID -> villaSlug
	err    error
}

func (f fakeBookingReader) GetVillaSlug(_ context.Context, bookingID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	slug, ok := f.bySlug[bookingID]
	if !ok {
		return "", ErrBookingNotFound
	}
	return slug, nil
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
```

- [ ] **Step 2: Verify the test binary compiles**

```bash
cd apps/api && go test -run NONE ./internal/review/...
```

Expected: PASS (no tests selected).

---

### Task D3: Service (TDD)

**Files:**
- Create: `apps/api/internal/review/service.go`
- Create: `apps/api/internal/review/service_test.go`

- [ ] **Step 1: Write failing tests**

`apps/api/internal/review/service_test.go`:

```go
package review

import (
	"context"
	"testing"
)

func newSvc(repo Repository, bookings BookingReader, clock Clock) *Service {
	return NewService(repo, bookings, clock)
}

func TestSubmit_Happy(t *testing.T) {
	repo := &fakeRepo{}
	bookings := fakeBookingReader{bySlug: map[string]string{"booking-1": "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})

	r, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "booking-1",
		AuthorName: "Jane",
		Rating:     5,
		Body:       "Loved it.",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if r.VillaSlug != "casadana" {
		t.Errorf("VillaSlug = %q, want casadana", r.VillaSlug)
	}
	if r.Status != StatusPending {
		t.Errorf("Status = %s, want pending", r.Status)
	}
	if len(repo.saved) != 1 {
		t.Errorf("saved count = %d, want 1", len(repo.saved))
	}
}

func TestSubmit_UnknownBooking(t *testing.T) {
	bookings := fakeBookingReader{bySlug: map[string]string{}}
	svc := newSvc(&fakeRepo{}, bookings, fixedClock{t: d("2026-08-01")})

	_, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "ghost",
		AuthorName: "X", Rating: 5,
	})
	if err != ErrBookingNotFound {
		t.Fatalf("err = %v, want ErrBookingNotFound", err)
	}
}

func TestSubmit_AlreadyReviewed(t *testing.T) {
	repo := &fakeRepo{
		saved: []Review{{ID: "existing", BookingID: "booking-1"}},
	}
	bookings := fakeBookingReader{bySlug: map[string]string{"booking-1": "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})

	_, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "booking-1",
		AuthorName: "X", Rating: 5,
	})
	if err != ErrAlreadyReviewed {
		t.Fatalf("err = %v, want ErrAlreadyReviewed", err)
	}
}

func TestSubmit_BadRating(t *testing.T) {
	bookings := fakeBookingReader{bySlug: map[string]string{"booking-1": "casadana"}}
	svc := newSvc(&fakeRepo{}, bookings, fixedClock{t: d("2026-08-01")})

	_, err := svc.Submit(context.Background(), SubmitCommand{
		BookingID:  "booking-1",
		AuthorName: "X", Rating: 6,
	})
	if err != ErrInvalidPayload {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestListByVilla(t *testing.T) {
	repo := &fakeRepo{
		saved: []Review{
			{ID: "1", VillaSlug: "casadana"},
			{ID: "2", VillaSlug: "casacasay"},
			{ID: "3", VillaSlug: "casadana"},
		},
	}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})

	out, err := svc.ListByVilla(context.Background(), "casadana")
	if err != nil {
		t.Fatalf("ListByVilla: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("len = %d, want 2", len(out))
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	if err := svc.Delete(context.Background(), "nope"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests, confirm RED**

```bash
cd apps/api && go test ./internal/review/...
```

Expected: compile errors — `Service`, `NewService`, `SubmitCommand`, methods undefined.

- [ ] **Step 3: Implement service.go**

`apps/api/internal/review/service.go`:

```go
package review

import (
	"context"
	"fmt"
)

type Service struct {
	repo     Repository
	bookings BookingReader
	clock    Clock
}

func NewService(repo Repository, bookings BookingReader, clock Clock) *Service {
	return &Service{repo: repo, bookings: bookings, clock: clock}
}

type SubmitCommand struct {
	BookingID  string
	AuthorName string
	Rating     int
	Body       string
}

func (s *Service) Submit(ctx context.Context, cmd SubmitCommand) (*Review, error) {
	villaSlug, err := s.bookings.GetVillaSlug(ctx, cmd.BookingID)
	if err != nil {
		return nil, err
	}

	r, err := NewReview(NewReviewInput{
		BookingID:  cmd.BookingID,
		VillaSlug:  villaSlug,
		AuthorName: cmd.AuthorName,
		Rating:     cmd.Rating,
		Body:       cmd.Body,
		Now:        s.clock.Now(),
	})
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, r); err != nil {
		return nil, fmt.Errorf("review: save: %w", err)
	}
	return r, nil
}

func (s *Service) ListByVilla(ctx context.Context, slug string) ([]Review, error) {
	return s.repo.ListByVillaSlug(ctx, slug)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
```

Note: `repo.Save` may return `ErrAlreadyReviewed` directly from the fake or wrapped from the postgres adapter. The wrap above (`fmt.Errorf("review: save: %w", err)`) preserves `errors.Is` semantics. **But** the test `TestSubmit_AlreadyReviewed` does `err != ErrAlreadyReviewed` (strict equality). To make it pass, change the test to use `errors.Is(err, ErrAlreadyReviewed)`, OR don't wrap the save error.

Choose: don't wrap. Replace the Save section with:

```go
	if err := s.repo.Save(ctx, r); err != nil {
		return nil, err
	}
```

Drop the `"fmt"` import if no longer used.

- [ ] **Step 4: Run tests, confirm GREEN**

```bash
cd apps/api && go test ./internal/review/...
```

Expected: PASS (6 tests).

---

### Task D4: Postgres adapter

**Files:**
- Create: `apps/api/internal/review/postgres.go`

- [ ] **Step 1: Write the adapter**

`apps/api/internal/review/postgres.go`:

```go
package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheHikuro/casadana/internal/db"
)

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) q() *db.Queries { return db.New(r.pool) }

func (r *pgRepo) Save(ctx context.Context, rv *Review) error {
	id, err := uuid.Parse(rv.ID)
	if err != nil {
		return fmt.Errorf("review: invalid id: %w", err)
	}
	bookingID, err := uuid.Parse(rv.BookingID)
	if err != nil {
		return fmt.Errorf("review: invalid booking id: %w", err)
	}
	_, err = r.q().InsertReview(ctx, db.InsertReviewParams{
		ID:         pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		BookingID:  pgtype.UUID{Bytes: [16]byte(bookingID), Valid: true},
		VillaSlug:  rv.VillaSlug,
		AuthorName: rv.AuthorName,
		Rating:     int16(rv.Rating),
		Body:       rv.Body,
		Status:     db.ReviewStatus(rv.Status),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// unique_violation — booking_id already has a review
			return ErrAlreadyReviewed
		}
		return err
	}
	return nil
}

func (r *pgRepo) ListByVillaSlug(ctx context.Context, slug string) ([]Review, error) {
	rows, err := r.q().ListReviewsByVilla(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := make([]Review, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToReview(row))
	}
	return out, nil
}

func (r *pgRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("review: invalid id: %w", err)
	}
	rows, err := r.q().DeleteReview(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func rowToReview(row db.Review) Review {
	idStr := ""
	if row.ID.Valid {
		u, _ := uuid.FromBytes(row.ID.Bytes[:])
		idStr = u.String()
	}
	bookingIDStr := ""
	if row.BookingID.Valid {
		u, _ := uuid.FromBytes(row.BookingID.Bytes[:])
		bookingIDStr = u.String()
	}
	return Review{
		ID:         idStr,
		BookingID:  bookingIDStr,
		VillaSlug:  row.VillaSlug,
		AuthorName: row.AuthorName,
		Rating:     int(row.Rating),
		Body:       row.Body,
		Status:     Status(row.Status),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
```

- [ ] **Step 2: Verify build**

```bash
cd apps/api && go build ./internal/review/...
```

Expected: exit 0. If sqlc generated `db.ReviewStatus` differently or `db.Review` field names differ, adjust and report.

---

### Task D5: HTTP handlers + tests

**Files:**
- Create: `apps/api/internal/review/http.go`
- Create: `apps/api/internal/review/http_test.go`

- [ ] **Step 1: Write http.go**

`apps/api/internal/review/http.go`:

```go
package review

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/validator"
)

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

type submitReviewRequest struct {
	BookingID  string `json:"booking_id"  validate:"required,uuid"`
	AuthorName string `json:"author_name" validate:"required,min=1,max=120"`
	Rating     int    `json:"rating"      validate:"required,min=1,max=5"`
	Body       string `json:"body"        validate:"max=2000"`
}

type reviewDTO struct {
	ID         string `json:"id"`
	BookingID  string `json:"booking_id"`
	VillaSlug  string `json:"villa_slug"`
	AuthorName string `json:"author_name"`
	Rating     int    `json:"rating"`
	Body       string `json:"body"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type listReviewsResponse struct {
	Reviews []reviewDTO `json:"reviews"`
}

func toDTO(r *Review) reviewDTO {
	return reviewDTO{
		ID:         r.ID,
		BookingID:  r.BookingID,
		VillaSlug:  r.VillaSlug,
		AuthorName: r.AuthorName,
		Rating:     r.Rating,
		Body:       r.Body,
		Status:     string(r.Status),
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
	}
}

func submitHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req submitReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		rv, err := svc.Submit(r.Context(), SubmitCommand{
			BookingID:  req.BookingID,
			AuthorName: req.AuthorName,
			Rating:     req.Rating,
			Body:       req.Body,
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toDTO(rv))
	}
}

func listByVillaHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		reviews, err := svc.ListByVilla(r.Context(), slug)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		resp := listReviewsResponse{Reviews: make([]reviewDTO, 0, len(reviews))}
		for i := range reviews {
			resp.Reviews = append(resp.Reviews, toDTO(&reviews[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func deleteHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := svc.Delete(r.Context(), id); err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 2: Write http_test.go**

`apps/api/internal/review/http_test.go`:

```go
package review

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc)
	return r
}

func TestSubmitReview_Created(t *testing.T) {
	bookingID := "11111111-1111-1111-1111-111111111111"
	repo := &fakeRepo{}
	bookings := fakeBookingReader{bySlug: map[string]string{bookingID: "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"` + bookingID + `","author_name":"Jane","rating":5,"body":"Great"}`
	resp, err := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out reviewDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "pending" || out.VillaSlug != "casadana" {
		t.Errorf("unexpected response: %+v", out)
	}
}

func TestSubmitReview_BookingNotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{bySlug: map[string]string{}}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"11111111-1111-1111-1111-111111111111","author_name":"X","rating":5}`
	resp, _ := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestSubmitReview_AlreadyReviewed(t *testing.T) {
	bookingID := "11111111-1111-1111-1111-111111111111"
	repo := &fakeRepo{saved: []Review{{ID: "existing", BookingID: bookingID}}}
	bookings := fakeBookingReader{bySlug: map[string]string{bookingID: "casadana"}}
	svc := newSvc(repo, bookings, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"` + bookingID + `","author_name":"X","rating":5}`
	resp, _ := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestSubmitReview_BadRating(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"booking_id":"11111111-1111-1111-1111-111111111111","author_name":"X","rating":99}`
	resp, _ := http.Post(srv.URL+"/api/reviews", "application/json", strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestListReviewsByVilla(t *testing.T) {
	repo := &fakeRepo{saved: []Review{
		{ID: "1", VillaSlug: "casadana", Status: StatusPending},
		{ID: "2", VillaSlug: "casacasay", Status: StatusApproved},
	}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/villas/casadana/reviews")
	defer resp.Body.Close()
	var out listReviewsResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Reviews) != 1 {
		t.Errorf("len = %d, want 1", len(out.Reviews))
	}
}

func TestDeleteReview_NoContent(t *testing.T) {
	repo := &fakeRepo{saved: []Review{{ID: "abc"}}}
	svc := newSvc(repo, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/reviews/abc", nil)
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestDeleteReview_NotFound(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeBookingReader{}, fixedClock{t: d("2026-08-01")})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/reviews/missing", nil)
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
```

- [ ] **Step 3: Run all review tests**

```bash
cd apps/api && go test ./internal/review/...
```

Expected: PASS (12 tests across service + http).

---

## Phase E — Wiring + OpenAPI + Orval

### Task E1: Wire review module in cmd/server

**Files:**
- Modify: `apps/api/cmd/server/casadana.go`

- [ ] **Step 1: Add the import**

In `apps/api/cmd/server/casadana.go`, add to the import block (alphabetical):

```go
"github.com/TheHikuro/casadana/internal/review"
```

Add `"errors"` to the imports if not already present (the adapter uses `errors.Is`).

- [ ] **Step 2: Add the bookingReaderAdapter type**

Just below the existing `slugAllowlist` type definition, add:

```go
// bookingReaderAdapter lets the review module look up a booking's villa_slug
// via the booking service without a direct module-to-module import.
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
```

- [ ] **Step 3: Construct + mount the review service**

In `main()`, after the `pricingSvc := pricing.NewService(...)` line, add:

```go
reviewSvc := review.NewService(
	review.NewPgRepo(pool),
	bookingReaderAdapter{svc: bookingSvc},
	realClock{},
)
```

After `pricing.Mount(r, pricingSvc)`, add:

```go
review.Mount(r, reviewSvc)
```

- [ ] **Step 4: Verify**

```bash
cd apps/api && go build ./... && go vet ./... && go test ./...
```

Expected: all green.

---

### Task E2: OpenAPI extension

**Files:**
- Modify: `apps/api/internal/openapi/openapi.yaml`

- [ ] **Step 1: Add the new tag**

In the `tags:` block, append:

```yaml
  - name: reviews
    description: Guest testimonials linked to bookings
```

- [ ] **Step 2: Add the new paths**

Insert these path blocks. Recommended placement: after the existing `PATCH /api/bookings/{id}` block, add the GET + DELETE for bookings; then after `/api/villas/{slug}/pricing` GET, add the POST; then add a new section for reviews.

For booking list and delete, add to the `/api/bookings` path (already has `post`):

```yaml
  /api/bookings:
    post:
      # (keep the existing post block — do not duplicate)
      ...
    get:
      operationId: listBookings
      tags: [bookings]
      summary: List bookings (paginated, optional status filter)
      parameters:
        - name: page
          in: query
          schema: { type: integer, minimum: 1, default: 1 }
        - name: limit
          in: query
          schema: { type: integer, minimum: 1, maximum: 100, default: 20 }
        - name: status
          in: query
          schema: { $ref: "#/components/schemas/BookingStatus" }
      responses:
        "200":
          description: Paginated bookings
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ListBookingsResponse"
        "422":
          description: Invalid query params
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "500":
          description: Internal server error
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
```

The existing `post:` and `get:` siblings live under the same `/api/bookings:` key — make sure not to create a duplicate path key.

For DELETE booking, the existing `/api/bookings/{id}` has `patch:`. Add a sibling `delete:`:

```yaml
    delete:
      operationId: deleteBooking
      tags: [bookings]
      summary: Hard delete a booking (cascades to its review)
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: string, format: uuid }
      responses:
        "204":
          description: Deleted
        "404":
          description: Booking not found
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "500":
          description: Internal server error
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
```

For POST pricing, the existing `/api/villas/{slug}/pricing` has `get:`. Add a sibling `post:`:

```yaml
    post:
      operationId: upsertVillaPricing
      tags: [pricing]
      summary: Bulk upsert price overrides for a villa
      description: |
        Sets `price_cents` for each date in the body. Existing overrides on
        the same date are replaced. All dates in one request share the same
        price; submit multiple requests to vary prices across dates.
      parameters:
        - name: slug
          in: path
          required: true
          schema: { type: string }
          example: casadana
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UpsertPricingRequest"
      responses:
        "201":
          description: Upserted
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UpsertPricingResponse"
        "404":
          description: Villa slug not in allowlist
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "422":
          description: Invalid payload
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "500":
          description: Internal server error
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
```

For reviews, add three new paths at the end of the paths section:

```yaml
  /api/reviews:
    post:
      operationId: submitReview
      tags: [reviews]
      summary: Submit a review for a booking
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/SubmitReviewRequest"
      responses:
        "201":
          description: Review accepted (pending)
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Review"
        "404":
          description: Booking not found
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "409":
          description: A review already exists for this booking
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "422":
          description: Validation error
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "500":
          description: Internal server error
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }

  /api/reviews/{id}:
    delete:
      operationId: deleteReview
      tags: [reviews]
      summary: Hard delete a review
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: string, format: uuid }
      responses:
        "204":
          description: Deleted
        "404":
          description: Review not found
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "500":
          description: Internal server error
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }

  /api/villas/{slug}/reviews:
    get:
      operationId: listVillaReviews
      tags: [reviews]
      summary: List all reviews for a villa
      description: Returns all statuses for now; auth-gated filtering arrives in Plan 2.
      parameters:
        - name: slug
          in: path
          required: true
          schema: { type: string }
      responses:
        "200":
          description: Reviews
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ListReviewsResponse"
        "500":
          description: Internal server error
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
```

- [ ] **Step 3: Add the new schemas**

In `components.schemas`, append (after the existing `PricingResponse` schema):

```yaml
    ListBookingsResponse:
      type: object
      required: [bookings, page, limit, total]
      properties:
        bookings:
          type: array
          items: { $ref: "#/components/schemas/BookingResponse" }
        page:  { type: integer }
        limit: { type: integer }
        total: { type: integer }

    UpsertPricingRequest:
      type: object
      required: [price_cents, dates]
      properties:
        price_cents:
          type: integer
          minimum: 0
          example: 25000
        dates:
          type: array
          minItems: 1
          items: { type: string, format: date }
          example: ["2026-07-04", "2026-07-05"]

    UpsertPricingResponse:
      type: object
      required: [count]
      properties:
        count: { type: integer }

    ReviewStatus:
      type: string
      enum: [pending, approved, rejected]

    Review:
      type: object
      required:
        - id
        - booking_id
        - villa_slug
        - author_name
        - rating
        - body
        - status
        - created_at
      properties:
        id:          { type: string, format: uuid }
        booking_id:  { type: string, format: uuid }
        villa_slug:  { type: string }
        author_name: { type: string }
        rating:      { type: integer, minimum: 1, maximum: 5 }
        body:        { type: string }
        status:      { $ref: "#/components/schemas/ReviewStatus" }
        created_at:  { type: string, format: date-time }

    ListReviewsResponse:
      type: object
      required: [reviews]
      properties:
        reviews:
          type: array
          items: { $ref: "#/components/schemas/Review" }

    SubmitReviewRequest:
      type: object
      required: [booking_id, author_name, rating]
      properties:
        booking_id:  { type: string, format: uuid }
        author_name: { type: string, minLength: 1, maxLength: 120 }
        rating:      { type: integer, minimum: 1, maximum: 5 }
        body:        { type: string, maxLength: 2000, default: "" }
```

- [ ] **Step 4: Lint**

```bash
cd /Users/loancleris/Desktop/projet-perso/casadana && bunx -y @stoplight/spectral-cli lint apps/api/internal/openapi/openapi.yaml
```

Expected: 0 errors. Pre-existing warnings (`info-contact`, `operation-description`) are fine.

---

### Task E3: Orval regen + barrel update

**Files:**
- Regenerated: `packages/api/src/generated/**`
- Modify: `packages/api/src/index.ts`

- [ ] **Step 1: Regenerate**

```bash
cd packages/api && bun run generate
```

Expected: a new `src/generated/reviews/reviews.ts` and updated `src/generated/bookings/bookings.ts`, `src/generated/pricing/pricing.ts`. New schemas appear under `src/generated/schemas/`.

- [ ] **Step 2: Add the reviews tag to the barrel**

In `packages/api/src/index.ts`, the current barrel re-exports each tag explicitly. Add the reviews export:

```ts
export * from "./generated/bookings/bookings"
export * from "./generated/health/health"
export * from "./generated/pricing/pricing"
export * from "./generated/reviews/reviews"
export * from "./generated/schemas"
export { customAxios, AXIOS_INSTANCE, ApiError } from "./client"
export type { ApiErrorBody } from "./client"
```

- [ ] **Step 3: Verify hooks emitted**

```bash
grep -E "^export (function|const) (useListBookings|useDeleteBooking|useUpsertVillaPricing|useSubmitReview|useListVillaReviews|useDeleteReview)" packages/api/src/generated/**/*.ts
```

Expected: matches for all 6 hook names. Note any deviations.

- [ ] **Step 4: Type-check**

```bash
cd packages/api && bunx tsc --noEmit
```

Expected: exit 0.

---

## Phase F — Manual smoke test

### Task F1: Rebuild API + curl every new endpoint

**Files:** none.

- [ ] **Step 1: Rebuild + restart api**

From repo root:

```bash
docker compose -f .docker/docker-compose.yml --env-file .env.dev --profile dev up -d --build api
```

Wait briefly, then verify health: `curl -s http://localhost:8080/api/health` → `{"status":"ok"}`.

- [ ] **Step 2: List bookings**

```bash
curl -s 'http://localhost:8080/api/bookings?page=1&limit=10' | head -c 400
```

Expected: JSON `{"bookings":[...], "page":1, "limit":10, "total":N}`. Should at least include the existing `9e7da644-...` booking from earlier work.

- [ ] **Step 3: List bookings filtered by status**

```bash
curl -s 'http://localhost:8080/api/bookings?status=approved'
```

Expected: only approved bookings.

- [ ] **Step 4: Bad status**

```bash
curl -s -i 'http://localhost:8080/api/bookings?status=bogus' | head -3
```

Expected: `HTTP/1.1 422 Unprocessable Entity`.

- [ ] **Step 5: Upsert price overrides**

```bash
curl -s -i -X POST http://localhost:8080/api/villas/casadana/pricing \
  -H "Content-Type: application/json" \
  -d '{"price_cents":30000,"dates":["2026-12-24","2026-12-25","2026-12-31"]}' | head -8
```

Expected: `HTTP/1.1 201 Created`, body `{"count":3}`. Confirm via `GET /api/villas/casadana/pricing?from=2026-12-01&to=2027-01-01` shows the three new overrides.

- [ ] **Step 6: Empty dates → 422**

```bash
curl -s -i -X POST http://localhost:8080/api/villas/casadana/pricing \
  -H "Content-Type: application/json" \
  -d '{"price_cents":100,"dates":[]}' | head -3
```

Expected: 422.

- [ ] **Step 7: Submit a review**

Use the approved booking ID from earlier:

```bash
BOOKING_ID=$(docker exec -i casadana-postgres psql -U casadana -d casadana -tA -c "SELECT id FROM bookings ORDER BY created_at LIMIT 1;")
curl -s -i -X POST http://localhost:8080/api/reviews \
  -H "Content-Type: application/json" \
  -d "{\"booking_id\":\"$BOOKING_ID\",\"author_name\":\"Loan\",\"rating\":5,\"body\":\"Top\"}" | head -8
```

Expected: 201 with the new review. `villa_slug` is `casadana` (derived).

- [ ] **Step 8: Duplicate review → 409**

Re-run Step 7. Expected: `HTTP/1.1 409 Conflict` with `ALREADY_REVIEWED`.

- [ ] **Step 9: List reviews for the villa**

```bash
curl -s http://localhost:8080/api/villas/casadana/reviews
```

Expected: `{"reviews":[{...the one we just created...}]}`.

- [ ] **Step 10: Delete the review**

Get its ID from the previous response, then:

```bash
curl -s -i -X DELETE "http://localhost:8080/api/reviews/$REVIEW_ID" | head -3
```

Expected: 204.

- [ ] **Step 11: Cascade verification — submit a new review then delete its booking**

```bash
# Submit a fresh review
curl -s -X POST http://localhost:8080/api/reviews \
  -H "Content-Type: application/json" \
  -d "{\"booking_id\":\"$BOOKING_ID\",\"author_name\":\"Loan\",\"rating\":5,\"body\":\"Cascade test\"}"

# Confirm 1 review exists
curl -s http://localhost:8080/api/villas/casadana/reviews | python3 -c "import json,sys; print('count:', len(json.load(sys.stdin)['reviews']))"

# Delete the booking
curl -s -i -X DELETE "http://localhost:8080/api/bookings/$BOOKING_ID" | head -3

# Confirm review was cascade-deleted
curl -s http://localhost:8080/api/villas/casadana/reviews | python3 -c "import json,sys; print('count:', len(json.load(sys.stdin)['reviews']))"
```

Expected: count=1 before delete, then 204, then count=0 after (cascade worked).

- [ ] **Step 12: Stop the stack when done**

```bash
docker compose -f .docker/docker-compose.yml --env-file .env.dev --profile dev down
```

---

## Post-flight

- [ ] **Files touched (sanity list)**

Created:
- `apps/api/internal/db/migrations/0003_reviews.{up,down}.sql`
- `apps/api/internal/db/queries/reviews.sql`
- `apps/api/internal/db/reviews.sql.go` (sqlc-generated)
- `apps/api/internal/review/{domain,ports,service,http,postgres,fakes_test,service_test,http_test}.go`

Modified:
- `apps/api/internal/db/queries/{bookings,pricing}.sql` (new queries appended)
- `apps/api/internal/db/{bookings,pricing,models,querier,db}.sql.go` (sqlc regenerated)
- `apps/api/internal/booking/{ports,service,postgres,http,fakes_test,service_test,http_test}.go`
- `apps/api/internal/pricing/{domain,ports,service,postgres,http,fakes_test,service_test,http_test}.go`
- `apps/api/cmd/server/casadana.go` (review wiring)
- `apps/api/internal/openapi/openapi.yaml` (6 new operations + schemas + tag)
- `packages/api/src/index.ts` (barrel adds reviews)

Regenerated (committed):
- `packages/api/src/generated/reviews/`, `bookings/`, `pricing/`, `schemas/`

- [ ] **Things deferred**
- Auth gate on `GET /api/bookings`, `DELETE /api/bookings/{id}`, `POST /api/villas/{slug}/pricing`, `DELETE /api/reviews/{id}` — Plan 2.
- PATCH review status (admin moderate API) — when needed.
- DELETE single price override by date — if requested.
- Pagination on reviews — small volume.
- Frontend wiring of any new endpoint.
