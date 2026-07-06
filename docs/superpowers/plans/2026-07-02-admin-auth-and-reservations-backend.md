# Admin Auth + Reservations — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Commit policy:** DO NOT run `git commit` during execution of this plan. Make file changes only. The user runs commits manually at their preferred boundaries. (Memory: `feedback-no-auto-commit`.)

**Goal:** Add real admin authentication (email+password, signed session cookie, in-app user management) to the Go API, close the existing auth hole on `GET/PATCH/DELETE /api/bookings*`, and extend the booking domain with `villa_slug` filtering and a `source` field — everything the admin Reservations screen needs from the backend.

**Architecture:** New `internal/adminauth` module follows the existing hexagonal shape (domain → ports → service → http → postgres), decoupled from `booking` via a generic `func(http.Handler) http.Handler` middleware injected at the composition root — `booking` never imports `adminauth`. Sessions are a stateless HMAC-signed cookie (secret = existing unused `JWT_SECRET`), not a DB-backed session table.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, sqlc, golang-migrate, golang.org/x/crypto/bcrypt (already an indirect dependency, promoted to direct), testcontainers-go.

**Spec reference:** `docs/superpowers/specs/2026-07-02-admin-auth-and-reservations-design.md`

**Working dir for all paths below:** `apps/api/` unless stated otherwise.

## Global Constraints

- No `git commit` during execution — file changes only (see Commit policy above).
- `booking` package must not import `adminauth` (module isolation, spec §2.4/§4.2).
- Session TTL is 12h; cookie name is `casa_admin_session`; secret comes from `config.Config.JWTSecret`.
- `POST /api/bookings` and `GET /api/villas/{slug}/availability` stay unauthenticated; `GET/PATCH/DELETE /api/bookings*` require a valid admin session.
- Self-delete of your own admin account is always blocked (this alone guarantees at least one admin always exists — see Task C3 note on why the spec's separate "last admin" check was dropped as unreachable).
- Orval regen (Task E2) is in scope — it's the only real validation of the OpenAPI changes. Actually wiring the generated hooks into `apps/web` React components is out of scope; that's the separate frontend plan.

---

## Phase A — Config, migrations, sqlc

### Task A1: Add `COOKIE_SECURE` config field

**Files:**
- Modify: `apps/api/internal/platform/config/config.go`
- Modify: `apps/api/internal/platform/config/config_test.go`
- Modify: `.env.example` (repo root)
- Modify: `.env.dev` (repo root)

**Interfaces:**
- Produces: `config.Config.CookieSecure bool`, consumed by Task D1 (composition root) and Task C4 (adminauth http.go).

- [ ] **Step 1: Write the failing test**

Add to `internal/platform/config/config_test.go` (same file, new test function — check the existing `TestLoad_ReadsRequiredEnv` first to match its `t.Setenv` list before adding this):

```go
func TestLoad_CookieSecureDefaultsTrue(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "casadana")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("MAIL_FROM", "no-reply@casadana.com")
	t.Setenv("WEB_ORIGIN", "http://localhost:5173")
	t.Setenv("ADMIN_NOTIFY_EMAIL", "owner@casadana.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure default = false, want true")
	}
}

func TestLoad_CookieSecureFalseOverride(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "casadana")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("MAIL_FROM", "no-reply@casadana.com")
	t.Setenv("WEB_ORIGIN", "http://localhost:5173")
	t.Setenv("ADMIN_NOTIFY_EMAIL", "owner@casadana.com")
	t.Setenv("COOKIE_SECURE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, want false after override")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/config/...`
Expected: FAIL — `cfg.CookieSecure` doesn't compile (field doesn't exist yet).

- [ ] **Step 3: Add the field**

In `internal/platform/config/config.go`, add to the `Config` struct (after `MigrateOnBoot`):

```go
	CookieSecure     bool   `env:"COOKIE_SECURE" envDefault:"true"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/config/...`
Expected: PASS (all tests, including the two new ones).

- [ ] **Step 5: Update env files**

Add `COOKIE_SECURE=false` to repo-root `.env.dev` (local dev runs over plain HTTP) right after `WEB_ORIGIN=http://localhost:5173`. Add `COOKIE_SECURE=true` to repo-root `.env.example` after the `PORT=8080` line, with no comment needed (defaults to true if omitted, but explicit is clearer for the example file).

---

### Task A2: Migrations — `admin_users` table and `bookings.source` column

**Files:**
- Create: `apps/api/internal/db/migrations/0004_admin_users.up.sql`
- Create: `apps/api/internal/db/migrations/0004_admin_users.down.sql`
- Create: `apps/api/internal/db/migrations/0005_booking_source.up.sql`
- Create: `apps/api/internal/db/migrations/0005_booking_source.down.sql`

- [ ] **Step 1: Write the migrations**

`internal/db/migrations/0004_admin_users.up.sql`:

```sql
CREATE TABLE admin_users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

`internal/db/migrations/0004_admin_users.down.sql`:

```sql
DROP TABLE IF EXISTS admin_users;
```

`internal/db/migrations/0005_booking_source.up.sql`:

```sql
ALTER TABLE bookings ADD COLUMN source TEXT NOT NULL DEFAULT 'direct';
```

`internal/db/migrations/0005_booking_source.down.sql`:

```sql
ALTER TABLE bookings DROP COLUMN source;
```

- [ ] **Step 2: Verify migration file naming matches existing convention**

Run: `ls internal/db/migrations/`
Expected: `0001_init.up.sql`, `0001_init.down.sql`, `0002_price_overrides.up.sql`, `0002_price_overrides.down.sql`, `0003_reviews.up.sql`, `0003_reviews.down.sql`, plus the four new `0004_*`/`0005_*` files.

- [ ] **Step 3: Sanity-check SQL syntax with a throwaway local Postgres (optional but recommended)**

If Docker is available: `docker run --rm -e POSTGRES_PASSWORD=x -p 5433:5432 -d --name casadana-migrate-check postgres:16-alpine`, wait ~3s, then `docker exec -i casadana-migrate-check psql -U postgres -c "$(cat internal/db/migrations/0004_admin_users.up.sql)"` (should print `CREATE TABLE`) — this requires the `uuid_generate_v4()` extension already enabled by `0001_init.up.sql` in a full migration run, so this quick check will fail on a bare Postgres; that's expected and fine, skip if it's inconvenient. Clean up: `docker rm -f casadana-migrate-check`. The real validation happens in Task C5/B7's `testcontainers` integration tests, which run the full migration chain.

---

### Task A3: sqlc queries — `admin_users.sql` and `bookings.sql` additions

**Files:**
- Create: `apps/api/internal/db/queries/admin_users.sql`
- Modify: `apps/api/internal/db/queries/bookings.sql`
- Generated (sqlc): `apps/api/internal/db/models.go`, `apps/api/internal/db/querier.go`, `apps/api/internal/db/bookings.sql.go`, `apps/api/internal/db/admin_users.sql.go` (new)

**Interfaces:**
- Produces (after `sqlc generate`): `db.AdminUser` struct, `db.Queries.InsertAdminUser/GetAdminUserByEmail/GetAdminUserByID/ListAdminUsers/DeleteAdminUser`, and updated `db.Booking` (gains `Source string`), `db.Queries.InsertBooking` (gains `Source` param), plus 4 new villa-filtered list/count methods. Consumed by Task C5 (adminauth postgres.go) and Task B4 (booking postgres.go).

- [ ] **Step 1: Create `internal/db/queries/admin_users.sql`**

```sql
-- name: InsertAdminUser :one
INSERT INTO admin_users (id, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAdminUserByEmail :one
SELECT * FROM admin_users WHERE email = $1;

-- name: GetAdminUserByID :one
SELECT * FROM admin_users WHERE id = $1;

-- name: ListAdminUsers :many
SELECT * FROM admin_users ORDER BY created_at ASC;

-- name: DeleteAdminUser :execrows
DELETE FROM admin_users WHERE id = $1;
```

- [ ] **Step 2: Modify `internal/db/queries/bookings.sql`**

Replace the existing `InsertBooking` query (source column added) and append the 4 new villa-filtered queries at the end of the file. The full modified file:

```sql
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
```

- [ ] **Step 3: Generate sqlc code**

Run: `sqlc generate` (from `apps/api/`)
Expected: no output on success. If `sqlc` is not installed: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` first.

- [ ] **Step 4: Verify generated code**

Run: `grep -n "Source" internal/db/models.go`
Expected: `Source string` appears in the `Booking` struct.

Run: `grep -n "AdminUser" internal/db/models.go`
Expected: a new `type AdminUser struct { ID pgtype.UUID; Email string; PasswordHash string; CreatedAt pgtype.Timestamptz }`.

Run: `go build ./internal/db/...`
Expected: fails at this point — `internal/booking/postgres.go`'s existing call to `InsertBooking` no longer compiles because `InsertBookingParams` gained a required `Source` field it isn't setting yet, and the villa/status branches in `internal/booking/service.go`/`ports.go` don't exist yet either. **This is expected** — Phase B fixes it. Do not attempt to fix `booking` in this task.

---

## Phase B — `booking` module: `source` field, `villa_slug` filter, auth-ready `Mount`

### Task B1: Domain — `Source` field on `Booking`

**Files:**
- Modify: `apps/api/internal/booking/domain.go`
- Modify: `apps/api/internal/booking/domain_test.go`

**Interfaces:**
- Produces: `Booking.Source string`, `NewBookingInput.Source string` (defaults to `"direct"` when empty).
- Consumes: nothing new.

- [ ] **Step 1: Write the failing tests**

Add to `internal/booking/domain_test.go`:

```go
func TestNewBooking_DefaultsSourceToDirect(t *testing.T) {
	c := validCmd()
	c.Source = ""
	b, err := NewBooking(c)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if b.Source != "direct" {
		t.Errorf("Source = %q, want direct", b.Source)
	}
}

func TestNewBooking_KeepsExplicitSource(t *testing.T) {
	c := validCmd()
	c.Source = "airbnb"
	b, err := NewBooking(c)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if b.Source != "airbnb" {
		t.Errorf("Source = %q, want airbnb", b.Source)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/booking/... -run TestNewBooking_DefaultsSourceToDirect`
Expected: FAIL — `c.Source` doesn't compile (field doesn't exist on `NewBookingInput`).

