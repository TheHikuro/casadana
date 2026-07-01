# Casa Dana — Pricing Feature Design

**Date:** 2026-05-13
**Status:** Approved — ready for implementation plan

## 1. Purpose & Scope

Introduce per-date price overrides on top of a per-villa default price (currently 95€ for both villas). The booking calendar shows the applicable price under each available date, and the booking sidebar sums the exact total for the selected range.

**In scope:**

- New backend module `internal/pricing/` (hexagonal slice: domain → ports → service → postgres + http adapters).
- DB migration adding a sparse `price_overrides` table keyed by `(villa_slug, date)` with `price_cents`.
- Public endpoint `GET /api/villas/{slug}/pricing?from=&to=` returning the raw override rows for the window.
- OpenAPI spec extended with the new operation + schemas; regenerate the Orval client.
- Frontend: change both villas' `nightly` to `95` in `villas.const.ts`; remove `cleaning` and `concierge` from the villa data model and from the booking sidebar; wire `useGetVillaPricing` into `villa-booking.tsx`; render `€X` under each available calendar cell; recompute the sidebar total by summing per-date prices.
- A dev-only seed file `apps/api/internal/db/seed_dev.sql` containing example overrides (executable manually with `psql`).

**Out of scope:**

- Admin endpoints to create/delete overrides — deferred to Plan 2 (admin auth).
- Locking a price on a booking record (no payment flow yet).
- Per-villa default prices in DB (defaults remain in `villas.const.ts`).
- Variable pricing rules (weekend multipliers, guest-count surcharges, promo codes).
- Currencies other than EUR.
- Any automatic seeding of bookings or price overrides in production.

## 2. Architecture Decisions

### 2.1 Sparse override storage, default in frontend

Per-villa default (`95` for both villas) lives in `apps/web/src/constants/villas.const.ts` under `booking.nightly`. The backend stores **only** date-specific overrides. The API returns raw overrides for a queried window; the frontend merges (override if present, else `villa.booking.nightly`).

**Why this shape?**

- Aligns with project memory `project-villas-static`: villa content (now including the baseline price) stays static in the frontend.
- Backend stays minimal — one table, one read endpoint, no knowledge of defaults.
- Single source of truth for the default: frontend constant. No duplication and drift risk.

**Trade-off:** Frontend has to do the merge. Acceptable — it's a few lines.

### 2.2 Money in integer cents

`price_cents INTEGER NOT NULL CHECK (price_cents >= 0)`. Avoids float pitfalls. Frontend divides by 100 for display. All API I/O uses `price_cents` (integer).

### 2.3 Pricing is informational, not stored on bookings

Bookings stay date-range + guest contact only. The total displayed in the sidebar is computed for display. When a payment flow exists, prices will be locked on the booking at that moment — out of scope here.

### 2.4 No automatic seeding in prod

- Migrations contain DDL only (CREATE TABLE, indexes). No `INSERT`.
- Dev seed lives in a non-migration file `internal/db/seed_dev.sql`, executed manually via `psql` when wanted. Not embedded in the binary, not run on boot.
- The existing `ADMIN_BOOTSTRAP_*` mechanism (Plan 1 spec §13) remains opt-in via env vars; that's the only acceptable production "seed" and only fires when explicitly enabled.

### 2.5 Remove cleaning + concierge fees

The product no longer charges cleaning or concierge fees. These fields are removed from:
- `VillaData["booking"]` type definition
- Both villa entries in `villas.const.ts`
- The booking sidebar UI in `villa-booking.tsx`
- The trilingual i18n keys (`villa_booking_cleaning_fee`, `villa_booking_concierge_welcome`)

## 3. Data Model

### 3.1 Migration `0002_price_overrides.up.sql`

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

### 3.2 Migration `0002_price_overrides.down.sql`

```sql
DROP TABLE IF EXISTS price_overrides;
```

### 3.3 sqlc query `internal/db/queries/pricing.sql`

```sql
-- name: ListPriceOverrides :many
SELECT villa_slug, date, price_cents
FROM price_overrides
WHERE villa_slug = $1
  AND date >= $2
  AND date < $3
ORDER BY date;
```

### 3.4 Dev seed `internal/db/seed_dev.sql`

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

