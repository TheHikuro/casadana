# Casa Dana Go Backend — Plan 1: Foundation + Public Booking Flow

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Commit policy:** DO NOT run `git commit` during execution of this plan. Make file changes only. The user runs commits manually at their preferred boundaries. (Memory: `feedback-no-auto-commit`.)

**Goal:** Stand up the Go API with hexagonal foundation (config, logger, postgres, http, email) plus a working public booking flow: submit a booking, get a confirmation email, query availability.

**Architecture:** Package-as-hexagon. `internal/platform/*` holds infra primitives with no domain knowledge. `internal/booking/` is one self-contained vertical slice (domain + service + ports + http adapter + postgres adapter). `cmd/server/casadana.go` wires everything.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5 + pgxpool, sqlc, golang-migrate, slog, go-playground/validator/v10, resend-go/v2, caarlos0/env/v11, google/uuid, testcontainers-go.

**Scope of this plan:**
- Platform packages: config, logger, postgres, httpserver, validator, email
- `internal/villaslug/` allowlist
- DB migrations infrastructure + initial `bookings` table
- sqlc queries for bookings
- Full `internal/booking/` module (domain → service → ports → postgres → http)
- Endpoints: `GET /api/health`, `POST /api/bookings`, `GET /api/villas/{slug}/availability`
- Tests: service-with-fakes, postgres-with-testcontainers, http-with-httptest

**Out of scope (covered by future plans):**
- Auth module + admin endpoints (Plan 2)
- Review module (Plan 3)
- Rate limiting, CORS hardening, graceful-shutdown polish, prod ops (Plan 4)

**Working dir for all paths below:** `apps/api/` (i.e. `apps/api/internal/...`).

**Spec reference:** `docs/superpowers/specs/2026-05-12-go-backend-design.md`.

---

## Phase A — Project skeleton & dependencies

### Task A1: Add dependencies and create directory skeleton

**Files:**
- Modify: `apps/api/go.mod`
- Create: directories under `apps/api/internal/`

- [ ] **Step 1: Add dependencies**

Run from `apps/api/`:

```bash
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/go-playground/validator/v10
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/resend/resend-go/v2
go get github.com/caarlos0/env/v11
go get github.com/google/uuid
go mod tidy
```

- [ ] **Step 2: Create directory skeleton**

Run from `apps/api/`:

```bash
mkdir -p internal/platform/config
mkdir -p internal/platform/logger
mkdir -p internal/platform/postgres
mkdir -p internal/platform/httpserver
mkdir -p internal/platform/validator
mkdir -p internal/platform/email
mkdir -p internal/villaslug
mkdir -p internal/booking
mkdir -p internal/db/queries
mkdir -p internal/db/migrations
```

- [ ] **Step 3: Verify `go build ./...` still passes**

Run: `go build ./...`
Expected: no output, exit 0.

---

## Phase B — Platform packages

### Task B1: Config package

**Files:**
- Create: `apps/api/internal/platform/config/config.go`
- Create: `apps/api/internal/platform/config/config_test.go`

- [ ] **Step 1: Write the failing test**

`internal/platform/config/config_test.go`:

```go
package config

import (
	"testing"
)

func TestLoad_ReadsRequiredEnv(t *testing.T) {
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
	if cfg.Port != 8080 {
		t.Errorf("default Port = %d, want 8080", cfg.Port)
	}
	if cfg.DB.Host != "localhost" || cfg.DB.Name != "casadana" {
		t.Errorf("DB config not parsed: %+v", cfg.DB)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}

func TestLoad_FailsOnMissingRequired(t *testing.T) {
	// no env set — JWT_SECRET required
	t.Setenv("POSTGRES_HOST", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error from missing required env, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/config/...`
Expected: fails to compile — `Load` and `Config` undefined.

- [ ] **Step 3: Implement config**

`internal/platform/config/config.go`:

```go
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type DBConfig struct {
	Host     string `env:"POSTGRES_HOST,required"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	Name     string `env:"POSTGRES_DB,required"`
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

type Config struct {
	Port             int    `env:"PORT" envDefault:"8080"`
	LogLevel         string `env:"LOG_LEVEL" envDefault:"info"`
	DB               DBConfig
	JWTSecret        string `env:"JWT_SECRET,required"`
	ResendKey        string `env:"RESEND_API_KEY,required"`
	MailFrom         string `env:"MAIL_FROM,required"`
	WebOrigin        string `env:"WEB_ORIGIN,required"`
	AdminNotifyEmail string `env:"ADMIN_NOTIFY_EMAIL,required"`
	MigrateOnBoot    bool   `env:"MIGRATE_ON_BOOT" envDefault:"true"`
}

func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return c, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/config/...`
Expected: PASS.

---

### Task B2: Logger package

**Files:**
- Create: `apps/api/internal/platform/logger/logger.go`

- [ ] **Step 1: Implement logger**

`internal/platform/logger/logger.go`:

```go
package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: redactSensitive,
	})
	return slog.New(h)
}

var sensitiveKeys = map[string]struct{}{
	"password":      {},
	"token":         {},
	"authorization": {},
	"jwt":           {},
	"secret":        {},
}

func redactSensitive(_ []string, a slog.Attr) slog.Attr {
	if _, sensitive := sensitiveKeys[strings.ToLower(a.Key)]; sensitive {
		return slog.String(a.Key, "[REDACTED]")
	}
	return a
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/platform/logger/...`
Expected: exit 0.

---

### Task B3: Postgres pool + WithTx helper

**Files:**
- Create: `apps/api/internal/platform/postgres/pool.go`
- Create: `apps/api/internal/platform/postgres/tx.go`

- [ ] **Step 1: Implement pool factory**

`internal/platform/postgres/pool.go`:

```go
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse dsn: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 2: Implement tx helper**

`internal/platform/postgres/tx.go`:

```go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the minimal interface sqlc generates against. Both *pgxpool.Pool
// and pgx.Tx satisfy it, so repositories can accept this and work in either
// a pooled connection or a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgx.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx runs fn inside a transaction. Commits on nil error, rolls back otherwise.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/platform/postgres/...`
Expected: exit 0.

---

### Task B4: HTTP error mapper

**Files:**
- Create: `apps/api/internal/platform/httpserver/errors.go`
- Create: `apps/api/internal/platform/httpserver/errors_test.go`

- [ ] **Step 1: Write the failing test**

`internal/platform/httpserver/errors_test.go`:

```go
package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var errSentinel = errors.New("test: sentinel")

func TestWriteError_KnownStatus(t *testing.T) {
	Register(errSentinel, http.StatusConflict, "TEST_CONFLICT")

	rec := httptest.NewRecorder()
	WriteError(rec, nil, errSentinel)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"TEST_CONFLICT"`) {
		t.Errorf("body = %q, missing code", body)
	}
}