- [ ] **Step 3: Add `Source` to the domain**

In `internal/booking/domain.go`:

Add `Source string` to the `Booking` struct, right after `Status Status`:

```go
type Booking struct {
	ID         string
	VillaSlug  string
	GuestName  string
	GuestEmail string
	GuestPhone string
	CheckIn    time.Time
	CheckOut   time.Time
	Adults     int
	Children   int
	Message    string
	Status     Status
	Source     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
```

Add `Source string` to `NewBookingInput`, right after `Message string`:

```go
type NewBookingInput struct {
	VillaSlug  string
	GuestName  string
	GuestEmail string
	GuestPhone string
	CheckIn    time.Time
	CheckOut   time.Time
	Adults     int
	Children   int
	Message    string
	Source     string
	Now        time.Time
}
```

In `NewBooking`, add source normalization right after the existing `in.VillaSlug = strings.TrimSpace(in.VillaSlug)` line:

```go
	in.Source = strings.TrimSpace(in.Source)
	if in.Source == "" {
		in.Source = "direct"
	}
```

And add `Source: in.Source,` to the `&Booking{...}` literal at the end of `NewBooking`, right after `Message: in.Message,`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/booking/... -run TestNewBooking`
Expected: PASS (all `TestNewBooking_*` tests, including the two new ones and the pre-existing `TestNewBooking_Happy`/`TestNewBooking_Rejects`).

---

### Task B2: Ports — `villaSlug` filter on `List`/`Count`

**Files:**
- Modify: `apps/api/internal/booking/ports.go`
- Modify: `apps/api/internal/booking/fakes_test.go`

**Interfaces:**
- Produces: `Repository.List(ctx, villaSlug *string, status *Status, limit, offset int) ([]Booking, error)`, `Repository.Count(ctx, villaSlug *string, status *Status) (int, error)` — consumed by Task B3 (service.go) and Task B4 (postgres.go).
- Consumes: nothing new (this task only changes the interface + its fake; the real implementation and callers are updated in B3/B4).

- [ ] **Step 1: Update the interface**

In `internal/booking/ports.go`, change:

```go
	List(ctx context.Context, status *Status, limit, offset int) ([]Booking, error)
	Count(ctx context.Context, status *Status) (int, error)
```

to:

```go
	List(ctx context.Context, villaSlug *string, status *Status, limit, offset int) ([]Booking, error)
	Count(ctx context.Context, villaSlug *string, status *Status) (int, error)
```

- [ ] **Step 2: Run build to confirm it's broken everywhere it should be**

Run: `go build ./internal/booking/...`
Expected: FAIL — `fakeRepo.List`/`fakeRepo.Count` (in `fakes_test.go`) and `Service.List` (in `service.go`) no longer satisfy the interface / no longer compile against it. This is expected; fixed in this task's Step 3 and Task B3.

- [ ] **Step 3: Update `fakeRepo` in `fakes_test.go`**

Replace the existing `List` and `Count` methods on `fakeRepo` with:

```go
func (f *fakeRepo) List(_ context.Context, villaSlug *string, status *Status, limit, offset int) ([]Booking, error) {
	filtered := make([]Booking, 0, len(f.saved))
	for _, b := range f.saved {
		if villaSlug != nil && b.VillaSlug != *villaSlug {
			continue
		}
		if status != nil && b.Status != *status {
			continue
		}
		filtered = append(filtered, b)
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

func (f *fakeRepo) Count(_ context.Context, villaSlug *string, status *Status) (int, error) {
	n := 0
	for _, b := range f.saved {
		if villaSlug != nil && b.VillaSlug != *villaSlug {
			continue
		}
		if status != nil && b.Status != *status {
			continue
		}
		n++
	}
	return n, nil
}
```

- [ ] **Step 4: Run build again**

Run: `go build ./internal/booking/...`
Expected: still FAILS — `service.go`'s `Service.List` method still calls the old 2-arg repo signature. Fixed in Task B3. Confirm the *only* remaining error mentions `service.go`, not `fakes_test.go`, to be sure Step 3 succeeded.

---

### Task B3: Service — `villaSlug` param on `List`, allowlist validation

**Files:**
- Modify: `apps/api/internal/booking/service.go`
- Modify: `apps/api/internal/booking/service_test.go`

**Interfaces:**
- Produces: `Service.List(ctx, villaSlug *string, status *Status, page, limit int) ([]Booking, int, error)` — consumed by Task B5 (http.go handler).
- Consumes: `Repository.List`/`Count` from Task B2, `VillaAllowlist.IsKnown` (already exists).

- [ ] **Step 1: Write the failing tests**

Add to `internal/booking/service_test.go`:

```go
func TestList_FilterByVillaSlug(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", VillaSlug: "casadana", Status: StatusPending},
			{ID: "2", VillaSlug: "casacasay", Status: StatusPending},
			{ID: "3", VillaSlug: "casadana", Status: StatusApproved},
		},
	}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true, "casacasay": true}}
	svc := newSvc(repo, &fakeMailer{}, allow, d("2026-05-12"))

	slug := "casadana"
	bookings, total, err := svc.List(context.Background(), &slug, nil, 1, 50)
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

