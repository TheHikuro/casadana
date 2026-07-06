# Casa Dana — Admin Auth + Reservations Design

**Date:** 2026-07-02
**Status:** Approved — ready for implementation plan

## 1. Purpose & Scope

Build the first real screen of the admin dashboard imported from the Claude Design mockup (`admin.html`), scoped to two of its five tabs: **auth** (replacing the mockup's client-side passcode gate) and **Reservations**. The other three tabs (Pricing, Reviews, Owner & access) and History are separate subsystems with real domain gaps against the mockup and are deferred to later plans.

This plan also closes an existing security hole: `GET /api/bookings`, `PATCH /api/bookings/{id}`, and `DELETE /api/bookings/{id}` currently have **no auth at all** (see `docs/superpowers/specs/2026-05-15-bookings-pricing-reviews-batch-design.md` §9, which explicitly deferred this to "Plan 2" — this is that plan).

**In scope:**

- New `internal/adminauth` module: admin user accounts (email + bcrypt password hash), login/logout, signed session cookie, `RequireAdminSession` middleware.
- In-app admin user management (list/create/delete), flat access model — any admin manages both properties, no roles.
- `RequireAdminSession` applied to `GET/PATCH/DELETE /api/bookings*`. `POST /api/bookings` (public booking creation) stays open.
- `booking` module: add `villa_slug` filter to `List`/`Count`; add `source` field (`direct` default) to the domain, DB, and DTOs.
- Frontend: split the root layout so `/admin` gets its own chrome; admin login page; authenticated admin shell (sidebar, property switcher, topbar) matching the mockup; Reservations view (list/filter/paginate/add/status-transition/delete).
- New `Toast` UI primitive (`components/ui`), matching the mockup's toast styling, used for mutation feedback.
- OpenAPI spec extension + Orval regen for all of the above.

**Out of scope (explicit):**

- Pricing tab (base rate/fees/season-rule ranges — no backend concept of this exists yet; current pricing model is per-date overrides only).
- Reviews tab (mockup wants freeform curated reviews with an aggregate rating breakdown; current `review` module requires a real `booking_id` and has no "featured"/aggregate concept).
- Owner & access tab beyond user management (host contact, payout/IBAN, notes — undecided where sensitive payout data should live).
- History / activity log (no such feature exists anywhere yet).
- Reservation `total`/price column (depends on the deferred Pricing subsystem).
- Session revocation list / "log out other devices" (stateless signed cookie has no server-side revocation; acceptable tradeoff for a 1–2 admin internal tool).
- Frontend automated tests (no test infra exists in `apps/web` today; not introduced as a side effect here).

## 2. Architecture Decisions

### 2.1 Session mechanism: stateless HMAC-signed cookie, not a sessions table

`config.Config.JWTSecret` (`JWT_SECRET` env var) already exists, required, and is currently unused anywhere in the codebase — it was clearly reserved for this. Rather than add a `golang-jwt` dependency or a `sessions` table, the token is a minimal custom format signed with that secret:

```
token = "<admin_id>.<expiry_unix>.<hex(hmac_sha256(admin_id + "." + expiry_unix, secret))>"
```

`admin_id` is a UUID (no `.` characters), so `.` is a safe delimiter. Verification recomputes the HMAC and compares with `hmac.Equal` (constant-time), then checks `expiry_unix > now`. Expiry is 12h from issuance; the frontend redirects to `/admin/login` on any 401 rather than silently refreshing.

Tradeoff accepted: logout only clears the cookie client-side — a copied token stays valid until it expires. Given flat access and a tiny admin user base, this is an acceptable simplicity/security tradeoff versus a `sessions` table with revocation. Documented here so it's a deliberate choice, not an oversight.

### 2.2 Cookie attributes

`httpOnly`, `SameSite=Lax`, `Secure` — gated by a new `COOKIE_SECURE` config bool (`envDefault:"true"`), set to `false` in `.env.dev` since local dev runs over plain `http://localhost`. Name: `casa_admin_session`. Path: `/`.

### 2.3 Flat admin access, in-app user management

Any authenticated admin can manage both properties and can create/delete other admin users — no role tiers, no per-property scoping (per your decision). Guardrails: `DELETE /api/admin/users/{id}` returns `409 CANNOT_DELETE_SELF` if `id` is the caller's own id, and `409 LAST_ADMIN` if it's the only remaining admin user — prevents locking everyone out.

### 2.4 Booking auth retrofit, not a new admin-scoped booking API

Rather than duplicating booking endpoints under `/api/admin/bookings`, `RequireAdminSession` is attached directly to the existing `GET/PATCH/DELETE /api/bookings*` routes in the composition root. `POST /api/bookings` is unaffected (guests book without an account). The admin "Add reservation" form calls this same public `POST /api/bookings` — no separate admin-only creation path, so it inherits the real availability-conflict and villa-allowlist checks (per your earlier decision).

### 2.5 `source` field on bookings

New nullable-but-defaulted `source TEXT NOT NULL DEFAULT 'direct'` column. `createBookingRequest` gains an optional `source` field (defaults server-side to `"direct"` if omitted, so the existing public booking form on the villa pages needs no change). Admin's manual-add form sends an explicit value (`direct` / `airbnb` / `booking_com`).

### 2.6 Reservation status stays the real 5-state machine

The admin UI surfaces `pending / approved / rejected / cancelled / paid` with the existing `Booking.Transition` rules enforced server-side (already implemented, unchanged). The reservations table renders a status badge plus a small set of action buttons scoped to whatever transitions are actually legal from the current state (e.g. a `pending` row shows "Approve" / "Reject" / "Cancel"; an `approved` row shows "Mark paid" / "Cancel"), rather than the mockup's free `<select>` with 3 options.

### 2.7 `villa_slug` filter on booking list

`booking.Service.List` and `Repository.List`/`Count` gain an optional `villaSlug *string` parameter (validated against `villaslug.IsKnown` in the service, `ErrUnknownVilla` on mismatch), so the admin property switcher filters server-side.

### 2.8 Frontend root layout split

`__root.tsx` currently wraps every route in the public `Navbar`/`Footer`. Split into:
- `__root.tsx` → bare `<Outlet />` (+ global providers already there).
- `routes/_public.tsx` → pathless layout carrying `Navbar`/`Footer`; existing `index.tsx` and `villa/$villaId.tsx` move to be its children (mechanical move, no behavior change for public pages).
- `routes/admin/*` → its own tree, described in §7.

### 2.9 Property switcher as a validated search param

`?property=casadana|casacasay` on admin routes (TanStack Router search-param validation against the villa allowlist), not component state — survives refresh and back/forward, matches how the rest of the app already treats shareable state.

### 2.10 Sidebar shows all 5 nav items; only Reservations is live

For visual fidelity to the approved mockup, the sidebar renders all 5 nav items (Reservations, Pricing, Reviews, Owner & access, History). The latter 4 route to a shared `ComingSoon` placeholder component until their respective plans land.

## 3. Database

### 3.1 Migration `0004_admin_users.up.sql`

```sql
CREATE TABLE admin_users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 3.2 Migration `0004_admin_users.down.sql`

```sql
DROP TABLE IF EXISTS admin_users;
```

### 3.3 Migration `0005_booking_source.up.sql`

```sql
ALTER TABLE bookings ADD COLUMN source TEXT NOT NULL DEFAULT 'direct';
```

### 3.4 Migration `0005_booking_source.down.sql`

```sql
ALTER TABLE bookings DROP COLUMN source;
```

### 3.5 sqlc additions — new `admin_users.sql`

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

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM admin_users;

-- name: DeleteAdminUser :execrows
DELETE FROM admin_users WHERE id = $1;
```

### 3.6 sqlc additions to `bookings.sql`

```sql
-- name: ListBookingsPagedByVillaAndStatus :many
SELECT * FROM bookings
WHERE villa_slug = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListBookingsPagedByVilla :many
SELECT * FROM bookings
WHERE villa_slug = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountBookingsByVillaAndStatus :one
SELECT COUNT(*) FROM bookings WHERE villa_slug = $1 AND status = $2;

-- name: CountBookingsByVilla :one
SELECT COUNT(*) FROM bookings WHERE villa_slug = $1;
```

(The existing unfiltered/status-only queries stay; the repo picks the right query based on which filters are non-nil.)

## 4. Backend modules

### 4.1 New `internal/adminauth` module

Layout, mirroring the existing `review` module:

```
internal/adminauth/
├── domain.go         # AdminUser entity, token sign/verify, sentinel errors
├── ports.go          # Repository, Clock
├── service.go        # Login, Logout(no-op server side), CreateUser, ListUsers, DeleteUser, Authenticate(token)
├── http.go           # Mount + handlers + DTOs + RequireAdminSession middleware
├── postgres.go        # adapter
├── fakes_test.go
├── domain_test.go     # token sign/verify incl. expiry + tamper cases
├── service_test.go
├── http_test.go
└── postgres_test.go   # //go:build integration
```

Domain:

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
    ErrLastAdmin          = errors.New("cannot delete the last remaining admin")
    ErrInvalidToken       = errors.New("invalid or expired session")
)

