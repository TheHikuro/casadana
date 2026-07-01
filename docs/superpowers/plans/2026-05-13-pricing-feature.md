# Pricing Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Commit policy:** DO NOT run `git commit`, `git add`, `git stash`, `git checkout`, `git reset`, or any git state-changing command. Make file changes only. The user runs commits manually. (Memories: `feedback-no-auto-commit`, `feedback-subagent-no-git-stash`.)

**Goal:** Add per-date price overrides on top of a per-villa default (95€). Display the applicable price under each available calendar cell and compute the booking sidebar total as the exact sum of per-night prices. Remove the now-unused cleaning + concierge fee lines.

**Architecture:** New hexagonal slice `internal/pricing/` (domain + ports + service + postgres adapter + http adapter). DB table `price_overrides(villa_slug, date, price_cents)` stores only overrides — defaults stay in the frontend `villas.const.ts` (`booking.nightly = 95` for both villas). A single public read endpoint `GET /api/villas/{slug}/pricing?from=&to=` returns the raw overrides for a window; the React calendar merges (override if present, else default) for display and total computation.

**Tech Stack:** Go 1.25, chi, pgx/v5, sqlc, golang-migrate (embed.FS source). React Query 5, Orval-generated client. date-fns.

**Spec:** `docs/superpowers/specs/2026-05-13-pricing-feature-design.md`.

**Scope of this plan:**
- Phase A — Backend pricing module (migration + sqlc + domain/service/adapters + wire)
- Phase B — OpenAPI spec update + Orval regen
- Phase C — Frontend cleanup: villa constants (default 95, drop cleaning/concierge) + sidebar UI cleanup + i18n keys
- Phase D — Wire `useGetVillaPricing` in `villa-booking.tsx` (cell prices + total computation)
- Phase E — Manual smoke test

**Out of scope (deferred):**
- Admin write endpoints for overrides → Plan 2 (admin auth).
- Locking prices on the booking record → payment-flow era.
- Per-villa default in DB → when admin needs to change baselines.

**Working dir for backend commands:** `apps/api/`.
**Working dir for frontend commands:** `apps/web/` or repo root (Bun workspace filters).

---

## Phase A — Backend pricing module

### Task A1: Migration 0002 — `price_overrides` table

**Files:**
- Create: `apps/api/internal/db/migrations/0002_price_overrides.up.sql`
- Create: `apps/api/internal/db/migrations/0002_price_overrides.down.sql`

- [ ] **Step 1: Write `up` migration**

`apps/api/internal/db/migrations/0002_price_overrides.up.sql`:

```sql
CREATE TABLE price_overrides (
    villa_slug  TEXT       NOT NULL,
    date        DATE       NOT NULL,
    price_cents INTEGER    NOT NULL CHECK (price_cents >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (villa_slug, date)
);

CREATE INDEX price_overrides_slug_date_idx
    ON price_overrides (villa_slug, date);
```

- [ ] **Step 2: Write `down` migration**

`apps/api/internal/db/migrations/0002_price_overrides.down.sql`:

```sql
DROP TABLE IF EXISTS price_overrides;
```

- [ ] **Step 3: Verify embed picks up the new files**

The existing `apps/api/internal/db/embed.go` uses `//go:embed migrations/*.sql` which matches both `0001_*` and `0002_*` automatically. No code change needed. Confirm by running:

```bash
cd apps/api && go build ./...
```

Expected: exit 0.

---

### Task A2: sqlc query for pricing

**Files:**
- Create: `apps/api/internal/db/queries/pricing.sql`

- [ ] **Step 1: Write the query with explicit `sqlc.arg` names**

`apps/api/internal/db/queries/pricing.sql`:

```sql
-- name: ListPriceOverrides :many
SELECT villa_slug, date, price_cents
FROM price_overrides
WHERE villa_slug = sqlc.arg('villa_slug')
  AND date >= sqlc.arg('from')
  AND date < sqlc.arg('to')
ORDER BY date;
```

Explicit `sqlc.arg('name')` gives deterministic Go field names (`VillaSlug`, `From`, `To`) instead of letting sqlc pick.

- [ ] **Step 2: Regenerate sqlc code**

From `apps/api/`:

```bash
sqlc generate
```

Expected: a new file `internal/db/pricing.sql.go` appears containing:
- `type ListPriceOverridesParams struct { VillaSlug string; From pgtype.Date; To pgtype.Date }`
- `type ListPriceOverridesRow struct { VillaSlug string; Date pgtype.Date; PriceCents int32 }`
- `func (q *Queries) ListPriceOverrides(ctx, ListPriceOverridesParams) ([]ListPriceOverridesRow, error)`

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: exit 0.

---

### Task A3: Domain entity + sentinel errors

**Files:**
- Create: `apps/api/internal/pricing/domain.go`

- [ ] **Step 1: Implement domain**

`apps/api/internal/pricing/domain.go`:

```go
package pricing

import (
	"errors"
	"time"
)

// PriceOverride represents a per-date price override for a villa.
// Price is stored in integer cents to avoid float pitfalls.
type PriceOverride struct {
	VillaSlug  string
	Date       time.Time
	PriceCents int
}

var (
	ErrUnknownVilla = errors.New("unknown villa")
	ErrInvalidRange = errors.New("from must be before to")
)
```

- [ ] **Step 2: Verify build**

```bash
go build ./internal/pricing/...
```

Expected: exit 0.

---

### Task A4: Ports

**Files:**
- Create: `apps/api/internal/pricing/ports.go`

- [ ] **Step 1: Define interfaces**

`apps/api/internal/pricing/ports.go`:

```go
package pricing

import (
	"context"
	"time"
)

type Repository interface {
	ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error)
}

type VillaAllowlist interface {
	IsKnown(slug string) bool
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./internal/pricing/...
```

Expected: exit 0.

---

### Task A5: Test fakes

**Files:**
- Create: `apps/api/internal/pricing/fakes_test.go`

- [ ] **Step 1: Implement fakes for service tests**

`apps/api/internal/pricing/fakes_test.go`:

```go
package pricing

import (
	"context"
	"time"
)

type fakeRepo struct {
	overrides []PriceOverride
	listErr   error
}

func (f *fakeRepo) ListOverrides(_ context.Context, _ string, _, _ time.Time) ([]PriceOverride, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.overrides, nil
}

type fakeAllowlist struct {
	allowed map[string]bool
}

func (f fakeAllowlist) IsKnown(slug string) bool { return f.allowed[slug] }

// d parses YYYY-MM-DD into a time.Time. Test helper.
func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
```

- [ ] **Step 2: Verify fakes compile under test binary**

```bash
go test -run NONE ./internal/pricing/...
```

Expected: `PASS` (no tests to run, just builds the test binary).

---

### Task A6: Service (TDD)

**Files:**
- Create: `apps/api/internal/pricing/service.go`
- Create: `apps/api/internal/pricing/service_test.go`

- [ ] **Step 1: Write the failing tests**

`apps/api/internal/pricing/service_test.go`:

```go
package pricing

import (
	"context"
	"testing"
)

func newSvc(repo Repository, allow VillaAllowlist) *Service {
	return NewService(repo, allow)
}

func TestListOverrides_Happy(t *testing.T) {
	repo := &fakeRepo{
		overrides: []PriceOverride{
			{VillaSlug: "casadana", Date: d("2026-07-04"), PriceCents: 25000},
			{VillaSlug: "casadana", Date: d("2026-07-05"), PriceCents: 25000},
		},
	}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true}}
	svc := newSvc(repo, allow)

	out, err := svc.ListOverrides(context.Background(), "casadana", d("2026-07-01"), d("2026-08-01"))
	if err != nil {
		t.Fatalf("ListOverrides: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("len = %d, want 2", len(out))
	}
}

func TestListOverrides_UnknownVilla(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{}})

	_, err := svc.ListOverrides(context.Background(), "ghost", d("2026-07-01"), d("2026-08-01"))
	if err != ErrUnknownVilla {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
}

func TestListOverrides_InvalidRange(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})

	cases := []struct {
		name     string
		from, to string
	}{
		{"to before from", "2026-08-01", "2026-07-01"},
		{"to equals from", "2026-07-01", "2026-07-01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.ListOverrides(context.Background(), "casadana", d(c.from), d(c.to))
			if err != ErrInvalidRange {
				t.Fatalf("err = %v, want ErrInvalidRange", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests, confirm compile failure**

```bash
go test ./internal/pricing/...
```

Expected: compile error — `Service`, `NewService`, `ListOverrides` undefined.

- [ ] **Step 3: Implement the service**

`apps/api/internal/pricing/service.go`:

```go
package pricing

import (
	"context"
	"time"
)

type Service struct {
	repo  Repository
	allow VillaAllowlist
}

func NewService(repo Repository, allow VillaAllowlist) *Service {
	return &Service{repo: repo, allow: allow}
}

func (s *Service) ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error) {
	if !s.allow.IsKnown(villaSlug) {
		return nil, ErrUnknownVilla
	}
	if !to.After(from) {
		return nil, ErrInvalidRange
	}
	return s.repo.ListOverrides(ctx, villaSlug, from, to)
}
```

- [ ] **Step 4: Run tests, confirm PASS**

```bash
go test ./internal/pricing/...
```

Expected: `PASS` (4 tests, including 2 subtests).

---

### Task A7: Postgres adapter + integration test

**Files:**
- Create: `apps/api/internal/pricing/postgres.go`
- Create: `apps/api/internal/pricing/postgres_test.go`

- [ ] **Step 1: Implement the adapter**

`apps/api/internal/pricing/postgres.go`:

```go
package pricing

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheHikuro/casadana/internal/db"
)

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) q() *db.Queries { return db.New(r.pool) }