func TestList_UnknownVillaSlug(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{}}, d("2026-05-12"))

	slug := "ghost-villa"
	_, _, err := svc.List(context.Background(), &slug, nil, 1, 20)
	if err == nil || !isErr(err, ErrUnknownVilla) {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/booking/... -run TestList_FilterByVillaSlug`
Expected: FAIL — `svc.List` doesn't accept 5 args yet (compile error).

- [ ] **Step 3: Update `Service.List`**

In `internal/booking/service.go`, replace the `List` method:

```go
// List returns a page of bookings ordered by created_at DESC, with optional
// villa_slug and status filters. page is 1-based, limit is clamped to
// [1, 100], default 20.
func (s *Service) List(ctx context.Context, villaSlug *string, status *Status, page, limit int) ([]Booking, int, error) {
	if villaSlug != nil && !s.allow.IsKnown(*villaSlug) {
		return nil, 0, ErrUnknownVilla
	}
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

	bookings, err := s.repo.List(ctx, villaSlug, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("booking: list: %w", err)
	}
	total, err := s.repo.Count(ctx, villaSlug, status)
	if err != nil {
		return nil, 0, fmt.Errorf("booking: count: %w", err)
	}
	return bookings, total, nil
}
```

- [ ] **Step 4: Update the 3 existing call sites in `service_test.go`**

`TestList_Pagination`, `TestList_StatusFilter`, and `TestList_Clamps` all call `svc.List(context.Background(), <status-or-nil>, page, limit)` with 4 args. Update each call to pass `nil` as the new first (villaSlug) argument — e.g. `svc.List(context.Background(), nil, nil, 1, 2)` in `TestList_Pagination`, `svc.List(context.Background(), nil, &pending, 1, 50)` in `TestList_StatusFilter`, and `svc.List(context.Background(), nil, nil, c.page, c.limit)` inside `TestList_Clamps`'s loop.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/booking/... -run TestList`
Expected: PASS — all 5 `TestList_*` tests (3 pre-existing + 2 new).

---

### Task B4: Postgres adapter — villa-filtered queries, `source` column

**Files:**
- Modify: `apps/api/internal/booking/postgres.go`

**Interfaces:**
- Consumes: `db.Queries.ListBookingsPagedByVilla/ListBookingsPagedByVillaAndStatus/CountBookingsByVilla/CountBookingsByVillaAndStatus` (Task A3), `Repository.List`/`Count` signature (Task B2).
- Produces: working `pgRepo.List`/`Count`/`Save` against the real schema — consumed by Task B7 (integration test).

- [ ] **Step 1: Update `Save` to pass `Source`**

In `internal/booking/postgres.go`, in `Save`, add `Source: b.Source,` to the `db.InsertBookingParams{...}` literal, right after `Message: b.Message,`.

- [ ] **Step 2: Update `rowToBooking` to read `Source`**

Add `Source: row.Source,` to the `Booking{...}` literal in `rowToBooking`, right after `Message: row.Message,`.

- [ ] **Step 3: Replace `List` and `Count`**

Replace the existing `List` and `Count` methods with:

```go
func (r *pgRepo) List(ctx context.Context, villaSlug *string, status *Status, limit, offset int) ([]Booking, error) {
	switch {
	case villaSlug != nil && status != nil:
		rows, err := r.q().ListBookingsPagedByVillaAndStatus(ctx, db.ListBookingsPagedByVillaAndStatusParams{
			VillaSlug: *villaSlug,
			Status:    db.BookingStatus(*status),
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
		return mapBookingRows(rows, err)
	case villaSlug != nil:
		rows, err := r.q().ListBookingsPagedByVilla(ctx, db.ListBookingsPagedByVillaParams{
			VillaSlug: *villaSlug,
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
		return mapBookingRows(rows, err)
	case status != nil:
		rows, err := r.q().ListBookingsPagedByStatus(ctx, db.ListBookingsPagedByStatusParams{
			Status: db.BookingStatus(*status),
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		return mapBookingRows(rows, err)
	default:
		rows, err := r.q().ListBookingsPaged(ctx, db.ListBookingsPagedParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		return mapBookingRows(rows, err)
	}
}

func mapBookingRows(rows []db.Booking, err error) ([]Booking, error) {
	if err != nil {
		return nil, err
	}
	out := make([]Booking, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToBooking(row))
	}
	return out, nil
}

func (r *pgRepo) Count(ctx context.Context, villaSlug *string, status *Status) (int, error) {
	switch {
	case villaSlug != nil && status != nil:
		n, err := r.q().CountBookingsByVillaAndStatus(ctx, db.CountBookingsByVillaAndStatusParams{
			VillaSlug: *villaSlug,
			Status:    db.BookingStatus(*status),
		})
		return int(n), err
	case villaSlug != nil:
		n, err := r.q().CountBookingsByVilla(ctx, *villaSlug)
		return int(n), err
	case status != nil:
		n, err := r.q().CountBookingsByStatus(ctx, db.BookingStatus(*status))
		return int(n), err
	default:
		n, err := r.q().CountBookings(ctx)
		return int(n), err
	}
}
```

- [ ] **Step 4: Verify it builds**

Run: `go build ./internal/booking/...`
Expected: no output, exit 0. (This confirms sqlc's generated param struct field names match — if `ListBookingsPagedByVillaAndStatusParams` field names differ slightly, e.g. sqlc might order/name them differently; adjust the literal field names to match whatever `go build` reports.)

---

### Task B5: HTTP — `Mount` takes `requireAuth`, `source`/`villa_slug` in DTOs

**Files:**
- Modify: `apps/api/internal/booking/http.go`

**Interfaces:**
- Produces: `Mount(r chi.Router, svc *Service, requireAuth func(http.Handler) http.Handler)` — consumed by Task D1 (composition root) and Task B6 (http_test.go).
- Consumes: `Service.List` (Task B3), `Service.Create`'s `CreateCommand` (extended below).

- [ ] **Step 1: Add `Source` to `createBookingRequest` and `bookingResponse`; add `Adults`/`Children` to `bookingResponse`**

In `internal/booking/http.go`, add to `createBookingRequest` (after `Message string`):

```go
	Source     string `json:"source"      validate:"omitempty,oneof=direct airbnb booking_com"`
```

Add to `bookingResponse` (after `GuestEmail string`):

```go
	Source     string `json:"source"`
	Adults     int    `json:"adults"`
	Children   int    `json:"children"`
```

`Adults`/`Children` are already tracked on the `Booking` domain struct (set at creation) but were never exposed in the response DTO — the admin Reservations table needs the real guest count, not a placeholder, so this closes that gap while the file is already open for the `source` change.

Update `toResponse` to include, right after `GuestEmail: b.GuestEmail,`:

```go
		Source:     b.Source,
		Adults:     b.Adults,
		Children:   b.Children,
```

- [ ] **Step 2: Pass `Source` through `createHandler`**

`Service.Create`'s `CreateCommand` struct (in `service.go`) needs a `Source string` field too — add it right after `Message string` in `CreateCommand`, and pass it into `NewBookingInput{..., Source: cmd.Source, ...}` inside `Service.Create`.

In `createHandler` (`http.go`), add `Source: req.Source,` to the `booking.CreateCommand{...}` — wait, it's already in the same package (`booking`), so just `CreateCommand{...}` — literal, right after `Message: req.Message,`.

- [ ] **Step 3: Add `villa_slug` query param to `listBookingsHandler`**

Replace `listBookingsHandler`:

```go
func listBookingsHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		var villaSlugFilter *string
		if vs := r.URL.Query().Get("villa_slug"); vs != "" {
			villaSlugFilter = &vs
		}

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

		bookings, total, err := svc.List(r.Context(), villaSlugFilter, statusFilter, page, limit)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
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
```

- [ ] **Step 4: Change `Mount` to accept `requireAuth` and group the admin-only routes**

Replace `Mount`:

```go
// Mount wires booking routes. requireAuth guards the admin-only routes
// (list/patch/delete); POST (public booking creation) and the availability
// read stay open for unauthenticated guests.
func Mount(r chi.Router, svc *Service, requireAuth func(http.Handler) http.Handler) {
	r.Post("/api/bookings", createHandler(svc))
	r.Get("/api/villas/{slug}/availability", availabilityHandler(svc))
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/api/bookings", listBookingsHandler(svc))
		r.Patch("/api/bookings/{id}", patchBookingHandler(svc))
		r.Delete("/api/bookings/{id}", deleteBookingHandler(svc))
	})
}
```

- [ ] **Step 5: Verify it builds**

Run: `go build ./internal/booking/...`
Expected: FAILS — `http_test.go`'s `newRouter(svc)` helper still calls the old 2-arg `Mount`. This is expected; fixed in Task B6.

---

### Task B6: HTTP tests — update `newRouter`, add auth-gating and source/villa_slug tests

**Files:**
- Modify: `apps/api/internal/booking/http_test.go`

- [ ] **Step 1: Update the `newRouter` helper and add middleware fixtures**

Replace:

```go
func newRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc)
	return r
}
```

with:

```go
func newRouter(svc *Service, requireAuth func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc, requireAuth)
	return r
}

// noopAuth lets tests that aren't about auth exercise the admin-only routes
// as if a valid session were already present.
func noopAuth(next http.Handler) http.Handler { return next }

// denyAllAuth simulates a middleware that rejects every request — used to
// prove requireAuth is actually wired onto the routes that need it (and not
// onto the ones that shouldn't be gated).
func denyAllAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
}
```

- [ ] **Step 2: Update all existing `newRouter(svc)` call sites to `newRouter(svc, noopAuth)`**

Every test in the file calls `newRouter(svc)` — there are 9 call sites (`TestPostBooking_Created`, `TestPostBooking_ValidationError`, `TestPostBooking_DatesConflict`, `TestGetAvailability_Empty`, `TestGetAvailability_SeparatesPendingFromBooked`, `TestListBookings_PaginatedResponse`, `TestListBookings_BadStatus`, `TestDeleteBooking_NoContent`, `TestDeleteBooking_NotFound`). Change each to `newRouter(svc, noopAuth)` so existing behavior is unaffected by the new auth gate.

- [ ] **Step 3: Add auth-gating tests**

Append to the file:

```go
func TestListBookings_RequiresAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/bookings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPatchBooking_RequiresAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/bookings/abc", strings.NewReader(`{"status":"approved"}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestDeleteBooking_RequiresAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/bookings/abc", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestPostBooking_NotGatedByAuth(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33123","check_in":"2026-07-01","check_out":"2026-07-08","adults":2,"children":0,"message":"hi"}`
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (POST must not be auth-gated even when requireAuth denies everything)", resp.StatusCode)
	}
}

func TestGetAvailability_NotGatedByAuth(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, denyAllAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/availability?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
```

- [ ] **Step 4: Add `source`/`villa_slug` behavior tests**

Append:

```go
func TestPostBooking_DefaultsSourceToDirect(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33123","check_in":"2026-07-01","check_out":"2026-07-08","adults":2,"children":0}`
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out bookingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Source != "direct" {
		t.Errorf("Source = %q, want direct", out.Source)
	}
}