## 4. Backend Module

### 4.1 Layout

```
apps/api/internal/pricing/
├── domain.go         # PriceOverride entity, ErrUnknownVilla, ErrInvalidRange
├── ports.go          # Repository, VillaAllowlist
├── service.go        # *Service.ListOverrides(ctx, slug, from, to)
├── http.go           # Mount + handler + DTO + sentinel error registration
├── postgres.go       # NewPgRepo(pool) Repository
├── service_test.go   # fakes, unit tests
└── postgres_test.go  # //go:build integration — testcontainers
```

### 4.2 Domain

```go
package pricing

import (
    "errors"
    "time"
)

type PriceOverride struct {
    VillaSlug  string
    Date       time.Time   // truncated to date
    PriceCents int
}

var (
    ErrUnknownVilla  = errors.New("unknown villa")
    ErrInvalidRange  = errors.New("from must be before to")
)
```

### 4.3 Ports

```go
type Repository interface {
    ListOverrides(ctx context.Context, villaSlug string, from, to time.Time) ([]PriceOverride, error)
}

type VillaAllowlist interface {
    IsKnown(slug string) bool
}
```

### 4.4 Service

```go
type Service struct {
    repo  Repository
    allow VillaAllowlist
}

func NewService(repo Repository, allow VillaAllowlist) *Service

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

### 4.5 HTTP

```go
func init() {
    httpserver.Register(ErrUnknownVilla, http.StatusNotFound, "UNKNOWN_VILLA")
    httpserver.Register(ErrInvalidRange, http.StatusUnprocessableEntity, "INVALID_RANGE")
}

func Mount(r chi.Router, svc *Service) {
    r.Get("/api/villas/{slug}/pricing", listHandler(svc))
}
```

Request: `from` and `to` query params in `YYYY-MM-DD`. `to` is exclusive (matches the booking availability semantics).

Response (`200 OK`):

```json
{
  "overrides": [
    { "date": "2026-07-04", "price_cents": 25000 },
    { "date": "2026-07-05", "price_cents": 25000 }
  ]
}
```

Error codes (matches existing envelope `{error:{code,message}}`):
- 404 `UNKNOWN_VILLA`
- 422 `VALIDATION` (bad date params) / `INVALID_RANGE`
- 500 `INTERNAL`

### 4.6 Postgres adapter

Standard pgx + sqlc pattern. `pgtype.Date` ↔ `time.Time` conversion at the boundary (same shape as the `booking` adapter — see Plan 1 §3 of the backend design for the pattern).

### 4.7 Composition root

`cmd/server/casadana.go` gains:

```go
pricingSvc := pricing.NewService(pricing.NewPgRepo(pool), slugAllowlist{})
pricing.Mount(r, pricingSvc)
```

## 5. OpenAPI Spec Update

Add the new operation to `apps/api/internal/openapi/openapi.yaml`:

- Path `/api/villas/{slug}/pricing` with `getVillaPricing` operationId, tag `pricing` (new tag).
- Schemas: `PriceOverride` (`{date: date, price_cents: integer ≥0}`), `PricingResponse` (`{overrides: [PriceOverride]}`).
- Error responses 404, 422, 500 reference the existing `ErrorResponse`.

Regen Orval afterwards: `bun --filter @casa-dana/api generate`. New file `packages/api/src/generated/pricing/pricing.ts` will appear with `useGetVillaPricing` hook.

## 6. Frontend Changes

### 6.1 `villas.const.ts`

- Remove `cleaning` and `concierge` from the `VillaData["booking"]` type (lines 115-116 of the type definition).
- Remove the corresponding lines from both villa entries (lines 243-244 for casadana, 550-551 for casacasay).
- Change `nightly: 185` (casadana) and `nightly: 145` (casacasay) both to `nightly: 95`.

### 6.2 `villa-booking.tsx` — calendar cells

- New hook call: `useGetVillaPricing(villaSlug, { from, to }, { query: { enabled: activeField !== null, placeholderData: keepPreviousData } })`. Window matches the existing availability query window.
- Build `priceOverridesByDate: Map<string, number>` keyed by `yyyy-MM-dd`, value in **cents**.
- Helper `priceFor(date: Date): number` returns the override (cents) if present, else `booking.nightly * 100`.
- Cell render: when the cell is **available** (not blocked, not muted), render the day number on top and `€{priceFor(date) / 100}` below in a small mono font. Selected (CI/CO round) cells: omit the price (no space). Range-middle cells: keep the price. Blocked / muted / past cells: no price.

### 6.3 `villa-booking.tsx` — sidebar total

- Remove the cleaning + concierge `<div>` rows.
- Replace `subtotal = nights * booking.nightly` with `total = sum over each night d of priceFor(d) / 100`. The summed value is the total.
- Sidebar now shows only:
  ```
  3 nights                 €285
  ──────────────────────────────
  Total                    €285
  ```
- Remove i18n keys `villa_booking_cleaning_fee` and `villa_booking_concierge_welcome` from `messages/{en,es,fr}.json`. Regenerate paraglide.

### 6.4 Cell layout

ASCII sketch of the cell content:

```
┌──────┐
│  15  │   ← day number, existing styling
│ €95  │   ← new: mono 9px, tracking 0.05em, on-surface-variant color
└──────┘
```

Class for the price subtext: `font-mono text-[9px] tracking-tight text-on-surface-variant mt-0.5`.

The aspect-ratio of the cell may need a tiny relaxation since we add a line. Use `aspect-[1/1.15]` or remove `aspect-square` and add an explicit `min-h`. Implementation detail picked up by the plan.

## 7. Tests

- **Domain & service**: pure Go unit tests with fakes (`fakeRepo`, `fakeAllowlist`). Cases: happy path returns overrides; unknown villa → `ErrUnknownVilla`; `to <= from` → `ErrInvalidRange`.
- **Postgres adapter**: `//go:build integration` testcontainers test. Seed two overrides, query the window, assert order and content.
- **HTTP handler**: `httptest` + fake service. Verify 200 + JSON shape, 404 on unknown villa, 422 on missing/invalid `from`/`to`.
- **Frontend**: manual smoke test only. Seed `seed_dev.sql`, open the calendar, verify the override prices show on the correct dates and the total sums correctly.

