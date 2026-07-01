# Casa Dana — Go Backend Design

**Date:** 2026-05-12
**Status:** Approved — ready for implementation plan

## 1. Purpose & Scope

Build the Go backend (`apps/api`) for Casa Dana, a villa rental site. The backend supports four feature areas:

1. **Booking requests** — public form submissions, persisted, with confirmation email to the guest and notification email to the admin (via Resend).
2. **Availability calendar** — date-range conflict detection per villa; public endpoint the frontend queries to render the calendar.
3. **Admin panel + JWT auth** — authenticated admin endpoints to list and transition bookings, moderate reviews, manage admin users (seeded first admin + invite flow).
4. **Reviews** — public submission (moderated, default `pending`) and public listing of approved reviews per villa.

**Out of scope (MVP):**

- Backend-managed villa content. Villas (names, descriptions, photos, FAQ, amenities) stay **static in the frontend** (`apps/web/src/constants/villas.const.ts`). The backend references villas only by slug.
- Payments / payment gateway integration.
- Backend-side i18n. Content is stored as submitted; emails are not localized in MVP.
- Public user accounts. Auth is admin-only.

## 2. Architecture

**Style:** Hexagonal (Ports & Adapters), one Go package per bounded module (booking, review, auth). Each module package contains its own domain entities, application service, port interfaces, and adapters (HTTP and Postgres). The package boundary is the architectural boundary.

**Why not the textbook Clean Architecture 4-folder layout (`domain/`, `usecase/`, `interface/`, `infrastructure/`)?** In Go, splitting one feature across four packages forces cross-package wiring without adding isolation. Package-as-hexagon keeps each feature understandable in one screen while still being strictly hexagonal: `domain.go` imports only stdlib, `service.go` depends only on ports, adapters implement ports.

**Module isolation:** Modules MUST NOT import each other directly. Cross-module needs go through ports defined by the consumer (e.g. if `booking` ever needed villa data, it would define its own `VillaReader` interface — but in MVP no cross-module dependency exists).

### 2.1 Directory layout

```
apps/api/
├── cmd/server/
│   └── casadana.go               # composition root — the only file that wires modules
├── internal/
│   ├── platform/                 # cross-cutting infra, zero domain knowledge
│   │   ├── config/               # env → struct via caarlos0/env
│   │   ├── postgres/             # *pgxpool.Pool factory, WithTx helper
│   │   ├── httpserver/           # chi router, middleware stack, error mapper, graceful shutdown
│   │   ├── logger/               # slog JSON
│   │   ├── email/                # Resend client, implements module Mailer ports
│   │   ├── jwt/                  # token sign/verify
│   │   └── validator/            # go-playground/validator wrapper
│   ├── booking/                  # MODULE
│   │   ├── domain.go             # Booking entity, value objects, sentinel errors
│   │   ├── service.go            # use cases as methods on Service
│   │   ├── ports.go              # Repository, Mailer, Clock interfaces
│   │   ├── http.go               # chi handlers + DTO mapping + Mount()
│   │   ├── postgres.go           # Repository adapter using sqlc queries
│   │   ├── service_test.go       # fakes for ports
│   │   ├── postgres_test.go      # //go:build integration — testcontainers
│   │   └── http_test.go          # httptest against fake service
│   ├── review/                   # same shape as booking
│   ├── auth/                     # same shape + middleware.go (AdminOnly)
│   ├── villaslug/                # NOT a hexagon — just an allowlist
│   │   └── catalog.go            # var Known = map[string]struct{}{"casa-dana": {}, ...}
│   └── db/                       # sqlc-generated code
│       ├── migrations/           # golang-migrate
│       ├── queries/              # *.sql consumed by sqlc
│       └── *.go                  # generated
└── go.mod / go.sum / sqlc.yaml
```

### 2.2 Tech stack