func TestListBookings_FilterByVillaSlug(t *testing.T) {
	repo := &fakeRepo{
		saved: []Booking{
			{ID: "1", VillaSlug: "casadana", GuestName: "A", GuestEmail: "a@x.com", CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08")},
			{ID: "2", VillaSlug: "casacasay", GuestName: "B", GuestEmail: "b@x.com", CheckIn: d("2026-08-01"), CheckOut: d("2026-08-08")},
		},
	}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true, "casacasay": true}}, d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc, noopAuth))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/bookings?villa_slug=casadana")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out listBookingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Total != 1 || len(out.Bookings) != 1 || out.Bookings[0].VillaSlug != "casadana" {
		t.Errorf("out = %+v, want exactly the casadana booking", out)
	}
}
```

- [ ] **Step 5: Add the missing imports**

The new tests use `strings` (already imported) and `encoding/json` (already imported) — no new imports needed. Confirm with:

Run: `go build ./internal/booking/...`
Expected: no output, exit 0.

- [ ] **Step 6: Run the full booking test suite**

Run: `go test ./internal/booking/...`
Expected: PASS — all tests (pre-existing + new).

---

### Task B7: Postgres integration test — villa_slug filter, and fix the pre-existing broken `MigrateUp` call

**Files:**
- Modify: `apps/api/internal/booking/postgres_test.go`

**Note:** the current `setupPg` helper in this file calls `pg.MigrateUp(pool, "file://"+abs)` — a 2-argument call against a function that now takes 3 (`pool, fs.FS, subPath`; see `internal/pricing/postgres_test.go` for the correct, currently-working pattern). This test file does not compile under `-tags integration` today. Since this task touches this exact file, fix it to match the working pattern instead of leaving it broken.

- [ ] **Step 1: Replace `setupPg` with the working pattern**

Replace the whole `setupPg` function and its imports block:

```go
//go:build integration

package booking

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	dbpkg "github.com/TheHikuro/casadana/internal/db"
	pg "github.com/TheHikuro/casadana/internal/platform/postgres"
)