## 8. Error Mapping

| HTTP | Code              | When                                       |
|------|-------------------|--------------------------------------------|
| 404  | `UNKNOWN_VILLA`   | slug not in `villaslug.Known`              |
| 422  | `VALIDATION`      | `from` or `to` missing / not YYYY-MM-DD    |
| 422  | `INVALID_RANGE`   | parsed dates with `to <= from`             |
| 500  | `INTERNAL`        | unexpected (logged)                        |

## 9. Configuration

No new env vars. Reuses the existing `WEB_ORIGIN`, postgres connection, etc.

## 10. Composition & wiring summary

Files created / modified by this feature:

**Backend new:**
- `apps/api/internal/pricing/{domain.go, ports.go, service.go, http.go, postgres.go, service_test.go, postgres_test.go}`
- `apps/api/internal/db/migrations/0002_price_overrides.{up,down}.sql`
- `apps/api/internal/db/queries/pricing.sql`
- `apps/api/internal/db/seed_dev.sql`

**Backend modified:**
- `apps/api/internal/openapi/openapi.yaml` (new path + schemas)
- `apps/api/cmd/server/casadana.go` (wire pricing module)
- `apps/api/internal/db/` (sqlc regenerates the pricing query)

**Frontend modified:**
- `packages/api/src/generated/**` (Orval regen — committed)
- `apps/web/src/constants/villas.const.ts` (defaults to 95, drop cleaning/concierge)
- `apps/web/src/components/sections/villa/villa-booking.tsx` (pricing hook, cell layout, total)
- `apps/web/messages/{en,es,fr}.json` (drop two keys)
- `apps/web/src/paraglide/**` (regenerated)

## 11. Out-of-Scope (explicit)

- Admin write endpoints for overrides — Plan 2.
- Lock booking prices for payment — when payment flow exists.
- Per-villa default in DB — when admin needs to edit baseline.
- Weekend / season multipliers, min-stay rules — when product needs them.
- Multi-currency.
- Booking record carrying a snapshot of the price at submission time — payment-flow-era concern.
- Any seeded data in production. Migrations are DDL-only; the dev seed file is never imported into the binary nor referenced by the embed FS.