| Concern | Choice | Notes |
|---|---|---|
| HTTP router | `github.com/go-chi/chi/v5` | already in `go.mod` |
| DB driver | `github.com/jackc/pgx/v5` + `pgxpool` | |
| Query gen | `sqlc` | already configured in `sqlc.yaml` |
| Migrations | `github.com/golang-migrate/migrate/v4` | runs on boot in dev/test; init-container in prod |
| Validation | `github.com/go-playground/validator/v10` | DTOs only — domain validates itself |
| Auth — hashing | `golang.org/x/crypto/bcrypt` | cost 12 |
| Auth — tokens | `github.com/golang-jwt/jwt/v5` | HS256 |
| Email | `github.com/resend/resend-go/v2` | uses `RESEND_API_KEY` env |
| Logging | stdlib `log/slog` | JSON handler |
| Config | `github.com/caarlos0/env/v11` | fail-fast on missing required vars |
| UUID | `github.com/google/uuid` | |
| Rate limiting | `github.com/go-chi/httprate` | on POST endpoints |
| CORS | `github.com/go-chi/cors` | |
| Test (integration) | `github.com/testcontainers/testcontainers-go` | Postgres |

## 3. Modules & Endpoints

### 3.1 Public endpoints

| Method | Path | Module | Purpose |
|---|---|---|---|
| GET  | `/api/health` | platform | Liveness |
| GET  | `/api/villas/{slug}/availability?from=&to=` | booking | Booked date ranges in window |
| POST | `/api/bookings` | booking | Submit booking → confirmation + admin notification email |
| GET  | `/api/villas/{slug}/reviews` | review | Approved reviews |
| POST | `/api/reviews` | review | Submit review (status=pending) |
| POST | `/api/admin/auth/login` | auth | Returns access token (body) + sets refresh cookie |
| POST | `/api/admin/auth/refresh` | auth | Rotate refresh, return new access |
| POST | `/api/admin/auth/accept-invite` | auth | Set password from invite token |

### 3.2 Admin endpoints (under `/api/admin/*`, behind `auth.AdminOnly`)

| Method | Path | Purpose |
|---|---|---|
| POST | `/api/admin/auth/logout` | Revoke current refresh token |
| POST | `/api/admin/auth/invite` | Email invite link to a new admin |
| GET  | `/api/admin/auth/me` | Current admin |
| GET, PATCH | `/api/admin/bookings`, `/api/admin/bookings/{id}` | List, approve/reject/cancel/mark-paid |
| GET, PATCH, DELETE | `/api/admin/reviews`, `/api/admin/reviews/{id}` | Moderate |

### 3.3 Per-module surface (illustrative — `booking`)

```go
// ports.go
type Repository interface {
    Save(ctx context.Context, b *Booking) error
    FindOverlapping(ctx context.Context, villaSlug string, from, to time.Time) ([]Booking, error)
    ListByStatus(ctx context.Context, status Status) ([]Booking, error)
    Get(ctx context.Context, id uuid.UUID) (*Booking, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error
}
type Mailer interface {
    SendBookingConfirmation(ctx context.Context, to string, b *Booking) error
    SendAdminNotification(ctx context.Context, b *Booking) error
}
type Clock interface { Now() time.Time }

// service.go
type Service struct { repo Repository; mailer Mailer; clock Clock }
func (s *Service) Create(ctx context.Context, cmd CreateBookingCommand) (*Booking, error)
func (s *Service) Availability(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error)
func (s *Service) Approve(ctx context.Context, id uuid.UUID) error
func (s *Service) Reject(ctx context.Context, id uuid.UUID, reason string) error
```

## 4. Domain Model

- **Booking** `{ id, villa_slug (string), guest_name, guest_email, guest_phone, check_in (date), check_out (date), adults, children, message, status, created_at, updated_at }`
  - Status: `pending → approved | rejected | cancelled | paid`
  - Invariants enforced in `NewBooking()` constructor: `check_out > check_in`; `adults >= 1`; valid email; villa_slug allowed.
- **Review** `{ id, villa_slug, author_name, rating (1..5), body, status, created_at }`
  - Status: `pending → approved | rejected`
- **AdminUser** `{ id, email, password_hash, status, invited_by, created_at }`
  - Status: `pending_invite | active | disabled`
- **Invite** `{ token_hash, email, expires_at (now+48h), used_at }`
- **RefreshToken** `{ token_hash, admin_user_id, expires_at (now+30d), revoked_at }`

No `Villa` table. Villas live in the frontend; backend only validates `villa_slug` against `internal/villaslug.Known`, kept in sync manually with `apps/web/src/constants/villas.const.ts`.