func TestWriteError_UnknownDefaultsTo500(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, nil, errors.New("boom"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("internal error message leaked to client")
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/platform/httpserver/...`
Expected: compile error — `Register`, `WriteError` undefined.

- [ ] **Step 3: Implement error mapper**

`internal/platform/httpserver/errors.go`:

```go
package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
)

type mapping struct {
	status int
	code   string
}

var (
	mu       sync.RWMutex
	registry = map[error]mapping{}
)

// Register declares how a sentinel error should be rendered.
// Modules call this in init() so the HTTP layer doesn't need to know domain errors.
func Register(err error, status int, code string) {
	mu.Lock()
	defer mu.Unlock()
	registry[err] = mapping{status: status, code: code}
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ValidationError lets handlers signal a 422 with a public message.
type ValidationError struct{ Message string }

func (v *ValidationError) Error() string { return v.Message }

// WriteError converts an error into a JSON response. Unknown errors render as 500
// with a generic message; the underlying error is logged with the request's logger.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	mu.RLock()
	defer mu.RUnlock()

	for sentinel, m := range registry {
		if errors.Is(err, sentinel) {
			writeJSON(w, m.status, m.code, sentinel.Error())
			return
		}
	}
	var vErr *ValidationError
	if errors.As(err, &vErr) {
		writeJSON(w, http.StatusUnprocessableEntity, "VALIDATION", vErr.Message)
		return
	}
	if r != nil {
		slog.ErrorContext(r.Context(), "internal server error", "err", err.Error(), "path", r.URL.Path)
	}
	writeJSON(w, http.StatusInternalServerError, "INTERNAL", "Something went wrong.")
}

func writeJSON(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var body errorBody
	body.Error.Code = code
	body.Error.Message = msg
	_ = json.NewEncoder(w).Encode(body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/httpserver/...`
Expected: PASS.

---

### Task B5: HTTP router + middleware stack

**Files:**
- Create: `apps/api/internal/platform/httpserver/router.go`
- Create: `apps/api/internal/platform/httpserver/server.go`

- [ ] **Step 1: Implement router factory**

`internal/platform/httpserver/router.go`:

```go
package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(log *slog.Logger) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(structuredLogger(log))
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}

func structuredLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"latency_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
```

- [ ] **Step 2: Implement server with graceful shutdown skeleton**

`internal/platform/httpserver/server.go`:

```go
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(handler http.Handler, port int, log *slog.Logger) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		log.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-stopCh:
		log.Info("shutting down", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/platform/httpserver/...`
Expected: exit 0.

---

### Task B6: Validator wrapper

**Files:**
- Create: `apps/api/internal/platform/validator/validator.go`

- [ ] **Step 1: Implement validator wrapper**

`internal/platform/validator/validator.go`:

```go
package validator

import (
	"fmt"
	"strings"
	"sync"

	pv "github.com/go-playground/validator/v10"
)

var (
	once sync.Once
	v    *pv.Validate
)

func instance() *pv.Validate {
	once.Do(func() {
		v = pv.New(pv.WithRequiredStructEnabled())
	})
	return v
}

// Struct validates s and returns a flat human-readable error suitable for a
// 422 response. nil if valid.
func Struct(s any) error {
	err := instance().Struct(s)
	if err == nil {
		return nil
	}
	var verrs pv.ValidationErrors
	if !errorsAs(err, &verrs) {
		return err
	}
	parts := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		parts = append(parts, fmt.Sprintf("%s: %s", fe.Field(), fe.Tag()))
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}

// errorsAs is errors.As but typed for our single use to keep imports tight.
func errorsAs(err error, target *pv.ValidationErrors) bool {
	for e := err; e != nil; {
		if ve, ok := e.(pv.ValidationErrors); ok {
			*target = ve
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := e.(unwrapper)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/platform/validator/...`
Expected: exit 0.

---

### Task B7: Email package (Resend)

**Files:**
- Create: `apps/api/internal/platform/email/resend.go`

- [ ] **Step 1: Implement Resend mailer**

`internal/platform/email/resend.go`:

```go
package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type Mailer struct {
	client      *resend.Client
	from        string
	adminNotify string
}

func NewMailer(apiKey, from, adminNotify string) *Mailer {
	return &Mailer{
		client:      resend.NewClient(apiKey),
		from:        from,
		adminNotify: adminNotify,
	}
}

// Message is the generic shape consumed by module Mailer ports. Each module
// defines its own Mailer interface; the methods below satisfy them.
type Message struct {
	To      string
	Subject string
	HTML    string
}

func (m *Mailer) Send(ctx context.Context, msg Message) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("email: mailer not configured")
	}
	_, err := m.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.HTML,
	})
	if err != nil {
		return fmt.Errorf("email: send: %w", err)
	}
	return nil
}

func (m *Mailer) AdminNotifyAddress() string { return m.adminNotify }
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/platform/email/...`
Expected: exit 0.

---

### Task B8: villaslug allowlist

**Files:**
- Create: `apps/api/internal/villaslug/catalog.go`
- Create: `apps/api/internal/villaslug/catalog_test.go`

- [ ] **Step 1: Inspect existing villa slugs**

Run from repo root: `grep -E "slug:" apps/web/src/constants/villas.const.ts | head`
Note: pick the actual slugs used by the frontend. For this plan we will start with two; update the file if your slugs differ.

- [ ] **Step 2: Write the failing test**

`internal/villaslug/catalog_test.go`:

```go
package villaslug

import "testing"

func TestIsKnown(t *testing.T) {
	cases := []struct {
		slug string
		want bool
	}{
		{"casadana", true},
		{"casacasay", true},
		{"unknown-villa", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsKnown(c.slug); got != c.want {
			t.Errorf("IsKnown(%q) = %v, want %v", c.slug, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Implement catalog**

`internal/villaslug/catalog.go`:

```go
// Package villaslug is the authoritative allowlist of villa slugs the API
// will accept. It mirrors apps/web/src/constants/villas.const.ts and must be
// updated manually when a villa is added or removed from the frontend.
package villaslug

var known = map[string]struct{}{
	"casadana": {},
	"casacasay":  {},
}

func IsKnown(slug string) bool {
	_, ok := known[slug]
	return ok
}

// All returns the known slugs in no particular order. Useful for diagnostics.
func All() []string {
	out := make([]string, 0, len(known))
	for s := range known {
		out = append(out, s)
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/villaslug/...`
Expected: PASS.

---

## Phase C — Database setup

### Task C1: Migration runner

**Files:**
- Create: `apps/api/internal/platform/postgres/migrate.go`

- [ ] **Step 1: Implement migrator**

`internal/platform/postgres/migrate.go`:

```go
package postgres

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// MigrateUp runs all pending migrations from the given filesystem source.
// sourceURL example: "file://internal/db/migrations"
func MigrateUp(pool *pgxpool.Pool, sourceURL string) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("migrate: driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: new: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/platform/postgres/...`
Expected: exit 0. (May require `go get github.com/jackc/pgx/v5/stdlib` if not present — `go mod tidy` afterwards.)

---

### Task C2: Initial migration — bookings table

**Files:**
- Create: `apps/api/internal/db/migrations/0001_init.up.sql`
- Create: `apps/api/internal/db/migrations/0001_init.down.sql`

- [ ] **Step 1: Write up migration**

`internal/db/migrations/0001_init.up.sql`:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE booking_status AS ENUM (
    'pending',
    'approved',
    'rejected',
    'cancelled',
    'paid'
);

CREATE TABLE bookings (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    villa_slug   TEXT        NOT NULL,
    guest_name   TEXT        NOT NULL,
    guest_email  TEXT        NOT NULL,
    guest_phone  TEXT        NOT NULL,
    check_in     DATE        NOT NULL,
    check_out    DATE        NOT NULL,
    adults       SMALLINT    NOT NULL CHECK (adults  >= 1),
    children     SMALLINT    NOT NULL DEFAULT 0 CHECK (children >= 0),
    message      TEXT        NOT NULL DEFAULT '',
    status       booking_status NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT booking_dates_valid CHECK (check_out > check_in)
);

CREATE INDEX bookings_villa_slug_dates_idx
    ON bookings (villa_slug, check_in, check_out)
    WHERE status IN ('approved', 'paid');
```

- [ ] **Step 2: Write down migration**

`internal/db/migrations/0001_init.down.sql`:

```sql
DROP TABLE IF EXISTS bookings;
DROP TYPE  IF EXISTS booking_status;
```

---

### Task C3: sqlc queries for bookings

**Files:**
- Create: `apps/api/internal/db/queries/bookings.sql`

- [ ] **Step 1: Write queries**

`internal/db/queries/bookings.sql`:

```sql
-- name: InsertBooking :one
INSERT INTO bookings (
    id, villa_slug, guest_name, guest_email, guest_phone,
    check_in, check_out, adults, children, message, status
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11
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
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: ListBookedRanges :many
SELECT check_in, check_out FROM bookings
WHERE villa_slug = $1
  AND status IN ('approved', 'paid')
  AND check_in  < $3
  AND check_out > $2
ORDER BY check_in;
```

- [ ] **Step 2: Generate sqlc code**

Run from `apps/api/`: `sqlc generate`
Expected: files written under `internal/db/` (per `sqlc.yaml`). If `sqlc` is not installed, install via `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: exit 0.

---

## Phase D — Booking module

### Task D1: Booking domain (entity, value objects, errors)

**Files:**
- Create: `apps/api/internal/booking/domain.go`
- Create: `apps/api/internal/booking/domain_test.go`

- [ ] **Step 1: Write failing tests**

`internal/booking/domain_test.go`:

```go
package booking

import (
	"strings"
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

func validCmd() NewBookingInput {
	return NewBookingInput{
		VillaSlug:  "casadana",
		GuestName:  "Jane Doe",
		GuestEmail: "jane@example.com",
		GuestPhone: "+33123456789",
		CheckIn:    d("2026-07-01"),
		CheckOut:   d("2026-07-08"),
		Adults:     2,
		Children:   1,
		Message:    "Looking forward to it",
		Now:        d("2026-05-12"),
	}
}

func TestNewBooking_Happy(t *testing.T) {
	b, err := NewBooking(validCmd())
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	if b.Status != StatusPending {
		t.Errorf("status = %s, want pending", b.Status)
	}
	if b.ID == "" {
		t.Error("ID was empty")
	}
}

func TestNewBooking_Rejects(t *testing.T) {
	cases := map[string]func(*NewBookingInput){
		"check_out before check_in": func(c *NewBookingInput) { c.CheckOut = c.CheckIn.AddDate(0, 0, -1) },
		"check_out equals check_in": func(c *NewBookingInput) { c.CheckOut = c.CheckIn },
		"check_in in the past":      func(c *NewBookingInput) { c.CheckIn = c.Now.AddDate(0, 0, -1); c.CheckOut = c.Now },
		"adults < 1":                func(c *NewBookingInput) { c.Adults = 0 },
		"children < 0":              func(c *NewBookingInput) { c.Children = -1 },
		"empty guest name":          func(c *NewBookingInput) { c.GuestName = strings.Repeat(" ", 3) },
		"empty guest email":         func(c *NewBookingInput) { c.GuestEmail = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validCmd()
			mutate(&c)
			if _, err := NewBooking(c); err == nil {
				t.Fatalf("expected error for %q, got nil", name)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/booking/...`
Expected: compile error.

- [ ] **Step 3: Implement domain**

`internal/booking/domain.go`:

```go
package booking

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
	StatusCancelled Status = "cancelled"
	StatusPaid      Status = "paid"
)

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
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

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
	Now        time.Time // injected so tests are deterministic
}

var (
	ErrDatesConflict = errors.New("those dates are not available")
	ErrUnknownVilla  = errors.New("unknown villa")
	ErrInvalidStatus = errors.New("invalid booking status transition")
)

func NewBooking(in NewBookingInput) (*Booking, error) {
	in.GuestName = strings.TrimSpace(in.GuestName)
	in.GuestEmail = strings.TrimSpace(in.GuestEmail)
	in.VillaSlug = strings.TrimSpace(in.VillaSlug)

	if in.VillaSlug == "" {
		return nil, errors.New("villa_slug required")
	}
	if in.GuestName == "" {
		return nil, errors.New("guest_name required")
	}
	if in.GuestEmail == "" || !strings.Contains(in.GuestEmail, "@") {
		return nil, errors.New("guest_email invalid")
	}
	if in.Adults < 1 {
		return nil, errors.New("adults must be >= 1")
	}
	if in.Children < 0 {
		return nil, errors.New("children must be >= 0")
	}
	if !in.CheckOut.After(in.CheckIn) {
		return nil, errors.New("check_out must be after check_in")
	}
	if in.CheckIn.Before(in.Now.Truncate(24 * time.Hour)) {
		return nil, errors.New("check_in must not be in the past")
	}

	now := in.Now
	return &Booking{
		ID:         uuid.NewString(),
		VillaSlug:  in.VillaSlug,
		GuestName:  in.GuestName,
		GuestEmail: in.GuestEmail,
		GuestPhone: in.GuestPhone,
		CheckIn:    in.CheckIn,
		CheckOut:   in.CheckOut,
		Adults:     in.Adults,
		Children:   in.Children,
		Message:    in.Message,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Transition returns the booking with a new status if the transition is allowed.
func (b Booking) Transition(next Status, now time.Time) (Booking, error) {
	allowed := map[Status]map[Status]bool{
		StatusPending:  {StatusApproved: true, StatusRejected: true, StatusCancelled: true},
		StatusApproved: {StatusPaid: true, StatusCancelled: true},
		StatusPaid:     {StatusCancelled: true},
	}
	if !allowed[b.Status][next] {
		return Booking{}, ErrInvalidStatus
	}
	b.Status = next
	b.UpdatedAt = now
	return b, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/booking/...`
Expected: PASS.

---

### Task D2: Booking ports

**Files:**
- Create: `apps/api/internal/booking/ports.go`

- [ ] **Step 1: Implement ports**

`internal/booking/ports.go`:

```go
package booking

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, b *Booking) error
	FindOverlapping(ctx context.Context, villaSlug string, from, to time.Time) ([]Booking, error)
	BookedRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error)
	Get(ctx context.Context, id string) (*Booking, error)
	UpdateStatus(ctx context.Context, id string, status Status, updatedAt time.Time) error
}

type Mailer interface {
	SendBookingConfirmation(ctx context.Context, b *Booking) error
	SendAdminNotification(ctx context.Context, b *Booking) error
}

type Clock interface {
	Now() time.Time
}

type VillaAllowlist interface {
	IsKnown(slug string) bool
}

type DateRange struct {
	CheckIn  time.Time
	CheckOut time.Time
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/booking/...`
Expected: exit 0.

---

### Task D3: Test doubles (fakes for service tests)

**Files:**
- Create: `apps/api/internal/booking/fakes_test.go`

- [ ] **Step 1: Implement fakes**

`internal/booking/fakes_test.go`:

```go
package booking

import (
	"context"
	"errors"
	"time"
)

type fakeRepo struct {
	saved        []Booking
	overlapping  []Booking
	bookedRanges []DateRange
	saveErr      error
}

func (f *fakeRepo) Save(_ context.Context, b *Booking) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, *b)
	return nil
}
func (f *fakeRepo) FindOverlapping(_ context.Context, _ string, _, _ time.Time) ([]Booking, error) {
	return f.overlapping, nil
}
func (f *fakeRepo) BookedRanges(_ context.Context, _ string, _, _ time.Time) ([]DateRange, error) {
	return f.bookedRanges, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (*Booking, error) {
	for i := range f.saved {
		if f.saved[i].ID == id {
			b := f.saved[i]
			return &b, nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeRepo) UpdateStatus(_ context.Context, id string, status Status, updatedAt time.Time) error {
	for i := range f.saved {
		if f.saved[i].ID == id {
			f.saved[i].Status = status
			f.saved[i].UpdatedAt = updatedAt
			return nil
		}
	}
	return errors.New("not found")
}

type fakeMailer struct {
	confirmations  []Booking
	adminNotices   []Booking
	confirmErr     error
	adminNoticeErr error
}

func (f *fakeMailer) SendBookingConfirmation(_ context.Context, b *Booking) error {
	if f.confirmErr != nil {
		return f.confirmErr
	}
	f.confirmations = append(f.confirmations, *b)
	return nil
}
func (f *fakeMailer) SendAdminNotification(_ context.Context, b *Booking) error {
	if f.adminNoticeErr != nil {
		return f.adminNoticeErr
	}
	f.adminNotices = append(f.adminNotices, *b)
	return nil
}

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

type fakeAllowlist struct{ allowed map[string]bool }

func (f fakeAllowlist) IsKnown(slug string) bool { return f.allowed[slug] }
```

- [ ] **Step 2: Verify build**

Run: `go test -run NONE ./internal/booking/...`
Expected: PASS (no tests selected, just verifies compilation).

---

### Task D4: Service.Create (TDD)

**Files:**
- Create: `apps/api/internal/booking/service.go`
- Create: `apps/api/internal/booking/service_test.go`

- [ ] **Step 1: Write failing tests**

`internal/booking/service_test.go`:

```go
package booking

import (
	"context"
	"testing"
	"time"
)

func newSvc(repo Repository, mailer Mailer, allow VillaAllowlist, now time.Time) *Service {
	return NewService(repo, mailer, allow, fixedClock{t: now})
}

func TestCreate_Happy(t *testing.T) {
	repo := &fakeRepo{}
	mailer := &fakeMailer{}
	allow := fakeAllowlist{allowed: map[string]bool{"casadana": true}}
	svc := newSvc(repo, mailer, allow, d("2026-05-12"))

	b, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug:  "casadana",
		GuestName:  "Jane",
		GuestEmail: "jane@example.com",
		GuestPhone: "+33",
		CheckIn:    d("2026-07-01"),
		CheckOut:   d("2026-07-08"),
		Adults:     2,
		Children:   0,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, want := len(repo.saved), 1; got != want {
		t.Errorf("saved count = %d, want %d", got, want)
	}
	if got, want := len(mailer.confirmations), 1; got != want {
		t.Errorf("confirmations = %d, want %d", got, want)
	}
	if got, want := len(mailer.adminNotices), 1; got != want {
		t.Errorf("admin notices = %d, want %d", got, want)
	}
	if b.Status != StatusPending {
		t.Errorf("status = %s, want pending", b.Status)
	}
}

func TestCreate_UnknownVilla(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{}}, d("2026-05-12"))

	_, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug: "ghost-villa",
		GuestName: "X", GuestEmail: "x@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 1,
	})
	if err == nil || !isErr(err, ErrUnknownVilla) {
		t.Fatalf("err = %v, want ErrUnknownVilla", err)
	}
	if len(repo.saved) != 0 {
		t.Error("repo should not have been written")
	}
}

func TestCreate_DatesConflict(t *testing.T) {
	repo := &fakeRepo{overlapping: []Booking{{ID: "x"}}}
	svc := newSvc(repo, &fakeMailer{}, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))

	_, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug: "casadana",
		GuestName: "Jane", GuestEmail: "jane@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 1,
	})
	if err == nil || !isErr(err, ErrDatesConflict) {
		t.Fatalf("err = %v, want ErrDatesConflict", err)
	}
	if len(repo.saved) != 0 {
		t.Error("repo should not have been written")
	}
}

func TestCreate_MailerFailure_DoesNotFailBooking(t *testing.T) {
	repo := &fakeRepo{}
	mailer := &fakeMailer{confirmErr: errBoom}
	svc := newSvc(repo, mailer, fakeAllowlist{allowed: map[string]bool{"casadana": true}}, d("2026-05-12"))

	_, err := svc.Create(context.Background(), CreateCommand{
		VillaSlug: "casadana",
		GuestName: "Jane", GuestEmail: "jane@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Error("booking should have been persisted despite mailer failure")
	}
}

// helpers
var errBoom = simpleErr("boom")

type simpleErr string

func (s simpleErr) Error() string { return string(s) }

func isErr(err, target error) bool {
	for e := err; e != nil; {
		if e == target {
			return true
		}
		type unwrap interface{ Unwrap() error }
		u, ok := e.(unwrap)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/booking/...`
Expected: compile error — `Service`, `NewService`, `CreateCommand`, `svc.Create` undefined.

- [ ] **Step 3: Implement service**

`internal/booking/service.go`:

```go
package booking

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Service struct {
	repo   Repository
	mailer Mailer
	allow  VillaAllowlist
	clock  Clock
}

func NewService(repo Repository, mailer Mailer, allow VillaAllowlist, clock Clock) *Service {
	return &Service{repo: repo, mailer: mailer, allow: allow, clock: clock}
}

type CreateCommand struct {
	VillaSlug  string
	GuestName  string
	GuestEmail string
	GuestPhone string
	CheckIn    time.Time
	CheckOut   time.Time
	Adults     int
	Children   int
	Message    string
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*Booking, error) {
	if !s.allow.IsKnown(cmd.VillaSlug) {
		return nil, ErrUnknownVilla
	}

	overlapping, err := s.repo.FindOverlapping(ctx, cmd.VillaSlug, cmd.CheckIn, cmd.CheckOut)
	if err != nil {
		return nil, fmt.Errorf("booking: check overlap: %w", err)
	}
	if len(overlapping) > 0 {
		return nil, ErrDatesConflict
	}

	b, err := NewBooking(NewBookingInput{
		VillaSlug:  cmd.VillaSlug,
		GuestName:  cmd.GuestName,
		GuestEmail: cmd.GuestEmail,
		GuestPhone: cmd.GuestPhone,
		CheckIn:    cmd.CheckIn,
		CheckOut:   cmd.CheckOut,
		Adults:     cmd.Adults,
		Children:   cmd.Children,
		Message:    cmd.Message,
		Now:        s.clock.Now(),
	})
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, b); err != nil {
		return nil, fmt.Errorf("booking: save: %w", err)
	}

	// Emails are best-effort: a transient mail failure must not lose the booking.
	if err := s.mailer.SendBookingConfirmation(ctx, b); err != nil {
		slog.WarnContext(ctx, "booking confirmation email failed", "booking_id", b.ID, "err", err.Error())
	}
	if err := s.mailer.SendAdminNotification(ctx, b); err != nil {
		slog.WarnContext(ctx, "admin notification email failed", "booking_id", b.ID, "err", err.Error())
	}

	return b, nil
}

func (s *Service) Availability(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error) {
	if !s.allow.IsKnown(villaSlug) {
		return nil, ErrUnknownVilla
	}
	if !to.After(from) {
		return nil, fmt.Errorf("booking: 'to' must be after 'from'")
	}
	return s.repo.BookedRanges(ctx, villaSlug, from, to)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/booking/...`
Expected: PASS.

---

### Task D5: Postgres adapter (integration test with testcontainers)

**Files:**
- Create: `apps/api/internal/booking/postgres.go`
- Create: `apps/api/internal/booking/postgres_test.go`

- [ ] **Step 1: Implement adapter**

`internal/booking/postgres.go`:

```go
package booking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheHikuro/casadana/internal/db"
)

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewPgRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) queries() *db.Queries { return db.New(r.pool) }

func (r *pgRepo) Save(ctx context.Context, b *Booking) error {
	id, err := uuid.Parse(b.ID)
	if err != nil {
		return fmt.Errorf("booking: invalid id: %w", err)
	}
	_, err = r.queries().InsertBooking(ctx, db.InsertBookingParams{
		ID:         id,
		VillaSlug:  b.VillaSlug,
		GuestName:  b.GuestName,
		GuestEmail: b.GuestEmail,
		GuestPhone: b.GuestPhone,
		CheckIn:    b.CheckIn,
		CheckOut:   b.CheckOut,
		Adults:     int16(b.Adults),
		Children:   int16(b.Children),
		Message:    b.Message,
		Status:     db.BookingStatus(b.Status),
	})
	return err
}

func (r *pgRepo) FindOverlapping(ctx context.Context, villaSlug string, from, to time.Time) ([]Booking, error) {
	rows, err := r.queries().FindOverlappingBookings(ctx, db.FindOverlappingBookingsParams{
		VillaSlug: villaSlug,
		CheckIn:   from,
		CheckOut:  to,
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

func (r *pgRepo) BookedRanges(ctx context.Context, villaSlug string, from, to time.Time) ([]DateRange, error) {
	rows, err := r.queries().ListBookedRanges(ctx, db.ListBookedRangesParams{
		VillaSlug: villaSlug,
		CheckIn:   from,
		CheckOut:  to,
	})
	if err != nil {
		return nil, err
	}
	out := make([]DateRange, 0, len(rows))
	for _, row := range rows {
		out = append(out, DateRange{CheckIn: row.CheckIn, CheckOut: row.CheckOut})
	}
	return out, nil
}

func (r *pgRepo) Get(ctx context.Context, id string) (*Booking, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("booking: invalid id: %w", err)
	}
	row, err := r.queries().GetBookingByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	b := rowToBooking(row)
	return &b, nil
}

func (r *pgRepo) UpdateStatus(ctx context.Context, id string, status Status, _ time.Time) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("booking: invalid id: %w", err)
	}
	return r.queries().UpdateBookingStatus(ctx, db.UpdateBookingStatusParams{
		ID:     uid,
		Status: db.BookingStatus(status),
	})
}

func rowToBooking(row db.Booking) Booking {
	return Booking{
		ID:         row.ID.String(),
		VillaSlug:  row.VillaSlug,
		GuestName:  row.GuestName,
		GuestEmail: row.GuestEmail,
		GuestPhone: row.GuestPhone,
		CheckIn:    row.CheckIn,
		CheckOut:   row.CheckOut,
		Adults:     int(row.Adults),
		Children:   int(row.Children),
		Message:    row.Message,
		Status:     Status(row.Status),
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
```

NOTE: the exact field names returned by sqlc depend on its codegen settings. After running `sqlc generate`, adjust `db.InsertBookingParams` / `db.Booking` field accesses to match — the names should align since the SQL columns match Go identifiers (e.g. `villa_slug` → `VillaSlug`).

- [ ] **Step 2: Write integration test**

`internal/booking/postgres_test.go`:

```go
//go:build integration

package booking

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

	abs, _ := filepath.Abs("../../internal/db/migrations")
	if err := pg.MigrateUp(pool, "file://"+abs); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func TestPgRepo_SaveAndFindOverlap(t *testing.T) {
	pool := setupPg(t)
	repo := NewPgRepo(pool)
	ctx := context.Background()

	b, err := NewBooking(NewBookingInput{
		VillaSlug: "casadana", GuestName: "Jane", GuestEmail: "jane@example.com",
		CheckIn: d("2026-07-01"), CheckOut: d("2026-07-08"),
		Adults: 2, Now: d("2026-05-12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, b); err != nil {
		t.Fatalf("save: %v", err)
	}

	overlapping, err := repo.FindOverlapping(ctx, "casadana", d("2026-07-05"), d("2026-07-10"))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(overlapping) != 1 {
		t.Errorf("overlapping = %d, want 1", len(overlapping))
	}
}
```

- [ ] **Step 3: Run unit tests (no integration)**

Run: `go test ./internal/booking/...`
Expected: PASS (integration test skipped by build tag).

- [ ] **Step 4: Run integration test if Docker is available**

Run: `go test -tags integration ./internal/booking/...`
Expected: PASS. If Docker isn't running, skip this step and note it for later CI.

---

### Task D6: HTTP handler

**Files:**
- Create: `apps/api/internal/booking/http.go`
- Create: `apps/api/internal/booking/http_test.go`
- Create: `apps/api/internal/booking/mailer_adapter.go`

- [ ] **Step 1: Implement mailer adapter (bridge between booking.Mailer and platform/email)**

`internal/booking/mailer_adapter.go`:

```go
package booking

import (
	"context"
	"fmt"

	"github.com/TheHikuro/casadana/internal/platform/email"
)

type ResendMailer struct {
	inner *email.Mailer
}

func NewResendMailer(m *email.Mailer) Mailer { return &ResendMailer{inner: m} }

func (r *ResendMailer) SendBookingConfirmation(ctx context.Context, b *Booking) error {
	return r.inner.Send(ctx, email.Message{
		To:      b.GuestEmail,
		Subject: "Your Casa Dana booking request",
		HTML: fmt.Sprintf(
			`<p>Hi %s,</p><p>We received your booking request for <b>%s</b> from %s to %s. We'll be in touch shortly.</p>`,
			b.GuestName, b.VillaSlug,
			b.CheckIn.Format("2006-01-02"), b.CheckOut.Format("2006-01-02"),
		),
	})
}

func (r *ResendMailer) SendAdminNotification(ctx context.Context, b *Booking) error {
	return r.inner.Send(ctx, email.Message{
		To:      r.inner.AdminNotifyAddress(),
		Subject: fmt.Sprintf("New booking request — %s", b.VillaSlug),
		HTML: fmt.Sprintf(
			`<p>New booking request for <b>%s</b></p>
<ul>
  <li>Guest: %s &lt;%s&gt; %s</li>
  <li>Dates: %s → %s</li>
  <li>Guests: %d adults, %d children</li>
  <li>Message: %s</li>
</ul>`,
			b.VillaSlug, b.GuestName, b.GuestEmail, b.GuestPhone,
			b.CheckIn.Format("2006-01-02"), b.CheckOut.Format("2006-01-02"),
			b.Adults, b.Children, b.Message,
		),
	})
}
```

- [ ] **Step 2: Implement HTTP layer**

`internal/booking/http.go`:

```go
package booking

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/validator"
)

func init() {
	httpserver.Register(ErrDatesConflict, http.StatusConflict, "DATES_CONFLICT")
	httpserver.Register(ErrUnknownVilla, http.StatusNotFound, "UNKNOWN_VILLA")
	httpserver.Register(ErrInvalidStatus, http.StatusConflict, "INVALID_STATUS")
}

func Mount(r chi.Router, svc *Service) {
	r.Post("/api/bookings", createHandler(svc))
	r.Get("/api/villas/{slug}/availability", availabilityHandler(svc))
}

type createBookingRequest struct {
	VillaSlug  string `json:"villa_slug"  validate:"required,min=1,max=64"`
	GuestName  string `json:"guest_name"  validate:"required,min=1,max=120"`
	GuestEmail string `json:"guest_email" validate:"required,email,max=255"`
	GuestPhone string `json:"guest_phone" validate:"required,min=4,max=40"`
	CheckIn    string `json:"check_in"    validate:"required,datetime=2006-01-02"`
	CheckOut   string `json:"check_out"   validate:"required,datetime=2006-01-02"`
	Adults     int    `json:"adults"      validate:"required,min=1,max=20"`
	Children   int    `json:"children"    validate:"min=0,max=20"`
	Message    string `json:"message"     validate:"max=2000"`
}

type bookingResponse struct {
	ID         string `json:"id"`
	VillaSlug  string `json:"villa_slug"`
	Status     string `json:"status"`
	CheckIn    string `json:"check_in"`
	CheckOut   string `json:"check_out"`
	GuestName  string `json:"guest_name"`
	GuestEmail string `json:"guest_email"`
	CreatedAt  string `json:"created_at"`
}

func toResponse(b *Booking) bookingResponse {
	return bookingResponse{
		ID:         b.ID,
		VillaSlug:  b.VillaSlug,
		Status:     string(b.Status),
		CheckIn:    b.CheckIn.Format("2006-01-02"),
		CheckOut:   b.CheckOut.Format("2006-01-02"),
		GuestName:  b.GuestName,
		GuestEmail: b.GuestEmail,
		CreatedAt:  b.CreatedAt.Format(time.RFC3339),
	}
}

func createHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createBookingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "invalid json: " + err.Error()})
			return
		}
		if err := validator.Struct(&req); err != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: err.Error()})
			return
		}
		checkIn, _ := time.Parse("2006-01-02", req.CheckIn)
		checkOut, _ := time.Parse("2006-01-02", req.CheckOut)

		b, err := svc.Create(r.Context(), CreateCommand{
			VillaSlug: req.VillaSlug, GuestName: req.GuestName,
			GuestEmail: req.GuestEmail, GuestPhone: req.GuestPhone,
			CheckIn: checkIn, CheckOut: checkOut,
			Adults: req.Adults, Children: req.Children, Message: req.Message,
		})
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(toResponse(b))
	}
}