func (r *pgRepo) ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error) {
	rows, err := r.q().ListPriceOverrides(ctx, db.ListPriceOverridesParams{
		VillaSlug: villaSlug,
		From:      pgtype.Date{Time: from, Valid: true},
		To:        pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]PriceOverride, 0, len(rows))
	for _, row := range rows {
		out = append(out, PriceOverride{
			VillaSlug:  row.VillaSlug,
			Date:       row.Date.Time,
			PriceCents: int(row.PriceCents),
		})
	}
	return out, nil
}
```

If the sqlc-generated `ListPriceOverridesParams` field for `from`/`to` is named differently (e.g. `Date` and `Date_2` if `sqlc.arg` wasn't honored), inspect `internal/db/pricing.sql.go` and adapt. The plan assumes `From` and `To` because of the explicit `sqlc.arg('from')` / `sqlc.arg('to')` in the query.

- [ ] **Step 2: Write the integration test**

`apps/api/internal/pricing/postgres_test.go`:

```go
//go:build integration

package pricing

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pg "github.com/TheHikuro/casadana/internal/platform/postgres"
	dbpkg "github.com/TheHikuro/casadana/internal/db"
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

	abs, _ := filepath.Abs("../db/migrations")
	_ = abs // not used in iofs path; embed is the source
	if err := pg.MigrateUp(pool, dbpkg.Migrations, "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func TestPgRepo_ListOverrides(t *testing.T) {
	pool := setupPg(t)
	ctx := context.Background()

	// Seed two overrides
	_, err := pool.Exec(ctx,
		`INSERT INTO price_overrides (villa_slug, date, price_cents) VALUES
		 ('casadana', '2026-07-04', 25000),
		 ('casadana', '2026-07-05', 25000),
		 ('casacasay', '2026-07-04', 18000)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewPgRepo(pool)
	out, err := repo.ListOverrides(ctx, "casadana", d("2026-07-01"), d("2026-08-01"))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("len = %d, want 2", len(out))
	}
	if out[0].PriceCents != 25000 {
		t.Errorf("price_cents = %d, want 25000", out[0].PriceCents)
	}
}
```

- [ ] **Step 3: Run unit tests (no integration)**

```bash
go test ./internal/pricing/...
```

Expected: PASS (integration test skipped by build tag).

- [ ] **Step 4: Run integration test if Docker available**

```bash
go test -tags integration ./internal/pricing/...
```

Expected: PASS. If Docker isn't running, skip and note it.

---

### Task A8: HTTP handler

**Files:**
- Create: `apps/api/internal/pricing/http.go`
- Create: `apps/api/internal/pricing/http_test.go`

- [ ] **Step 1: Implement the handler**

`apps/api/internal/pricing/http.go`:

```go
package pricing

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
)

func init() {
	httpserver.Register(ErrUnknownVilla, http.StatusNotFound, "UNKNOWN_VILLA")
	httpserver.Register(ErrInvalidRange, http.StatusUnprocessableEntity, "INVALID_RANGE")
}

func Mount(r chi.Router, svc *Service) {
	r.Get("/api/villas/{slug}/pricing", listHandler(svc))
}

type priceOverrideDTO struct {
	Date       string `json:"date"`
	PriceCents int    `json:"price_cents"`
}

type pricingResponse struct {
	Overrides []priceOverrideDTO `json:"overrides"`
}

func listHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		from, errFrom := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, errTo := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if errFrom != nil || errTo != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{
				Message: "from and to must be YYYY-MM-DD",
			})
			return
		}

		overrides, err := svc.ListOverrides(r.Context(), slug, from, to)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}

		resp := pricingResponse{Overrides: make([]priceOverrideDTO, 0, len(overrides))}
		for _, o := range overrides {
			resp.Overrides = append(resp.Overrides, priceOverrideDTO{
				Date:       o.Date.Format("2006-01-02"),
				PriceCents: o.PriceCents,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
```

- [ ] **Step 2: Write the http test**

`apps/api/internal/pricing/http_test.go`:

```go
package pricing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc)
	return r
}

func TestGetPricing_Empty(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/villas/casadana/pricing?from=2026-07-01&to=2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out pricingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Overrides) != 0 {
		t.Errorf("overrides = %v, want []", out.Overrides)
	}
}

func TestGetPricing_WithOverrides(t *testing.T) {
	repo := &fakeRepo{overrides: []PriceOverride{
		{VillaSlug: "casadana", Date: d("2026-07-04"), PriceCents: 25000},
	}}
	svc := newSvc(repo, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/villas/casadana/pricing?from=2026-07-01&to=2026-08-01")
	defer resp.Body.Close()

	var out pricingResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Overrides) != 1 || out.Overrides[0].Date != "2026-07-04" || out.Overrides[0].PriceCents != 25000 {
		t.Errorf("unexpected overrides: %+v", out.Overrides)
	}
}

func TestGetPricing_UnknownVilla(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/villas/ghost/pricing?from=2026-07-01&to=2026-08-01")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetPricing_BadDates(t *testing.T) {
	svc := newSvc(&fakeRepo{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}})
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/villas/casadana/pricing?from=oops&to=2026-08-01")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}
```

- [ ] **Step 3: Run all pricing tests**

```bash
go test ./internal/pricing/...
```

Expected: PASS.

---

### Task A9: Wire pricing into `cmd/server`

**Files:**
- Modify: `apps/api/cmd/server/casadana.go`

- [ ] **Step 1: Add import**

In `apps/api/cmd/server/casadana.go`, the imports already include the pattern for booking. Add `pricing` to the imports:

```go
import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/TheHikuro/casadana/internal/booking"
	"github.com/TheHikuro/casadana/internal/db"
	"github.com/TheHikuro/casadana/internal/openapi"
	"github.com/TheHikuro/casadana/internal/platform/config"
	"github.com/TheHikuro/casadana/internal/platform/email"
	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/logger"
	"github.com/TheHikuro/casadana/internal/platform/postgres"
	"github.com/TheHikuro/casadana/internal/pricing"
	"github.com/TheHikuro/casadana/internal/villaslug"
)
```

- [ ] **Step 2: Wire the service and mount**

Add the pricing service construction next to the booking service (after `bookingSvc := booking.NewService(...)`):

```go
	pricingSvc := pricing.NewService(pricing.NewPgRepo(pool), slugAllowlist{})
```

And mount it next to `booking.Mount(r, bookingSvc)`:

```go
	r := httpserver.NewRouter(log, cfg.WebOrigin)
	openapi.Mount(r)
	booking.Mount(r, bookingSvc)
	pricing.Mount(r, pricingSvc)
```

- [ ] **Step 3: Verify full build + tests**

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: build clean, vet clean, all tests PASS (including the new pricing tests).

---

### Task A10: Dev seed file

**Files:**
- Create: `apps/api/internal/db/seed_dev.sql`

- [ ] **Step 1: Write the seed file**

`apps/api/internal/db/seed_dev.sql`:

```sql
-- Dev-only example overrides. NOT a migration. Run manually with:
--   docker exec -i casadana-postgres psql -U casadana -d casadana < apps/api/internal/db/seed_dev.sql

INSERT INTO price_overrides (villa_slug, date, price_cents) VALUES
  ('casadana', '2026-07-04', 25000),
  ('casadana', '2026-07-05', 25000),
  ('casadana', '2026-07-11', 25000),
  ('casacasay', '2026-08-15', 18000),
  ('casacasay', '2026-08-16', 18000)
ON CONFLICT (villa_slug, date) DO UPDATE
SET price_cents = EXCLUDED.price_cents,
    updated_at = NOW();
```

This file is NOT embedded in the binary (the `//go:embed migrations/*.sql` in `internal/db/embed.go` only matches the migrations subdirectory). It's never auto-applied. Manual `psql` only.

---

## Phase B — OpenAPI spec update + Orval regen

### Task B1: Extend `openapi.yaml` with the pricing path + schemas

**Files:**
- Modify: `apps/api/internal/openapi/openapi.yaml`

- [ ] **Step 1: Add the `pricing` tag and the new path**

In `apps/api/internal/openapi/openapi.yaml`, find the `tags:` block:

```yaml
tags:
  - name: health
    description: Liveness
  - name: bookings
    description: Booking submission and availability
```

Append a new tag:

```yaml
  - name: pricing
    description: Per-date price overrides
```

Then in the `paths:` section, add the new path right after `/api/villas/{slug}/availability`:

```yaml
  /api/villas/{slug}/pricing:
    get:
      operationId: getVillaPricing
      tags: [pricing]
      summary: Get price overrides for a villa within a date window
      description: |
        Returns the sparse list of per-date price overrides for a villa within
        the [from, to) window. Dates without an override fall back to the villa's
        frontend-defined default (`booking.nightly` in villas.const.ts).
      parameters:
        - name: slug
          in: path
          required: true
          schema: { type: string }
          example: casadana
        - name: from
          in: query
          required: true
          schema: { type: string, format: date }
          description: Window start (inclusive). YYYY-MM-DD.
          example: "2026-07-01"
        - name: to
          in: query
          required: true
          schema: { type: string, format: date }
          description: Window end (exclusive). YYYY-MM-DD.
          example: "2026-08-01"
      responses:
        "200":
          description: Price overrides
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PricingResponse"
        "404":
          description: Villa slug not in allowlist
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "422":
          description: Invalid query params or invalid range
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "500":
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
```

- [ ] **Step 2: Add the new schemas**

In the same file, inside `components.schemas`, add (right after `AvailabilityResponse`):

```yaml
    PriceOverride:
      type: object
      required: [date, price_cents]
      properties:
        date: { type: string, format: date }
        price_cents:
          type: integer
          minimum: 0
          description: Price in EUR cents (e.g. 9500 = 95.00€).
          example: 9500

    PricingResponse:
      type: object
      required: [overrides]
      properties:
        overrides:
          type: array
          items: { $ref: "#/components/schemas/PriceOverride" }
```

- [ ] **Step 3: Verify the YAML is parseable**

From repo root:

```bash
bunx -y @stoplight/spectral-cli lint apps/api/internal/openapi/openapi.yaml
```

Expected: 0 errors (warnings about missing descriptions are OK).

---

### Task B2: Regenerate Orval client

**Files:**
- Modify (regenerated): `packages/api/src/generated/**`

- [ ] **Step 1: Regenerate**

From `packages/api/`:

```bash
bun run generate
```

Expected: a new tag directory `src/generated/pricing/` containing `pricing.ts` with `useGetVillaPricing` hook.

- [ ] **Step 2: Verify hook name**

```bash
grep -E "^export (function|const) useGetVillaPricing" packages/api/src/generated/pricing/pricing.ts
```

Expected: a match. If the name differs (e.g. `useGetVillaPricingByFromAndTo`), note the actual name; later tasks use whatever Orval produced.

- [ ] **Step 3: Update the barrel if needed**

The barrel at `packages/api/src/index.ts` re-exports tag files explicitly:

```ts
export * from "./generated/bookings/bookings"
export * from "./generated/health/health"
export * from "./generated/schemas"
export { customAxios, AXIOS_INSTANCE, ApiError } from "./client"
export type { ApiErrorBody } from "./client"
```

Add the pricing tag re-export so the new hook is exposed through `@casa-dana/api`:

```ts
export * from "./generated/bookings/bookings"
export * from "./generated/health/health"
export * from "./generated/pricing/pricing"
export * from "./generated/schemas"
export { customAxios, AXIOS_INSTANCE, ApiError } from "./client"
export type { ApiErrorBody } from "./client"
```

- [ ] **Step 4: Type-check the package**

```bash
cd packages/api && bunx tsc --noEmit
```

Expected: exit 0.

---

## Phase C — Frontend cleanup: defaults + sidebar fees

### Task C1: Drop cleaning/concierge and set nightly=95 in `villas.const.ts`

**Files:**
- Modify: `apps/web/src/constants/villas.const.ts`

- [ ] **Step 1: Update the type**

Find the `booking:` type definition (around lines 113-122 of `villas.const.ts`). The current shape includes:

```ts
booking: {
  nightly: number
  cleaning: number
  concierge: number
  rating: number
  reviewCount: number
  maxGuests: number
  defaultGuests: number
  defaultCheckIn: string
  defaultCheckOut: string
}
```

Remove `cleaning: number` and `concierge: number` lines so it becomes:

```ts
booking: {
  nightly: number
  rating: number
  reviewCount: number
  maxGuests: number
  defaultGuests: number
  defaultCheckIn: string
  defaultCheckOut: string
}
```

- [ ] **Step 2: Update both villa entries**

For **casadana** (around line 242), the current lines are:

```ts
      nightly: 185,
      cleaning: 80,
      concierge: 45,
```

Change to:

```ts
      nightly: 95,
```

(Remove the two fee lines entirely.)

For **casacasay** (around line 549), the current lines are:

```ts
      nightly: 145,
      cleaning: 60,
      concierge: 35,
```

Change to:

```ts
      nightly: 95,
```

- [ ] **Step 3: Verify tsc**

```bash
cd apps/web && bunx tsc --noEmit 2>&1 | grep -E "villas\.const\.ts" || echo "no errors in villas.const.ts"
```

Expected: "no errors in villas.const.ts".

---

### Task C2: Remove cleaning/concierge UI from `villa-booking.tsx` sidebar

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Locate the pricing sidebar block**

In `villa-booking.tsx`, find the block (currently near the bottom):

```tsx
      <div className="border-outline-variant mt-6 grid gap-3 border-t pt-5 text-[13.5px]">
        <div className="text-on-surface-variant flex justify-between">
          <span>
            {nights} {nights === 1 ? m.villa_booking_night_singular() : m.villa_booking_night_plural()} × €{booking.nightly}
          </span>
          <span>€{subtotal.toLocaleString()}</span>
        </div>
        <div className="text-on-surface-variant flex justify-between">
          <span>{m.villa_booking_cleaning_fee()}</span>
          <span>€{booking.cleaning}</span>
        </div>
        <div className="text-on-surface-variant flex justify-between">
          <span>{m.villa_booking_concierge_welcome()}</span>
          <span>€{booking.concierge}</span>
        </div>
        <div className="font-display text-primary border-outline-variant mt-1 flex justify-between border-t pt-3.5 text-[22px] italic">
          <span>{m.villa_booking_total()}</span>
          <span>€{total.toLocaleString()}</span>
        </div>
      </div>
```

Replace with (Phase D will further update the price-summary line; for now keep `nights × booking.nightly` which still works since nightly is now 95):

```tsx
      <div className="border-outline-variant mt-6 grid gap-3 border-t pt-5 text-[13.5px]">
        <div className="text-on-surface-variant flex justify-between">
          <span>
            {nights} {nights === 1 ? m.villa_booking_night_singular() : m.villa_booking_night_plural()}
          </span>
          <span>€{subtotal.toLocaleString()}</span>
        </div>
        <div className="font-display text-primary border-outline-variant mt-1 flex justify-between border-t pt-3.5 text-[22px] italic">
          <span>{m.villa_booking_total()}</span>
          <span>€{total.toLocaleString()}</span>
        </div>
      </div>
```

- [ ] **Step 2: Update the totals computation**

Find the `subtotal` / `total` computation (currently near line 84-86 of the post-Phase-E version):

```tsx
  const nights = nightsBetween(checkIn, checkOut)
  const subtotal = nights * booking.nightly
  const total = subtotal + booking.cleaning + booking.concierge
```

Replace with:

```tsx
  const nights = nightsBetween(checkIn, checkOut)
  const subtotal = nights * booking.nightly
  const total = subtotal
```

(Phase D replaces these with the per-night sum. This intermediate form just keeps the file compilable.)

- [ ] **Step 3: Verify tsc**

```bash
cd apps/web && bunx tsc --noEmit 2>&1 | grep -E "villa-booking\.tsx" || echo "no errors in villa-booking.tsx"
```

Expected: "no errors in villa-booking.tsx".

---

### Task C3: Drop cleaning/concierge i18n keys

**Files:**
- Modify: `apps/web/messages/en.json`
- Modify: `apps/web/messages/es.json`
- Modify: `apps/web/messages/fr.json`
- Modify (regenerated): `apps/web/src/paraglide/**`

- [ ] **Step 1: Remove the two keys from each locale file**

Delete these keys (and their values) from each of `apps/web/messages/{en,es,fr}.json`:

- `villa_booking_cleaning_fee`
- `villa_booking_concierge_welcome`

Make sure the JSON remains valid (no trailing comma issues).

- [ ] **Step 2: Regenerate paraglide**

```bash
cd apps/web && bun run paraglide
```

Expected: regenerates `apps/web/src/paraglide/messages.ts` without the deleted accessors.

- [ ] **Step 3: Verify build**

```bash
cd apps/web && bunx vite build 2>&1 | tail -3
```

Expected: `✓ built in <time>`.

---

## Phase D — Wire `useGetVillaPricing` in the calendar

### Task D1: Add the pricing query alongside availability

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Add the import**

At the top of `villa-booking.tsx`, in the `@casa-dana/api` import line, append `useGetVillaPricing`:

```tsx
import { ApiError, useCreateBooking, useGetVillaAvailability, useGetVillaPricing } from "@casa-dana/api"
```

- [ ] **Step 2: Add the query call**

Just below the `useGetVillaAvailability` call inside the component, add:

```tsx
const { data: pricing } = useGetVillaPricing(
  villaSlug,
  { from: queryWindow.from, to: queryWindow.to },
  {
    query: {
      enabled: activeField !== null,
      placeholderData: keepPreviousData,
    },
  },
)
```

(`queryWindow`, `activeField`, `villaSlug`, `keepPreviousData` are all already in scope from previous phases.)

- [ ] **Step 3: Verify tsc**

```bash
cd apps/web && bunx tsc --noEmit 2>&1 | grep -E "villa-booking\.tsx" || echo "no errors in villa-booking.tsx"
```

Expected: "no errors in villa-booking.tsx".

---

### Task D2: Build the override map + `priceCentsFor` helper

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Add the memoized map and helper**

Just below the existing `blockedNights` memo and `isBlocked` helper, add:

```tsx
const priceOverridesByDate = useMemo(() => {
  const map = new Map<string, number>()
  for (const o of pricing?.overrides ?? []) {
    map.set(o.date, o.price_cents)
  }
  return map
}, [pricing])

const priceCentsFor = (date: Date): number => {
  const key = format(date, "yyyy-MM-dd")
  const override = priceOverridesByDate.get(key)
  return override ?? booking.nightly * 100
}
```

The helper returns cents. The default fallback `booking.nightly * 100` converts the frontend default (95) to cents.

- [ ] **Step 2: Verify tsc**

```bash
cd apps/web && bunx tsc --noEmit 2>&1 | grep -E "villa-booking\.tsx" || echo "no errors in villa-booking.tsx"
```

Expected: "no errors".

---

### Task D3: Render price under each available cell

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Find the cell render block**

In `villa-booking.tsx`, find the cell render block (post Phase F). The current "available cell" branch looks like:

```tsx
const isCI = sameDay(cell.date, checkIn)
const isCO = sameDay(cell.date, checkOut)
const inRange = cell.date > checkIn && cell.date < checkOut
return (
  <button
    key={i}
    type="button"
    onClick={() => pickDate(cell.date as Date)}
    className={cn(
      "text-on-surface flex aspect-square items-center justify-center text-[13px] transition-colors",
      inRange && "bg-secondary-container text-on-secondary-container",
      (isCI || isCO) && "bg-primary text-on-primary rounded-full",
      !inRange && !isCI && !isCO && "hover:bg-surface-container-low",
    )}
  >
    {cell.d}
  </button>
)
```

- [ ] **Step 2: Replace with a vertical flex layout including the price**

Selected (CI/CO) cells stay round and omit the price (no room). All other available cells show the price line.

Replace the available-cell branch with:

```tsx
const isCI = sameDay(cell.date, checkIn)
const isCO = sameDay(cell.date, checkOut)
const inRange = cell.date > checkIn && cell.date < checkOut
const showPrice = !(isCI || isCO)
const priceCents = priceCentsFor(cell.date)
return (
  <button
    key={i}
    type="button"
    onClick={() => pickDate(cell.date as Date)}
    className={cn(
      "text-on-surface flex min-h-[42px] flex-col items-center justify-center gap-0.5 text-[13px] transition-colors",
      inRange && "bg-secondary-container text-on-secondary-container",
      (isCI || isCO) && "bg-primary text-on-primary aspect-square rounded-full",
      !inRange && !isCI && !isCO && "hover:bg-surface-container-low",
    )}
  >
    <span>{cell.d}</span>
    {showPrice && (
      <span className="text-on-surface-variant font-mono text-[9px] leading-none">
        €{Math.round(priceCents / 100)}
      </span>
    )}
  </button>
)
```

Key changes:
- `aspect-square` removed from the base class (it's only kept on the round selected variant)
- New `flex-col` + `min-h-[42px]` to accommodate two lines
- `<span>{cell.d}</span>` wraps the day number
- Conditional `<span>` shows `€{price}` only when not the selected boundary

Also update the muted/blocked branches to use the same min-h for visual alignment. Find:

```tsx
if (cell.muted || !cell.date) {
  return (
    <span
      key={i}
      className="text-outline-variant flex aspect-square cursor-default items-center justify-center text-[13px]"
    >
      {cell.d}
    </span>
  )
}
```

Replace with:

```tsx
if (cell.muted || !cell.date) {
  return (
    <span
      key={i}
      className="text-outline-variant flex min-h-[42px] cursor-default items-center justify-center text-[13px]"
    >
      {cell.d}
    </span>
  )
}
```

And the blocked branch:

```tsx
if (blocked) {
  return (
    <span
      key={i}
      aria-disabled="true"
      className="text-on-surface-variant/40 flex aspect-square cursor-not-allowed items-center justify-center text-[13px] line-through"
    >
      {cell.d}
    </span>
  )
}
```

Replace with:

```tsx
if (blocked) {
  return (
    <span
      key={i}
      aria-disabled="true"
      className="text-on-surface-variant/40 flex min-h-[42px] cursor-not-allowed items-center justify-center text-[13px] line-through"
    >
      {cell.d}
    </span>
  )
}
```

- [ ] **Step 3: Verify tsc + vite build**

```bash
cd apps/web && bunx tsc --noEmit 2>&1 | grep -E "villa-booking\.tsx" || echo "no errors in villa-booking.tsx"
cd apps/web && bunx vite build 2>&1 | tail -3
```

Expected: no errors; `✓ built in <time>`.

---

### Task D4: Recompute the sidebar total from per-night prices

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Replace `subtotal` and `total` with a per-night sum**

Find the previously-edited block from Task C2:

```tsx
const nights = nightsBetween(checkIn, checkOut)
const subtotal = nights * booking.nightly
const total = subtotal
```

Replace with:

```tsx
const nights = nightsBetween(checkIn, checkOut)
const totalCents = useMemo(() => {
  let sum = 0
  for (let d = new Date(checkIn); d < checkOut; d = addDays(d, 1)) {
    sum += priceCentsFor(d)
  }
  return sum
}, [checkIn, checkOut, priceOverridesByDate, booking.nightly])
const total = totalCents / 100
const subtotal = total
```

(`addDays` is already imported from `date-fns` since Phase F. `priceCentsFor` was added in D2. `priceOverridesByDate` and `booking.nightly` are the underlying inputs; React will recompute when they change.)

- [ ] **Step 2: Update the sidebar rendering**

In the sidebar block edited in Task C2 (the `{nights} nights` row), the layout already reflects the simplified shape. Verify it currently reads:

```tsx
<div className="text-on-surface-variant flex justify-between">
  <span>
    {nights} {nights === 1 ? m.villa_booking_night_singular() : m.villa_booking_night_plural()}
  </span>
  <span>€{subtotal.toLocaleString()}</span>
</div>
```

Leave this as-is. `subtotal` now reflects the precise sum.

The total row already uses `€{total.toLocaleString()}` and shows the same precise value.

- [ ] **Step 3: Final verification**

```bash
cd apps/web && bunx tsc --noEmit 2>&1 | grep -E "villa-booking\.tsx" || echo "no errors in villa-booking.tsx"
cd apps/web && bunx vite build 2>&1 | tail -3
```

Expected: no errors in `villa-booking.tsx`; vite build succeeds.

---

## Phase E — Manual smoke test

### Task E1: Rebuild + verify the pricing flow end-to-end

**Files:** none.

- [ ] **Step 1: Rebuild and restart the API**

From repo root:

```bash
docker compose -f .docker/docker-compose.yml --env-file .env.dev --profile dev up -d --build api
```

Wait for the container to be healthy:

```bash
docker compose -f .docker/docker-compose.yml ps
```

The new migration `0002_price_overrides` should be applied at boot.

- [ ] **Step 2: Verify the new endpoint is reachable**

```bash
curl -s http://localhost:8080/api/villas/casadana/pricing?from=2026-07-01&to=2026-08-01
```

Expected: `{"overrides":[]}` (no seed yet).

- [ ] **Step 3: Apply the dev seed**

```bash
docker exec -i casadana-postgres psql -U casadana -d casadana < apps/api/internal/db/seed_dev.sql
```

Expected: `INSERT 0 5` or similar.

- [ ] **Step 4: Verify the seed is reflected**

```bash
curl -s 'http://localhost:8080/api/villas/casadana/pricing?from=2026-07-01&to=2026-08-01'
```

Expected:

```json
{"overrides":[
  {"date":"2026-07-04","price_cents":25000},
  {"date":"2026-07-05","price_cents":25000},
  {"date":"2026-07-11","price_cents":25000}
]}
```

- [ ] **Step 5: Verify the Swagger UI shows the new endpoint**

Open `http://localhost:8080/api/docs` in a browser. The "pricing" tag should appear with `GET /api/villas/{slug}/pricing`. The "Try it out" should return live data.

- [ ] **Step 6: Verify the frontend calendar**

Start the web dev server (in a separate terminal):

```bash
bun --filter @casa-dana/web dev
```

Open `http://localhost:5173/villa/casadana`. Click the check-in field to open the calendar.

Expected:
- Each available cell shows the day number AND `€95` below (for non-override dates).
- Cells for July 4, 5, 11, 2026 show `€250`.
- Pick a range that spans some overrides (e.g. July 1 to July 12). The sidebar "Total" reflects the precise sum: 6 nights at 95 + 3 nights at 250 = 570 + 750 = **€1320**. Verify the displayed total matches.
- Cells you don't have an override on still show €95.
- Blocked cells (if any approved booking exists) show no price.
- The cleaning and concierge fee lines are gone from the sidebar.

- [ ] **Step 7: Stop the stack when done**

```bash
docker compose -f .docker/docker-compose.yml --env-file .env.dev --profile dev down
```

---

## Post-flight

- [ ] **Files touched (sanity list)**

Created:
- `apps/api/internal/pricing/{domain.go, ports.go, fakes_test.go, service.go, service_test.go, postgres.go, postgres_test.go, http.go, http_test.go}`
- `apps/api/internal/db/migrations/0002_price_overrides.{up,down}.sql`
- `apps/api/internal/db/queries/pricing.sql`
- `apps/api/internal/db/seed_dev.sql`
- `apps/api/internal/db/pricing.sql.go` (sqlc-generated)
- `packages/api/src/generated/pricing/pricing.ts` (Orval-generated)

Modified:
- `apps/api/cmd/server/casadana.go` (wire pricing module)
- `apps/api/internal/openapi/openapi.yaml` (new path + schemas + tag)
- `packages/api/src/index.ts` (barrel adds pricing tag re-export)
- `apps/web/src/constants/villas.const.ts` (type + both villa entries: drop cleaning/concierge, nightly=95)
- `apps/web/src/components/sections/villa/villa-booking.tsx` (new query + cell layout + total)
- `apps/web/messages/{en,es,fr}.json` (drop two keys)
- `apps/web/src/paraglide/**` (regenerated)

- [ ] **Things deferred to later plans**

- Admin write endpoints (`POST /api/admin/villas/{slug}/prices`, `DELETE /api/admin/villas/{slug}/prices/{date}`) — Plan 2 (admin auth).
- Locking the price on the booking record at submission time — payment-flow plan.
- Per-villa baseline in DB — when admin needs to change it without a frontend deploy.
- Weekend / season multipliers, min-stay rules — when product needs them.