## 5. Data Flow & Transactions

**Booking submission (example):**

```
POST /api/bookings (JSON)
  → booking/http.go Handler
       decode JSON → CreateBookingRequest DTO
       validate DTO (validator tags)
       map → CreateBookingCommand
       call svc.Create(ctx, cmd)
  → booking/service.go Service.Create
       villaslug.IsKnown(cmd.VillaSlug) → ErrUnknownVilla (404)
       repo.FindOverlapping(...)        → ErrDatesConflict (409)
       domain.NewBooking(...)           → enforces invariants
       repo.Save(b)                     → inside a tx
       mailer.SendBookingConfirmation   → errors logged, not fatal
       mailer.SendAdminNotification     → errors logged, not fatal
       return b
  → handler maps domain → BookingResponse DTO → 201 JSON
```

**Transactions:** `platform/postgres` exposes `WithTx(ctx, fn func(Querier) error)`. Repository methods accept the sqlc `Querier` interface so they work against either the pool or a tx. Services decide when to open a tx (write-paths yes; read-paths no).

## 6. Error Handling

Domain sentinel errors:

```go
var (
    ErrDatesConflict     = errors.New("booking: dates conflict")
    ErrUnknownVilla      = errors.New("booking: unknown villa")
    ErrInvalidCredentials = errors.New("auth: invalid credentials")
    ErrForbidden         = errors.New("auth: forbidden")
    // ...
)
```

One HTTP error mapper in `platform/httpserver/errors.go`:

```go
switch {
case errors.Is(err, booking.ErrDatesConflict):      status = 409
case errors.Is(err, booking.ErrUnknownVilla):       status = 404
case errors.Is(err, auth.ErrInvalidCredentials):    status = 401
case errors.Is(err, auth.ErrForbidden):             status = 403
case errors.As(err, &validationErr):                status = 422
default:                                            status = 500; log full err with request_id
}
```

Response shape:

```json
{ "error": { "code": "DATES_CONFLICT", "message": "Those dates are not available." } }
```

500-class errors never leak details to clients; full error logged with request_id.

## 7. Middleware Stack

Applied in this order on every request:

```
RequestID → RealIP → Recoverer → SLogStructuredLogger → CORS → RateLimit (POSTs only)
```

- `slog` JSON with `request_id`, `method`, `path`, `status`, `latency_ms`, `bytes`.
- CORS origin from `WEB_ORIGIN` env.
- `httprate`: 5 req/min/IP on `POST /api/bookings`, `POST /api/reviews`, `POST /api/admin/auth/login`.

## 8. Auth Scheme

- **Login** (`POST /api/admin/auth/login`) — verifies email + bcrypt password. On success:
  - Issues a 15-minute **access token** (JWT HS256, contains `sub`, `exp`, `iat`) → returned in JSON body.
  - Issues a 32-byte random **refresh token** → returned as `HttpOnly; Secure; SameSite=Lax; Path=/api/admin/auth` cookie. Stored in DB as SHA-256 hash with `expires_at = now+30d`.
- **Refresh** (`POST /api/admin/auth/refresh`) — reads cookie, looks up hash, **rotates** (revokes old, issues new), returns new access + sets new cookie.
- **Logout** — revokes the refresh token row, clears cookie.
- **Invite flow:**
  1. Admin calls `POST /api/admin/auth/invite { email }`.
  2. Service creates an `Invite` row with `token_hash = sha256(plaintext)`, `expires_at = now+48h`.
  3. Resend sends an email with link `{frontend}/admin/accept-invite?token={plaintext}`.
  4. Recipient submits `POST /api/admin/auth/accept-invite { token, password }` — verifies hash, creates active `AdminUser`, marks invite used.
- **AdminOnly middleware** — extracts `Authorization: Bearer <jwt>`, verifies via `platform/jwt`, attaches `admin_user_id` to ctx; rejects with 401 on missing/invalid, 403 on disabled user.

## 9. Configuration (env)