const sessionTTL = 12 * time.Hour

func hashPassword(plain string) (string, error) {
    b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
    return string(b), err
}

func verifyPassword(hash, plain string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

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

Ports:

```go
type Repository interface {
    Save(ctx context.Context, u *AdminUser) error              // ErrEmailTaken on unique conflict
    FindByEmail(ctx context.Context, email string) (*AdminUser, error) // ErrNotFound
    FindByID(ctx context.Context, id string) (*AdminUser, error)       // ErrNotFound
    List(ctx context.Context) ([]AdminUser, error)
    Count(ctx context.Context) (int, error)
    Delete(ctx context.Context, id string) error               // ErrNotFound if 0 rows
}

type Clock interface { Now() time.Time }
```

Service (sketch — full method bodies in the implementation plan):

```go
type Service struct {
    repo   Repository
    clock  Clock
    secret string
}

func (s *Service) Login(ctx context.Context, email, password string) (token string, err error) {
    // FindByEmail -> ErrInvalidCredentials on not-found (don't leak which)
    // verifyPassword -> ErrInvalidCredentials on mismatch
    // signToken(user.ID, s.secret, s.clock.Now())
}

func (s *Service) Authenticate(ctx context.Context, token string) (*AdminUser, error) {
    // verifyToken -> ErrInvalidToken
    // FindByID -> ErrInvalidToken if the user was deleted after the token was issued
}

func (s *Service) CreateUser(ctx context.Context, email, password string) (*AdminUser, error) {
    // hashPassword, repo.Save -> ErrEmailTaken passthrough
}

func (s *Service) ListUsers(ctx context.Context) ([]AdminUser, error) { /* repo.List */ }

func (s *Service) DeleteUser(ctx context.Context, callerID, targetID string) error {
    // targetID == callerID -> ErrCannotDeleteSelf
    // repo.Count() == 1 -> ErrLastAdmin
    // repo.Delete -> ErrNotFound passthrough
}
```

HTTP (sketch):

```go
func init() {
    httpserver.Register(ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS")
    httpserver.Register(ErrEmailTaken, http.StatusConflict, "EMAIL_TAKEN")
    httpserver.Register(ErrNotFound, http.StatusNotFound, "ADMIN_NOT_FOUND")
    httpserver.Register(ErrCannotDeleteSelf, http.StatusConflict, "CANNOT_DELETE_SELF")
    httpserver.Register(ErrLastAdmin, http.StatusConflict, "LAST_ADMIN")
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

// RequireAdminSession is exported so cmd/server can also apply it to the
// existing GET/PATCH/DELETE /api/bookings* routes.
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
            ctx := context.WithValue(r.Context(), adminCtxKey{}, admin)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### 4.2 `booking` module additions

- `Repository.List`/`Count` gain a `villaSlug *string` parameter alongside the existing `status *Status`.
- `Service.List(ctx, villaSlug *string, status *Status, page, limit int)` — validates `villaSlug` against the allowlist when non-nil.
- `Booking` domain struct and `NewBookingInput`/`CreateCommand` gain `Source string`; `NewBooking` defaults it to `"direct"` when empty.
- `createBookingRequest` DTO gains optional `source` (`validate:"omitempty,oneof=direct airbnb booking_com"`); `bookingResponse` gains `source`.
- `listBookingsHandler` reads an optional `villa_slug` query param.
- `Mount` signature changes to accept a generic middleware, keeping `booking` decoupled from `adminauth` (no import of it — same "inject behavior, don't import the module" pattern already used for `Mailer`/`BookingReader`):

```go
// Mount wires booking routes. requireAuth guards the admin-only routes
// (list/patch/delete); POST (public booking creation) and the availability
// read are left open.
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

  `cmd/server/casadana.go` passes `adminauth.RequireAdminSession(adminSvc)` as `requireAuth`.

## 5. OpenAPI

New tag: `admin`. New operations: 6.

| Operation ID | Method | Path | Tag | Auth |
|---|---|---|---|---|
| `adminLogin` | POST | `/api/admin/login` | admin | none |
| `adminLogout` | POST | `/api/admin/logout` | admin | none |
| `adminMe` | GET | `/api/admin/me` | admin | session |
| `listAdminUsers` | GET | `/api/admin/users` | admin | session |
| `createAdminUser` | POST | `/api/admin/users` | admin | session |
| `deleteAdminUser` | DELETE | `/api/admin/users/{id}` | admin | session |

Existing operations updated: `listBookings` gains `villa_slug` query param; `CreateBookingRequest`/`BookingResponse` gain `source`; all four `/api/bookings*` operations gain `security: [{ adminSession: [] }]` except `createBooking`.

New schemas:
- `AdminLoginRequest` — `{email, password}`
- `AdminUser` — `{id, email, created_at}` (no password hash exposed)
- `ListAdminUsersResponse` — `{users: [AdminUser]}`
- `CreateAdminUserRequest` — `{email, password (min 8)}`
- `BookingSource` — enum `direct|airbnb|booking_com`

`components.securitySchemes.adminSession` — `type: apiKey, in: cookie, name: casa_admin_session` (documents the cookie; orval's axios mutator already sends cookies via `withCredentials: true`, no client-side change needed).

## 6. Orval regen

`bun --filter @casa-dana/api generate` produces `packages/api/src/generated/admin/admin.ts` with `useAdminLogin`, `useAdminLogout`, `useAdminMe` (+ `getAdminMeQueryOptions` for use in `beforeLoad`), `useListAdminUsers`, `useCreateAdminUser`, `useDeleteAdminUser`. Existing `bookings.ts` regenerates with the `villa_slug` param and `source` field. Barrel `packages/api/src/index.ts` adds the new export.

## 7. Frontend

### 7.1 Route tree

```
apps/web/src/routes/
├── __root.tsx              # bare <Outlet/>, unchanged providers
├── _public.tsx              # NEW: pathless layout, Navbar/Footer (moved from __root)
├── index.tsx                 # moved under _public (same content)
├── villa/$villaId.tsx         # moved under _public (same content)
└── admin/
    ├── login.tsx              # NEW: standalone, no sidebar
    ├── route.tsx              # NEW: layout — beforeLoad auth check + AdminShell
    ├── index.tsx              # NEW: redirect -> /admin/reservations
    ├── reservations.tsx       # NEW: functional
    ├── pricing.tsx             # NEW: <ComingSoon/>
    ├── reviews.tsx             # NEW: <ComingSoon/>
    ├── owner.tsx               # NEW: <ComingSoon/>
    └── history.tsx             # NEW: <ComingSoon/>
```

`admin/route.tsx`:

```tsx
export const Route = createFileRoute("/admin")({
  beforeLoad: async ({ context, location }) => {
    try {
      await context.queryClient.fetchQuery(getGetAdminMeQueryOptions())
    } catch {
      throw redirect({ to: "/admin/login", search: { redirect: location.href } })
    }
  },
  component: AdminShell,
})
```

`AdminShell` renders the sidebar (property switcher + 5 nav buttons, matching the mockup's markup/structure translated to React + existing Tailwind tokens) and an `<Outlet/>` for the active view.

### 7.2 Property switcher

```tsx
export const Route = createFileRoute("/admin/reservations")({
  validateSearch: z.object({
    property: z.enum(["casadana", "casacasay"]).default("casadana"),
    status: z.enum(["pending", "approved", "rejected", "cancelled", "paid"]).optional(),
    page: z.number().int().min(1).default(1),
  }),
  component: ReservationsView,
})
```

### 7.3 Reservations view

- `useListBookings({ villa_slug: property, status, page, limit: 8 }, { placeholderData: keepPreviousData })` — mirrors the pagination pattern already used in `villa-booking.tsx`.
- Status column renders a badge + the legal-next-state action buttons (§2.6) wired to `usePatchBooking`, `onSuccess` invalidates the list query key.
- Delete wired to `useDeleteBooking`, same invalidation, with a confirm step (native `confirm()` is fine here, matching the mockup — no need for a custom dialog in this phase).
- "Add reservation" opens a form (modal, reusing the mockup's field layout) calling `useCreateBooking` with `source`; on success, invalidates the list and shows a toast.
- Stat row (Total / Pending / Approved / Paid / Cancelled counts): computing these from the current page would be wrong once paginated. Instead, issue 5 lightweight parallel calls — `useListBookings({ villa_slug, limit: 1 })` (no status, for Total) plus one per status (`useListBookings({ villa_slug, status, limit: 1 })`) — and read `total` off each response, ignoring the single returned row. `limit: 1` keeps the payload trivial; this reuses the existing endpoint with no backend change.

### 7.4 New `Toast` UI primitive

`components/ui/toast.tsx` — a minimal, self-contained toast (show/timeout/hide), styled with the existing cva/Tailwind convention, visually matching the mockup's `.toast` styling. A `useToast()` hook exposes `toast(message)`, called from mutation `onSuccess`/`onError` handlers.

## 8. Tests

- **Backend unit** (`adminauth/domain_test.go`): token sign/verify round-trip, expired token rejected, tampered signature rejected, wrong secret rejected.
- **Backend unit** (`adminauth/service_test.go`, fakes): login success/wrong-password/unknown-email, create user success/duplicate-email, delete user self/last-admin/not-found/happy-path.
- **Backend HTTP** (`adminauth/http_test.go`): login sets cookie with correct attributes; logout clears it; `/api/admin/me` 401 without cookie, 200 with valid cookie; `/api/admin/users` 401 without session.
- **Backend HTTP** (`booking/http_test.go` additions): `GET/PATCH/DELETE /api/bookings*` return 401 without a session cookie; `villa_slug` filter narrows results; `source` round-trips through create → response.
- **Backend integration** (`adminauth/postgres_test.go`, `//go:build integration`): save/find/list/delete against real Postgres via testcontainers, unique-email conflict mapped to `ErrEmailTaken`.
- **Frontend**: manual verification against the dev server — login with correct/incorrect credentials, session persists across reload, logout redirects to login, property switcher filters the table, status action buttons only show legal transitions, add/delete reservation, 401 on a stale/cleared cookie redirects to login. No automated frontend test infra introduced (none exists in this repo today; see §1).

## 9. Out-of-Scope (explicit, restated from §1)

- Pricing, Reviews, Owner & access (beyond user mgmt), History — separate future plans, each with real domain gaps against the mockup documented in the brainstorming session.
- Session revocation / multi-device logout.
- Role-based access control or per-property admin scoping.
- Reservation price/total display.
- Frontend automated tests.