type availabilityResponse struct {
	BookedRanges []struct {
		CheckIn  string `json:"check_in"`
		CheckOut string `json:"check_out"`
	} `json:"booked_ranges"`
}

func availabilityHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		from, err1 := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		to, err2 := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if err1 != nil || err2 != nil {
			httpserver.WriteError(w, r, &httpserver.ValidationError{Message: "from and to must be YYYY-MM-DD"})
			return
		}
		ranges, err := svc.Availability(r.Context(), slug, from, to)
		if err != nil {
			httpserver.WriteError(w, r, err)
			return
		}
		var resp availabilityResponse
		for _, rng := range ranges {
			resp.BookedRanges = append(resp.BookedRanges, struct {
				CheckIn  string `json:"check_in"`
				CheckOut string `json:"check_out"`
			}{
				CheckIn:  rng.CheckIn.Format("2006-01-02"),
				CheckOut: rng.CheckOut.Format("2006-01-02"),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
```

- [ ] **Step 3: Write HTTP test**

`internal/booking/http_test.go`:

```go
package booking

import (
	"bytes"
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

func TestPostBooking_Created(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33123","check_in":"2026-07-01","check_out":"2026-07-08","adults":2,"children":0,"message":"hi"}`
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out bookingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Status != "pending" {
		t.Errorf("status = %q, want pending", out.Status)
	}
}

func TestPostBooking_ValidationError(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := bytes.NewBufferString(`{"villa_slug":"casadana"}`)
	resp, err := http.Post(srv.URL+"/api/bookings", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestPostBooking_DatesConflict(t *testing.T) {
	repo := &fakeRepo{overlapping: []Booking{{ID: "x"}}}
	svc := newSvc(repo, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
	defer srv.Close()

	body := `{"villa_slug":"casadana","guest_name":"Jane","guest_email":"jane@example.com","guest_phone":"+33","check_in":"2026-07-01","check_out":"2026-07-08","adults":1,"children":0}`
	resp, _ := http.Post(srv.URL+"/api/bookings", "application/json", strings.NewReader(body))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestGetAvailability_Empty(t *testing.T) {
	svc := newSvc(&fakeRepo{}, &fakeMailer{},
		fakeAllowlist{allowed: map[string]bool{"casadana": true}},
		d("2026-05-12"))
	srv := httptest.NewServer(newRouter(svc))
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

- [ ] **Step 4: Run all booking tests**

Run: `go test ./internal/booking/...`
Expected: PASS.

---

## Phase E — Wire it up

### Task E1: Replace `cmd/server/casadana.go`

**Files:**
- Modify: `apps/api/cmd/server/casadana.go`

- [ ] **Step 1: Rewrite main**

`apps/api/cmd/server/casadana.go`:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/TheHikuro/casadana/internal/booking"
	"github.com/TheHikuro/casadana/internal/platform/config"
	"github.com/TheHikuro/casadana/internal/platform/email"
	"github.com/TheHikuro/casadana/internal/platform/httpserver"
	"github.com/TheHikuro/casadana/internal/platform/logger"
	"github.com/TheHikuro/casadana/internal/platform/postgres"
	"github.com/TheHikuro/casadana/internal/villaslug"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type slugAllowlist struct{}

func (slugAllowlist) IsKnown(slug string) bool { return villaslug.IsKnown(slug) }

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err.Error())
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx := context.Background()
	pool, err := postgres.Open(ctx, cfg.DB.DSN())
	if err != nil {
		log.Error("postgres open failed", "err", err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.MigrateOnBoot {
		if err := postgres.MigrateUp(pool, "file://internal/db/migrations"); err != nil {
			log.Error("migrate failed", "err", err.Error())
			os.Exit(1)
		}
		log.Info("migrations applied")
	}

	mailer := email.NewMailer(cfg.ResendKey, cfg.MailFrom, cfg.AdminNotifyEmail)
	bookingSvc := booking.NewService(
		booking.NewPgRepo(pool),
		booking.NewResendMailer(mailer),
		slugAllowlist{},
		realClock{},
	)

	r := httpserver.NewRouter(log)
	booking.Mount(r, bookingSvc)

	if err := httpserver.Run(r, cfg.Port, log); err != nil {
		log.Error("server crashed", "err", err.Error())
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Run full build**

Run from `apps/api/`: `go build ./...`
Expected: exit 0.

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: PASS across config, validator (none), httpserver, villaslug, booking. Integration tests under `-tags integration` should be run separately.

---

### Task E2: Verify against running stack (manual smoke)

**Files:** none.

- [ ] **Step 1: Boot the dev stack**

Run from repo root: `docker compose -f .docker/docker-compose.yml --profile dev up -d postgres api`
Wait until `api` is healthy.

- [ ] **Step 2: Health check**

Run: `curl -s http://localhost:8080/api/health`
Expected: `{"status":"ok"}`.

- [ ] **Step 3: Submit a booking**

Run:

```bash
curl -i -X POST http://localhost:8080/api/bookings \
  -H "Content-Type: application/json" \
  -d '{
    "villa_slug":"casadana",
    "guest_name":"Jane Doe",
    "guest_email":"jane@example.com",
    "guest_phone":"+33123456789",
    "check_in":"2026-07-01",
    "check_out":"2026-07-08",
    "adults":2,
    "children":1,
    "message":"Looking forward to it"
  }'
```

Expected: HTTP 201 with a JSON body containing `"status":"pending"` and an `id`. Resend should receive two API calls (confirmation + admin notification) — verify in the Resend dashboard or logs.

- [ ] **Step 4: Check availability**

Run: `curl -s 'http://localhost:8080/api/villas/casadana/availability?from=2026-06-01&to=2026-08-01'`
Expected: JSON `{"booked_ranges":[]}` (empty until a booking is moved to `approved`; in Plan 2 the admin will be able to transition it).

- [ ] **Step 5: Submit an overlapping booking**

Re-run the same POST as Step 3 — but for now, pending bookings DO block via `FindOverlapping`, so it should return HTTP 409 with `{"error":{"code":"DATES_CONFLICT","message":"..."}}`.

- [ ] **Step 6: Stop the stack**

Run: `docker compose -f .docker/docker-compose.yml --profile dev down`

---

## Post-flight

- [ ] **Files touched (sanity list):**
  - Created: `apps/api/internal/platform/config/{config.go,config_test.go}`
  - Created: `apps/api/internal/platform/logger/logger.go`
  - Created: `apps/api/internal/platform/postgres/{pool.go,tx.go,migrate.go}`
  - Created: `apps/api/internal/platform/httpserver/{router.go,server.go,errors.go,errors_test.go}`
  - Created: `apps/api/internal/platform/validator/validator.go`
  - Created: `apps/api/internal/platform/email/resend.go`
  - Created: `apps/api/internal/villaslug/{catalog.go,catalog_test.go}`
  - Created: `apps/api/internal/db/migrations/{0001_init.up.sql,0001_init.down.sql}`
  - Created: `apps/api/internal/db/queries/bookings.sql`
  - Created (generated): `apps/api/internal/db/*.go`
  - Created: `apps/api/internal/booking/{domain.go,domain_test.go,ports.go,fakes_test.go,service.go,service_test.go,postgres.go,postgres_test.go,http.go,http_test.go,mailer_adapter.go}`
  - Modified: `apps/api/cmd/server/casadana.go`, `apps/api/go.mod`, `apps/api/go.sum`

- [ ] **Things deferred to Plan 2:**
  - Auth module, admin endpoints, AdminOnly middleware
  - Admin booking transitions (approve / reject / cancel / mark-paid)
  - Admin invite flow

- [ ] **Things deferred to Plan 3:**
  - Review module (domain, service, postgres, http)

- [ ] **Things deferred to Plan 4:**
  - Rate limiting on POST endpoints
  - CORS strict configuration (`WEB_ORIGIN` plumbed through)
  - Dockerfile updates (multi-stage with migrations baked in; init container for prod migrations)
  - Production hardening (read-only filesystem, distroless image, etc.)