| Var | Required | Purpose |
|---|---|---|
| `PORT` | no (default 8080) | HTTP port |
| `POSTGRES_HOST` `POSTGRES_PORT` `POSTGRES_USER` `POSTGRES_PASSWORD` `POSTGRES_DB` | yes | DB connection |
| `JWT_SECRET` | yes | HMAC secret for access tokens |
| `RESEND_API_KEY` | yes | Resend |
| `MAIL_FROM` | yes | Sender address |
| `WEB_ORIGIN` | yes | CORS allowed origin |
| `ADMIN_NOTIFY_EMAIL` | yes | Where admin booking notifications go |
| `ADMIN_BOOTSTRAP_EMAIL` `ADMIN_BOOTSTRAP_PASSWORD` | optional | Seed first admin on boot if no admin exists |
| `MIGRATE_ON_BOOT` | no (default true in dev) | Run migrations during startup |
| `LOG_LEVEL` | no (default info) | slog level |

Loaded via `caarlos0/env`, fails fast on missing required vars.

## 10. Composition Root

`cmd/server/casadana.go` is the only file that knows everything:

```go
func main() {
    cfg := config.Load()
    log := logger.New(cfg.LogLevel)
    pool := postgres.MustOpen(ctx, cfg.DB)
    defer pool.Close()

    mailer := email.NewResendMailer(cfg.ResendKey, cfg.MailFrom, cfg.AdminNotifyEmail)
    tokens := jwt.New(cfg.JWTSecret)
    clock  := platform.RealClock{}

    authSvc    := auth.NewService(auth.NewPgRepo(pool), mailer, tokens, clock)
    bookingSvc := booking.NewService(booking.NewPgRepo(pool), mailer, clock)
    reviewSvc  := review.NewService(review.NewPgRepo(pool), clock)

    r := httpserver.NewRouter(log, cfg.CORSOrigin)
    auth.Mount(r, authSvc)
    booking.Mount(r, bookingSvc, authSvc.Middleware())
    review.Mount(r, reviewSvc, authSvc.Middleware())

    httpserver.Run(r, cfg.Port)   // graceful shutdown on SIGTERM, 30s drain
}
```

Each module exposes a `Mount(router, svc, [middleware])` function. Modules own their routing; `cmd/server` owns the composition.

## 11. Testing Strategy

- **Service tests** (`service_test.go`) — pure Go unit tests, fakes for every port (in-memory `Repository`, recording `Mailer`, fixed `Clock`). Run on every save. Fast.
- **Postgres adapter tests** (`postgres_test.go`) — `//go:build integration` tag; spin up Postgres via `testcontainers-go`, run real sqlc queries against a fresh DB with migrations applied. Run in CI and on demand locally.
- **HTTP handler tests** (`http_test.go`) — `httptest` server with handlers wired to a fake service. Asserts status codes, response shape, error mapping, validation behaviour.
- **Smoke test** (optional) — one end-to-end test in `cmd/server` against testcontainers Postgres + Resend mock to verify wiring.

## 12. Security Baseline

- Passwords: bcrypt cost 12.
- Refresh + invite tokens: SHA-256 hashed at rest; plaintext never persisted.
- All SQL via sqlc (parameterized) — no string concatenation.
- Secrets only from env; never logged. `slog` redacts `password`, `token`, `authorization` fields.
- CORS strictly scoped to `WEB_ORIGIN`.
- Rate limits on auth + write endpoints.
- HTTP server has read/write/idle timeouts set (no default-Go infinite timeouts).

## 13. Operational

- **Migrations on boot** in dev (`MIGRATE_ON_BOOT=true`). In prod, a separate `migrate` init-container runs migrations before the API starts; the API itself runs with `MIGRATE_ON_BOOT=false`.
- **Admin bootstrap:** if `ADMIN_BOOTSTRAP_EMAIL` + `ADMIN_BOOTSTRAP_PASSWORD` are set and no admin row exists, seed one and log a warning to change the password.
- **Graceful shutdown:** SIGTERM → stop accepting connections → wait up to 30s for in-flight requests → close DB pool.
- **Health:** `/api/health` returns 200 + `{"status":"ok"}` (no DB check — separate `/api/health/ready` could be added later if needed).

## 14. Out of Scope (explicit non-goals)

- Backend villa CRUD or villa content API.
- Public user signup or guest accounts.
- Payment processing.
- Backend i18n for emails or content.
- File upload / asset storage (photos stay in frontend).
- Real-time / websockets.
- A separate read model / CQRS / event sourcing.