func setupPg(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("casadana_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	pool, err := pg.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pg.MigrateUp(pool, dbpkg.Migrations, "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}
```

(This removes the now-unused `path/filepath` import — the old version used `filepath.Abs`, the new one doesn't.)

- [ ] **Step 2: Add a villa_slug filter integration test**

Append to the file:

```go
func TestPgRepo_ListFiltersByVillaSlug(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	for _, slug := range []string{"casadana", "casacasay"} {
		b, err := NewBooking(NewBookingInput{
			VillaSlug: slug, GuestName: "Jane", GuestEmail: "jane@example.com",
			CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
			Adults: 2, Now: d("2026-05-12"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, b); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	slug := "casadana"
	rows, err := repo.List(ctx, &slug, nil, 20, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].VillaSlug != "casadana" {
		t.Errorf("rows = %+v, want exactly 1 casadana booking", rows)
	}

	count, err := repo.Count(ctx, &slug, nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
```

- [ ] **Step 3: Run the integration tests (requires Docker running)**

Run: `go test -tags integration ./internal/booking/...`
Expected: PASS. If Docker isn't available in the execution environment, note that in the task result instead of skipping silently — this is the only way to validate the SQL actually works end-to-end.

---

## Phase C — New `internal/adminauth` module

### Task C1: Dependencies

**Files:**
- Modify: `apps/api/go.mod`, `apps/api/go.sum`

- [ ] **Step 1: Promote bcrypt to a direct dependency**

Run (from `apps/api/`): `go get golang.org/x/crypto/bcrypt`
Expected: `go.mod` now lists `golang.org/x/crypto` under the direct `require` block (it was already present as `// indirect` at `v0.49.0`; this just drops the indirect marker and confirms the version resolves).

- [ ] **Step 2: Verify**

Run: `go build ./...`
Expected: no output, exit 0 (nothing references bcrypt yet, but the dependency must resolve cleanly before Task C2 uses it).

---

### Task C2: Domain — `AdminUser`, password hashing, session tokens

**Files:**
- Create: `apps/api/internal/adminauth/domain.go`
- Create: `apps/api/internal/adminauth/domain_test.go`

**Interfaces:**
- Produces: `AdminUser{ID, Email, PasswordHash, CreatedAt}`, `hashPassword`, `verifyPassword`, `signToken`, `verifyToken`, sentinel errors `ErrInvalidCredentials/ErrEmailTaken/ErrNotFound/ErrCannotDeleteSelf/ErrInvalidToken` — consumed by Task C3 (service.go), Task C4 (http.go), Task C5 (postgres.go).

- [ ] **Step 1: Write the failing tests**

`internal/adminauth/domain_test.go`:

```go
package adminauth

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestSignVerifyToken_RoundTrip(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret", now)
	id, err := verifyToken(token, "secret", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("verifyToken: %v", err)
	}
	if id != "admin-1" {
		t.Errorf("id = %q, want admin-1", id)
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret", now)
	if _, err := verifyToken(token, "secret", now.Add(13*time.Hour)); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_TamperedSignature(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret", now)
	tampered := token[:len(token)-1] + "0"
	if _, err := verifyToken(tampered, "secret", now); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	now := d("2026-07-02")
	token := signToken("admin-1", "secret-a", now)
	if _, err := verifyToken(token, "secret-b", now); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestVerifyToken_Malformed(t *testing.T) {
	if _, err := verifyToken("not-a-token", "secret", d("2026-07-02")); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := hashPassword("correcthorse")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !verifyPassword(hash, "correcthorse") {
		t.Error("verifyPassword should accept the correct password")
	}
	if verifyPassword(hash, "wrong") {
		t.Error("verifyPassword should reject the wrong password")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adminauth/...`
Expected: FAIL — package doesn't exist yet / build error (no `domain.go`).

- [ ] **Step 3: Write `domain.go`**

```go
package adminauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AdminUser struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrNotFound           = errors.New("admin user not found")
	ErrCannotDeleteSelf   = errors.New("cannot delete your own account")
	ErrInvalidToken       = errors.New("invalid or expired session")
)

const sessionTTL = 12 * time.Hour

func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("adminauth: hash password: %w", err)
	}
	return string(b), nil
}

func verifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// signToken produces "<adminID>.<expiryUnix>.<hexHMAC>". adminID is a UUID
// (never contains '.'), so '.' is a safe delimiter.
func signToken(adminID string, secret string, now time.Time) string {
	exp := now.Add(sessionTTL).Unix()
	payload := fmt.Sprintf("%s.%d", adminID, exp)
	sig := hex.EncodeToString(hmacSum(payload, secret))
	return payload + "." + sig
}

func verifyToken(token, secret string, now time.Time) (adminID string, err error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return "", ErrInvalidToken
	}
	adminID, expStr, sig := parts[0], parts[1], parts[2]
	payload := adminID + "." + expStr
	expected := hex.EncodeToString(hmacSum(payload, secret))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", ErrInvalidToken
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() > exp {
		return "", ErrInvalidToken
	}
	return adminID, nil
}

func hmacSum(payload, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adminauth/...`
Expected: PASS (all 6 tests).

---

### Task C3: Ports + Service — login, authenticate, user management

**Files:**
- Create: `apps/api/internal/adminauth/ports.go`
- Create: `apps/api/internal/adminauth/service.go`
- Create: `apps/api/internal/adminauth/fakes_test.go`
- Create: `apps/api/internal/adminauth/service_test.go`

**Interfaces:**
- Produces: `Service.Login(ctx, email, password) (token string, admin *AdminUser, err error)`, `Service.Authenticate(ctx, token) (*AdminUser, error)`, `Service.CreateUser(ctx, email, password) (*AdminUser, error)`, `Service.ListUsers(ctx) ([]AdminUser, error)`, `Service.DeleteUser(ctx, callerID, targetID string) error` — all consumed by Task C4 (http.go).
- Consumes: `Repository`, `Clock` ports (defined in this task), `AdminUser`/token/hash helpers from Task C2.

**Design note — dropped the spec's separate "last admin" check:** the spec (§2.3) sketched two guardrails on delete: `ErrCannotDeleteSelf` and a `count == 1` `ErrLastAdmin` check. Since self-delete is unconditionally blocked, and every `DeleteUser` caller must themselves be an existing admin, the only way `admin_users` could ever reach 0 rows would require a caller to delete their own last-remaining row — already blocked by `ErrCannotDeleteSelf` alone. The `ErrLastAdmin` branch is therefore unreachable dead code; this plan implements only `ErrCannotDeleteSelf`, which provably keeps at least one admin row at all times.

- [ ] **Step 1: Write `ports.go`**

```go
package adminauth

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, u *AdminUser) error // ErrEmailTaken on unique conflict
	FindByEmail(ctx context.Context, email string) (*AdminUser, error) // ErrNotFound
	FindByID(ctx context.Context, id string) (*AdminUser, error)       // ErrNotFound
	List(ctx context.Context) ([]AdminUser, error)
	Delete(ctx context.Context, id string) error // ErrNotFound if 0 rows affected
}

type Clock interface {
	Now() time.Time
}
```

- [ ] **Step 2: Write `fakes_test.go`**

```go
package adminauth

import (
	"context"
	"time"
)

type fakeRepo struct {
	users   []AdminUser
	saveErr error
}

func (f *fakeRepo) Save(_ context.Context, u *AdminUser) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	for _, existing := range f.users {
		if existing.Email == u.Email {
			return ErrEmailTaken
		}
	}
	f.users = append(f.users, *u)
	return nil
}

func (f *fakeRepo) FindByEmail(_ context.Context, email string) (*AdminUser, error) {
	for i := range f.users {
		if f.users[i].Email == email {
			u := f.users[i]
			return &u, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) FindByID(_ context.Context, id string) (*AdminUser, error) {
	for i := range f.users {
		if f.users[i].ID == id {
			u := f.users[i]
			return &u, nil
		}
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) List(_ context.Context) ([]AdminUser, error) {
	return f.users, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	for i, u := range f.users {
		if u.ID == id {
			f.users = append(f.users[:i], f.users[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }
```

- [ ] **Step 3: Write the failing tests in `service_test.go`**

```go
package adminauth

import (
	"context"
	"testing"
)

func seedUser(t *testing.T, repo *fakeRepo, id, email, password string, now time.Time) AdminUser {
	t.Helper()
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	u := AdminUser{ID: id, Email: email, PasswordHash: hash, CreatedAt: now}
	repo.users = append(repo.users, u)
	return u
}

func TestLogin_Happy(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	token, admin, err := svc.Login(context.Background(), "loan@casa-dana.com", "correcthorse")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Error("token was empty")
	}
	if admin.Email != "loan@casa-dana.com" {
		t.Errorf("admin email = %q", admin.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	_, _, err := svc.Login(context.Background(), "loan@casa-dana.com", "wrong")
	if err != ErrInvalidCredentials {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")

	_, _, err := svc.Login(context.Background(), "ghost@casa-dana.com", "whatever")
	if err != ErrInvalidCredentials {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	token, _, err := svc.Login(context.Background(), "loan@casa-dana.com", "correcthorse")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	admin, err := svc.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if admin.Email != "loan@casa-dana.com" {
		t.Errorf("admin email = %q", admin.Email)
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")
	if _, err := svc.Authenticate(context.Background(), "garbage"); err != ErrInvalidToken {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	_, err := svc.CreateUser(context.Background(), "loan@casa-dana.com", "anotherpassword")
	if err != ErrEmailTaken {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestCreateUser_Happy(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	admin, err := svc.CreateUser(context.Background(), "new@casa-dana.com", "supersecret1")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if admin.Email != "new@casa-dana.com" {
		t.Errorf("email = %q", admin.Email)
	}
	if len(repo.users) != 1 {
		t.Errorf("repo.users len = %d, want 1", len(repo.users))
	}
}

func TestDeleteUser_CannotDeleteSelf(t *testing.T) {
	repo := &fakeRepo{}
	a := seedUser(t, repo, "admin-1", "loan@casa-dana.com", "pw", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	if err := svc.DeleteUser(context.Background(), a.ID, a.ID); err != ErrCannotDeleteSelf {
		t.Fatalf("err = %v, want ErrCannotDeleteSelf", err)
	}
}

func TestDeleteUser_Happy(t *testing.T) {
	repo := &fakeRepo{}
	a := seedUser(t, repo, "admin-1", "loan@casa-dana.com", "pw1", d("2026-07-02"))
	b := seedUser(t, repo, "admin-2", "co-host@casa-dana.com", "pw2", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")

	if err := svc.DeleteUser(context.Background(), a.ID, b.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if len(repo.users) != 1 {
		t.Errorf("repo.users len = %d, want 1", len(repo.users))
	}
}
```

Note: this file uses `time.Time` in `seedUser`'s signature — add `"time"` to the import block (`context`, `testing`, `time`).

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/adminauth/...`
Expected: FAIL — `NewService` doesn't exist yet.

- [ ] **Step 5: Write `service.go`**

```go
package adminauth

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo   Repository
	clock  Clock
	secret string
}

func NewService(repo Repository, clock Clock, secret string) *Service {
	return &Service{repo: repo, clock: clock, secret: secret}
}

// Login returns a signed session token and the authenticated user, or
// ErrInvalidCredentials for either an unknown email or a wrong password
// (deliberately not distinguished, to avoid leaking which one was wrong).
func (s *Service) Login(ctx context.Context, email, password string) (string, *AdminUser, error) {
	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if !verifyPassword(u.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}
	token := signToken(u.ID, s.secret, s.clock.Now())
	return token, u, nil
}

// Authenticate validates a session token and returns the current admin user.
// Returns ErrInvalidToken for a malformed/expired/tampered token, or if the
// admin it names was deleted after the token was issued.
func (s *Service) Authenticate(ctx context.Context, token string) (*AdminUser, error) {
	adminID, err := verifyToken(token, s.secret, s.clock.Now())
	if err != nil {
		return nil, ErrInvalidToken
	}
	u, err := s.repo.FindByID(ctx, adminID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return u, nil
}

func (s *Service) CreateUser(ctx context.Context, email, password string) (*AdminUser, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &AdminUser{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    s.clock.Now(),
	}
	if err := s.repo.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]AdminUser, error) {
	return s.repo.List(ctx)
}

// DeleteUser removes an admin account. Blocking self-delete unconditionally
// (rather than only when you're the last admin) is sufficient on its own to
// guarantee at least one admin always remains — see Task C3's design note.
func (s *Service) DeleteUser(ctx context.Context, callerID, targetID string) error {
	if callerID == targetID {
		return ErrCannotDeleteSelf
	}
	return s.repo.Delete(ctx, targetID)
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/adminauth/...`
Expected: PASS (all tests from Task C2 + this task).

---

### Task C4: HTTP — `Mount`, handlers, `RequireAdminSession` middleware

**Files:**
- Create: `apps/api/internal/adminauth/http.go`
- Create: `apps/api/internal/adminauth/http_test.go`

**Interfaces:**
- Produces: `Mount(r chi.Router, svc *Service, cookieSecure bool)`, `RequireAdminSession(svc *Service) func(http.Handler) http.Handler` — both consumed by Task D1 (composition root; the latter also by `booking.Mount`, Task B5).
- Consumes: `Service.Login/Authenticate/CreateUser/ListUsers/DeleteUser` (Task C3).

- [ ] **Step 1: Write the failing tests**

`internal/adminauth/http_test.go`:

```go
package adminauth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func newRouter(svc *Service, cookieSecure bool) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc, cookieSecure)
	return r
}

func loginAndGetJar(t *testing.T, baseURL, email, password string) http.CookieJar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}
	body := `{"email":"` + email + `","password":"` + password + `"}`
	resp, err := client.Post(baseURL+"/api/admin/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
	return jar
}

func TestLogin_SetsCookie(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	body := `{"email":"loan@casa-dana.com","password":"correcthorse"}`
	resp, err := http.Post(srv.URL+"/api/admin/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var found *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			found = c
		}
	}
	if found == nil {
		t.Fatal("session cookie was not set")
	}
	if !found.HttpOnly || !found.Secure {
		t.Errorf("cookie attrs HttpOnly=%v Secure=%v, want both true", found.HttpOnly, found.Secure)
	}
}

func TestLogin_WrongPassword_401(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	body := `{"email":"loan@casa-dana.com","password":"wrong"}`
	resp, err := http.Post(srv.URL+"/api/admin/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMe_WithoutCookie_401(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/admin/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMe_WithValidCookie_200(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	client := &http.Client{Jar: loginAndGetJar(t, srv.URL, "loan@casa-dana.com", "correcthorse")}
	resp, err := client.Get(srv.URL + "/api/admin/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out adminUserDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Email != "loan@casa-dana.com" {
		t.Errorf("email = %q", out.Email)
	}
}

func TestLogout_ClearsCookie(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	client := &http.Client{Jar: loginAndGetJar(t, srv.URL, "loan@casa-dana.com", "correcthorse")}

	resp, err := client.Post(srv.URL+"/api/admin/logout", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	meResp, err := client.Get(srv.URL + "/api/admin/me")
	if err != nil {
		t.Fatal(err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status after logout = %d, want 401", meResp.StatusCode)
	}
}

func TestUsers_RequiresAuth(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/admin/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
```

Add `"github.com/go-chi/chi/v5"` to the import block (used by `newRouter`).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adminauth/...`
Expected: FAIL — `Mount`/`sessionCookieName`/`adminUserDTO` don't exist yet.

- [ ] **Step 3: Write `http.go`**

```go
package adminauth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/validator"
)

const sessionCookieName = "casa_admin_session"

func init() {
	httpserver.Register(ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS")
	httpserver.Register(ErrEmailTaken, http.StatusConflict, "EMAIL_TAKEN")
	httpserver.Register(ErrNotFound, http.StatusNotFound, "ADMIN_NOT_FOUND")
	httpserver.Register(ErrCannotDeleteSelf, http.StatusConflict, "CANNOT_DELETE_SELF")
	httpserver.Register(ErrInvalidToken, http.StatusUnauthorized, "UNAUTHORIZED")
}

func Mount(r chi.Router, svc *Service, cookieSecure bool) {
	r.Post("/api/admin/login", loginHandler(svc, cookieSecure))
	r.Post("/api/admin/logout", logoutHandler(cookieSecure))
	r.Group(func(r chi.Router) {
		r.Use(RequireAdminSession(svc))
		r.Get("/api/admin/me", meHandler())
		r.Get("/api/admin/users", listUsersHandler(svc))
		r.Post("/api/admin/users", createUserHandler(svc))
		r.Delete("/api/admin/users/{id}", deleteUserHandler(svc))
	})
}

type ctxKey struct{}

func adminFromContext(ctx context.Context) (*AdminUser, bool) {
	u, ok := ctx.Value(ctxKey{}).(*AdminUser)
	return u, ok
}

// RequireAdminSession validates the session cookie and injects the admin
// identity into the request context. Exported so other modules (e.g.
// booking) can gate their own routes without importing adminauth's
// internals — they only depend on this function's stdlib-shaped signature.
func RequireAdminSession(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				httpserver.WriteError(w, r, ErrInvalidToken)
				return
			}
			admin, err := svc.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				httpserver.WriteError(w, r, err)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, admin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func setSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

type adminUserDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func toDTO(u *AdminUser) adminUserDTO {
	return adminUserDTO{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt.Format(time.RFC3339)}
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func loginHandler(svc *Service, cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		token, admin, err := svc.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		setSessionCookie(w, token, cookieSecure)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toDTO(admin))
	}
}

func logoutHandler(cookieSecure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clearSessionCookie(w, cookieSecure)
		w.WriteHeader(http.StatusNoContent)
	}
}

func meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := adminFromContext(r.Context())
		if !ok {
			httpserver.WriteError(w, r, ErrInvalidToken)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toDTO(admin))
	}
}

type createUserRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func createUserHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		admin, err := svc.CreateUser(r.Context(), req.Email, req.Password)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toDTO(admin))
	}
}

type listUsersResponse struct {
	Users []adminUserDTO `json:"users"`
}

func listUsersHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := svc.ListUsers(r.Context())
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		resp := listUsersResponse{Users: make([]adminUserDTO, 0, len(users))}
		for i := range users {
			resp.Users = append(resp.Users, toDTO(&users[i]))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func deleteUserHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, ok := adminFromContext(r.Context())
		if !ok {
			httpserver.WriteError(w, r, ErrInvalidToken)
			return
		}
		id := chi.URLParam(r, "id")
		if err := svc.DeleteUser(r.Context(), caller.ID, id); err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adminauth/...`
Expected: PASS (all tests from C2/C3/C4).

---

### Task C5: Postgres adapter

**Files:**
- Create: `apps/api/internal/adminauth/postgres.go`
- Create: `apps/api/internal/adminauth/postgres_test.go`

**Interfaces:**
- Consumes: `db.Queries.InsertAdminUser/GetAdminUserByEmail/GetAdminUserByID/ListAdminUsers/DeleteAdminUser` (Task A3), `Repository` interface (Task C3).
- Produces: `NewPgRepo(pool) Repository` — consumed by Task D1 (composition root).

- [ ] **Step 1: Write `postgres.go`**

```go
package adminauth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *pgRepo) Save(ctx context.Context, u *AdminUser) error {
	id, err := uuid.Parse(u.ID)
	if err != nil {
		return fmt.Errorf("adminauth: invalid id: %w", err)
	}
	row, err := r.q().InsertAdminUser(ctx, db.InsertAdminUserParams{
		ID:           pgtype.UUID{Bytes: [16]byte(id), Valid: true},
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return err
	}
	u.CreatedAt = row.CreatedAt.Time
	return nil
}

func (r *pgRepo) FindByEmail(ctx context.Context, email string) (*AdminUser, error) {
	row, err := r.q().GetAdminUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u := rowToAdminUser(row)
	return &u, nil
}

func (r *pgRepo) FindByID(ctx context.Context, id string) (*AdminUser, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("adminauth: invalid id: %w", err)
	}
	row, err := r.q().GetAdminUserByID(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u := rowToAdminUser(row)
	return &u, nil
}

func (r *pgRepo) List(ctx context.Context) ([]AdminUser, error) {
	rows, err := r.q().ListAdminUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminUser, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToAdminUser(row))
	}
	return out, nil
}

func (r *pgRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("adminauth: invalid id: %w", err)
	}
	rows, err := r.q().DeleteAdminUser(ctx, pgtype.UUID{Bytes: [16]byte(uid), Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func rowToAdminUser(row db.AdminUser) AdminUser {
	idStr := ""
	if row.ID.Valid {
		u, _ := uuid.FromBytes(row.ID.Bytes[:])
		idStr = u.String()
	}
	return AdminUser{
		ID:           idStr,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
	}
}
```

- [ ] **Step 2: Write `postgres_test.go`** (mirrors `internal/pricing/postgres_test.go`'s working `setupPg` pattern exactly)

```go
//go:build integration

package adminauth

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	dbpkg "github.com/TheHikuro/casadana/internal/db"
	pg "github.com/TheHikuro/casadana/internal/platform/postgres"
)

func setupPg(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("casadana_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	pool, err := pg.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pg.MigrateUp(pool, dbpkg.Migrations, "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func TestPgRepo_SaveFindDelete(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	u := &AdminUser{ID: "11111111-1111-1111-1111-111111111111", Email: "loan@casa-dana.com", PasswordHash: "hash"}
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := repo.FindByEmail(ctx, "loan@casa-dana.com")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if found.ID != u.ID {
		t.Errorf("ID = %q, want %q", found.ID, u.ID)
	}

	if err := repo.Delete(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, u.ID); err != ErrNotFound {
		t.Fatalf("err after delete = %v, want ErrNotFound", err)
	}
}

func TestPgRepo_Save_DuplicateEmail(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	a := &AdminUser{ID: "11111111-1111-1111-1111-111111111111", Email: "dup@casa-dana.com", PasswordHash: "hash"}
	b := &AdminUser{ID: "22222222-2222-2222-2222-222222222222", Email: "dup@casa-dana.com", PasswordHash: "hash2"}
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("save a: %v", err)
	}
	if err := repo.Save(ctx, b); err != ErrEmailTaken {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}
```

- [ ] **Step 3: Run the integration tests (requires Docker running)**

Run: `go test -tags integration ./internal/adminauth/...`
Expected: PASS.

---

## Phase D — Composition root wiring

### Task D1: Wire `adminauth` into `cmd/server/casadana.go`

**Files:**
- Modify: `apps/api/cmd/server/casadana.go`

**Interfaces:**
- Consumes: `adminauth.NewService`, `adminauth.NewPgRepo`, `adminauth.Mount`, `adminauth.RequireAdminSession` (Phase C); `booking.Mount`'s new `requireAuth` param (Task B5); `cfg.CookieSecure` (Task A1).

- [ ] **Step 1: Add the import**

Add `"github.com/TheHikuro/casadana/internal/adminauth"` to the import block in `cmd/server/casadana.go`.

- [ ] **Step 2: Wire the service and update `Mount` calls**

Find this block:

```go
	mailer := email.NewMailer(cfg.ResendKey, cfg.MailFrom, cfg.AdminNotifyEmail)
	bookingSvc := booking.NewService(
		booking.NewPgRepo(pool),
		booking.NewResendMailer(mailer),
		slugAllowlist{},
		realClock{},
	)
	pricingSvc := pricing.NewService(pricing.NewPgRepo(pool), slugAllowlist{})
	reviewSvc := review.NewService(
		review.NewPgRepo(pool),
		bookingReaderAdapter{svc: bookingSvc},
		realClock{},
	)

	r := httpserver.NewRouter(log, cfg.WebOrigin)
	openapi.Mount(r)
	booking.Mount(r, bookingSvc)
	pricing.Mount(r, pricingSvc)
	review.Mount(r, reviewSvc)
```

Replace with:

```go
	mailer := email.NewMailer(cfg.ResendKey, cfg.MailFrom, cfg.AdminNotifyEmail)
	adminAuthSvc := adminauth.NewService(adminauth.NewPgRepo(pool), realClock{}, cfg.JWTSecret)
	bookingSvc := booking.NewService(
		booking.NewPgRepo(pool),
		booking.NewResendMailer(mailer),
		slugAllowlist{},
		realClock{},
	)
	pricingSvc := pricing.NewService(pricing.NewPgRepo(pool), slugAllowlist{})
	reviewSvc := review.NewService(
		review.NewPgRepo(pool),
		bookingReaderAdapter{svc: bookingSvc},
		realClock{},
	)

	r := httpserver.NewRouter(log, cfg.WebOrigin)
	openapi.Mount(r)
	adminauth.Mount(r, adminAuthSvc, cfg.CookieSecure)
	booking.Mount(r, bookingSvc, adminauth.RequireAdminSession(adminAuthSvc))
	pricing.Mount(r, pricingSvc)
	review.Mount(r, reviewSvc)
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: no output, exit 0.

---

## Phase E — OpenAPI

### Task E1: Extend `openapi.yaml`

**Files:**
- Modify: `apps/api/internal/openapi/openapi.yaml`

**Note:** this task is entirely about the OpenAPI document itself (consumed by the separate frontend plan's `orval generate` step). Read the existing `openapi.yaml` structure first (`components.schemas`, `paths`, existing `bookings` tag operations) and match its exact YAML style/indentation before adding to it — this plan doesn't reproduce the full file since it's large; follow the shape of the existing `/api/bookings` and `/api/reviews` path entries.

- [ ] **Step 1: Add the `adminSession` security scheme**

Under `components.securitySchemes` (create this key if it doesn't exist yet, as a sibling of `components.schemas`):

```yaml
components:
  securitySchemes:
    adminSession:
      type: apiKey
      in: cookie
      name: casa_admin_session
```

- [ ] **Step 2: Add new schemas** under `components.schemas`, matching the existing style (e.g. `BookingResponse`, `ErrorResponse`):

```yaml
    AdminLoginRequest:
      type: object
      required: [email, password]
      properties:
        email:
          type: string
          format: email
        password:
          type: string
    AdminUser:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        created_at:
          type: string
          format: date-time
    ListAdminUsersResponse:
      type: object
      properties:
        users:
          type: array
          items:
            $ref: '#/components/schemas/AdminUser'
    CreateAdminUserRequest:
      type: object
      required: [email, password]
      properties:
        email:
          type: string
          format: email
        password:
          type: string
          minLength: 8
    BookingSource:
      type: string
      enum: [direct, airbnb, booking_com]
```

- [ ] **Step 3: Add the `source` field to the existing `CreateBookingRequest` and `BookingResponse` schemas**

Find `CreateBookingRequest` under `components.schemas` and add:

```yaml
        source:
          $ref: '#/components/schemas/BookingSource'
```

(not in `required` — it's optional, server defaults to `direct`). Find `BookingResponse` and add the same `source` property, plus `adults`/`children` (both `type: integer`) — these were already tracked server-side but never exposed in the response schema; the admin Reservations table needs a real guest count.

- [ ] **Step 4: Add the `villa_slug` query param to `GET /api/bookings`**

Find the `listBookings` operation (`GET /api/bookings`) and add to its `parameters` list, alongside the existing `page`/`limit`/`status` params:

```yaml
        - name: villa_slug
          in: query
          required: false
          schema:
            type: string
```

- [ ] **Step 5: Add `security: [{ adminSession: [] }]` to the 3 now-protected booking operations**

On the `listBookings` (`GET /api/bookings`), `patchBooking` (`PATCH /api/bookings/{id}`), and `deleteBooking` (`DELETE /api/bookings/{id}`) operations, add:

```yaml
      security:
        - adminSession: []
```

Do **not** add this to `createBooking` (`POST /api/bookings`) or `getVillaAvailability`.

- [ ] **Step 6: Add the 6 new `admin` paths**

Add a new tag `admin` to the top-level `tags` list (matching the style of the existing `bookings`/`pricing`/`reviews` tags), then add these paths (following the exact structural style of the existing `/api/reviews` path block for request/response wiring):

```yaml
  /api/admin/login:
    post:
      tags: [admin]
      operationId: adminLogin
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AdminLoginRequest'
      responses:
        '200':
          description: Logged in
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AdminUser'
        '401':
          $ref: '#/components/responses/Error'
  /api/admin/logout:
    post:
      tags: [admin]
      operationId: adminLogout
      responses:
        '204':
          description: Logged out
  /api/admin/me:
    get:
      tags: [admin]
      operationId: adminMe
      security:
        - adminSession: []
      responses:
        '200':
          description: Current admin user
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AdminUser'
        '401':
          $ref: '#/components/responses/Error'
  /api/admin/users:
    get:
      tags: [admin]
      operationId: listAdminUsers
      security:
        - adminSession: []
      responses:
        '200':
          description: All admin users
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ListAdminUsersResponse'
    post:
      tags: [admin]
      operationId: createAdminUser
      security:
        - adminSession: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateAdminUserRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AdminUser'
        '409':
          $ref: '#/components/responses/Error'
  /api/admin/users/{id}:
    delete:
      tags: [admin]
      operationId: deleteAdminUser
      security:
        - adminSession: []
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '204':
          description: Deleted
        '409':
          $ref: '#/components/responses/Error'
```

**Note:** the `$ref: '#/components/responses/Error'` shorthand assumes the file already defines a reusable `components.responses.Error` referencing `ErrorResponse` — check the existing file for this pattern (used by `/api/reviews` error responses) and match it exactly; if no such reusable response exists and each operation inlines its error schema instead, use that same inline style here for consistency instead of introducing a new shared-response pattern.

- [ ] **Step 7: Confirm the embedded spec still loads**

Run: `go build ./internal/openapi/...`
Expected: no output, exit 0 (the embedded `openapi.yaml` is loaded via `//go:embed` in `openapi.go` — this only catches a missing file, not invalid YAML/OpenAPI structure; real validation happens in Task E2).

---

### Task E2: Regenerate the Orval client

**Files:**
- Modify (generated): `packages/api/src/generated/**` (new `admin/admin.ts`, updated `bookings/bookings.ts` and `schemas/*`)
- Modify: `packages/api/src/index.ts` (barrel export, if Orval's `indexFiles` doesn't already cover the new directory — check after running)

This is the real end-to-end validation of Task E1's YAML — malformed OpenAPI fails this step loudly, unlike the build check in Step 7 above. It also means the frontend plan starts from working generated hooks instead of having to debug the spec itself.

- [ ] **Step 1: Run codegen**

Run (from repo root): `bun --filter @casa-dana/api generate`
Expected: regenerates `packages/api/src/generated/`. If this fails with a YAML/schema error, fix `openapi.yaml` (Task E1) and re-run — do not proceed until it succeeds.

- [ ] **Step 2: Verify the new hooks exist**

Run: `grep -E "^export (function|const) use" packages/api/src/generated/admin/admin.ts`
Expected: lists `useAdminLogin`, `useAdminLogout`, `useAdminMe`, `useListAdminUsers`, `useCreateAdminUser`, `useDeleteAdminUser` (exact names depend on Orval's operationId-to-hook-name convention — confirm against whatever it actually emits).

Run: `grep -n "source" packages/api/src/generated/schemas/bookingResponse.ts packages/api/src/generated/schemas/createBookingRequest.ts`
Expected: both files reference the new `source`/`BookingSource` field.

- [ ] **Step 3: Verify the barrel still exports everything**

Run: `cat packages/api/src/index.ts`
Expected: either Orval's `indexFiles: true` config already re-exports the new `admin` directory automatically (most likely, matching how `reviews` was picked up automatically per the `2026-05-15-bookings-pricing-reviews-batch` plan), or a manual `export * from "./generated/admin/admin"` needs adding — add it if missing.

- [ ] **Step 4: Type-check the (still backend-only-consumed) package**

Run (from repo root): `bun --filter @casa-dana/api exec tsc --noEmit` (or `cd packages/api && bunx tsc --noEmit` if the workspace filter syntax differs — check `packages/api/package.json` for an existing `typecheck`/`build` script first and prefer that if one exists).
Expected: no type errors. Nothing in `apps/web` imports these new hooks yet — that's the frontend plan's job.

---

## Phase F — Full verification

### Task F1: Full build, unit tests, vet

**Files:** none (verification only)

- [ ] **Step 1: Build everything**

Run (from `apps/api/`): `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Unit tests (no Docker required)**

Run: `go test ./...`
Expected: PASS across every package — `adminauth`, `booking`, `pricing`, `review`, `platform/config`, `villaslug`, etc.

- [ ] **Step 4: Integration tests (requires Docker running)**

Run: `go test -tags integration ./...`
Expected: PASS, including the new `adminauth` and `booking` integration tests. If Docker isn't available in this environment, report that explicitly rather than silently skipping — this is the only check that exercises the real migrations end-to-end.

- [ ] **Step 5: Manual smoke test (optional, requires a running Postgres + the app)**

If `mise run docker-dev` (or equivalent) is available and running:

```bash
curl -i -X POST http://localhost:8080/api/bookings \
  -H 'Content-Type: application/json' \
  -d '{"villa_slug":"casadana","guest_name":"Test","guest_email":"t@example.com","guest_phone":"+33","check_in":"2027-01-10","check_out":"2027-01-15","adults":2,"children":0}'
# expect 201, response includes "source":"direct"

curl -i http://localhost:8080/api/bookings
# expect 401 (no session cookie) — this is the security hole closing; confirm it's actually closed
```

There's no admin user seeded yet at this point (user creation requires being logged in already, which is a deliberate bootstrap gap — see note below), so a full login smoke test isn't possible until an admin row exists. Note this for whoever picks up the frontend plan: **the very first admin account must be inserted directly via SQL** (e.g. `INSERT INTO admin_users (id, email, password_hash) VALUES (uuid_generate_v4(), 'you@example.com', '<bcrypt hash>')`, generate the hash with `htpasswd -bnBC 10 "" yourpassword | tr -d ':\n' | sed 's/^\$2y/\$2a/'` or a small one-off Go script using `bcrypt.GenerateFromPassword`) since `POST /api/admin/users` itself requires an existing session. This bootstrap step isn't automated by this plan — flag it to the user as a manual one-time action before the admin UI can be used for the first time.

---

## Summary of files touched

- **Created:** `internal/adminauth/{domain,ports,service,http,postgres}.go` + their `_test.go` files (6 non-test + 5 test files, `fakes_test.go` isn't a `_test.go` pair-per-file but bundled with `service_test.go`'s task), `internal/db/migrations/0004_admin_users.{up,down}.sql`, `internal/db/migrations/0005_booking_source.{up,down}.sql`, `internal/db/queries/admin_users.sql`.
- **Modified:** `internal/platform/config/config.go` (+test), `internal/db/queries/bookings.sql`, `internal/db/{models,querier,bookings.sql}.go` (sqlc-generated), `internal/booking/{domain,ports,service,http,postgres}.go` (+their `_test.go` files), `cmd/server/casadana.go`, `internal/openapi/openapi.yaml`, `go.mod`/`go.sum`, repo-root `.env.example`/`.env.dev`, `packages/api/src/generated/**` (Orval-generated), `packages/api/src/index.ts` (if the barrel needs a manual export).
