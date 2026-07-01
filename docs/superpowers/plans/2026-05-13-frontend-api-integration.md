# Frontend ↔ Backend Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Commit policy:** DO NOT run `git commit` during execution. Make file changes only. The user runs commits manually at their preferred boundaries. (Memory: `feedback-no-auto-commit`.)

**Goal:** Wire the React frontend to the Go backend with a hand-authored OpenAPI spec, an Orval-generated React Query client living in a new `packages/api` workspace, and the booking form + calendar fully integrated with the API.

**Architecture:** OpenAPI YAML at `apps/api/openapi.yaml` is the source of truth. Orval reads it from `packages/api/orval.config.ts` and emits typed React Query hooks (+ zod schemas) into `packages/api/src/generated/`. All web hooks call a single axios instance (`packages/api/src/client.ts`) that resolves the base URL from `VITE_API_BASE_URL` and unwraps the `{error:{code,message}}` envelope into a typed `ApiError`. Backend gets CORS middleware (env-driven allowlist). The booking form replaces its `console.log` with `useCreateBooking`; the calendar reads `useGetVillaAvailability` and strikes through occupied nights (excluding the checkout day).

**Tech Stack:** OpenAPI 3.0.3, Orval 7, axios, `@tanstack/react-query` 5, `react-hook-form` (already in repo), `date-fns` (already in repo), `go-chi/cors`.

**Spec:** `docs/superpowers/specs/2026-05-13-frontend-api-integration-design.md`.

**Scope of this plan:**
- Phase A — Backend CORS
- Phase B — Author `apps/api/openapi.yaml`
- Phase C — Create `packages/api` workspace (delete `packages/types`)
- Phase D — Wire React Query in `apps/web`
- Phase E — Wire booking mutation in the form
- Phase F — Wire availability query in the calendar
- Phase G — Manual smoke test

**Out of scope (deferred):**
- Auth + admin endpoints (Plan 2 of the backend).
- Reviews module endpoints (Plan 3).
- `boneyard-js` skeleton screens (deferred to Plan 2/3 when list/table UIs appear).
- Optimistic booking mutations (need rollback logic on conflict — later).
- Splitting `guests` into adults+children form UI.

---

## Phase A — Backend CORS

### Task A1: Add go-chi/cors dependency

**Files:**
- Modify: `apps/api/go.mod`, `apps/api/go.sum`

- [ ] **Step 1: Install dependency**

Run from `apps/api/`:

```bash
go get github.com/go-chi/cors
go mod tidy
```

- [ ] **Step 2: Verify build still passes**

Run from `apps/api/`: `go build ./...`
Expected: exit 0.

---

### Task A2: Update `httpserver.NewRouter` to accept allowed origins

**Files:**
- Modify: `apps/api/internal/platform/httpserver/router.go`

- [ ] **Step 1: Modify router to accept and apply CORS**

Replace the contents of `apps/api/internal/platform/httpserver/router.go` with:

```go
package httpserver

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(log *slog.Logger, webOrigin string) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(structuredLogger(log))
	r.Use(corsMiddleware(webOrigin))
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}

func corsMiddleware(webOrigin string) func(http.Handler) http.Handler {
	allowed := strings.Split(webOrigin, ",")
	for i, s := range allowed {
		allowed[i] = strings.TrimSpace(s)
	}
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowed,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
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

- [ ] **Step 2: Run build, expect a single failure in `cmd/server`**

Run from `apps/api/`: `go build ./...`
Expected: build error on `cmd/server/casadana.go` — `not enough arguments in call to httpserver.NewRouter`. This is the next task.

---

### Task A3: Pass `cfg.WebOrigin` to the router from `cmd/server`

**Files:**
- Modify: `apps/api/cmd/server/casadana.go`

- [ ] **Step 1: Update the call site**

In `apps/api/cmd/server/casadana.go`, find the line:

```go
	r := httpserver.NewRouter(log)
```

Replace with:

```go
	r := httpserver.NewRouter(log, cfg.WebOrigin)
```

- [ ] **Step 2: Verify full build + vet + tests**

Run from `apps/api/`:

```bash
go build ./... && go vet ./... && go test ./...
```

Expected: build exit 0, vet exit 0, all tests PASS. (No existing test touches `NewRouter` directly — the http tests in `internal/booking/http_test.go` construct a fresh `chi.NewRouter()` and call `booking.Mount`, so they're unaffected.)

---

## Phase B — Author OpenAPI spec

### Task B1: Write `apps/api/openapi.yaml`

**Files:**
- Create: `apps/api/openapi.yaml`

- [ ] **Step 1: Author the spec**

Create `apps/api/openapi.yaml`:

```yaml
openapi: 3.0.3
info:
  title: Casa Dana API
  version: 0.1.0
  description: |
    Casa Dana villa rental API. Public endpoints for booking submission and
    availability lookup. Villa content (names, photos, descriptions) is static
    in the frontend and not exposed by this API.

servers:
  - url: http://localhost:8080
    description: local dev
  - url: https://casa-dana.com
    description: production

tags:
  - name: health
    description: Liveness
  - name: bookings
    description: Booking submission and availability

paths:
  /api/health:
    get:
      operationId: getHealth
      tags: [health]
      summary: Liveness probe
      responses:
        "200":
          description: Service is up
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/HealthResponse"

  /api/bookings:
    post:
      operationId: createBooking
      tags: [bookings]
      summary: Submit a booking request
      description: |
        Creates a `pending` booking and triggers a confirmation email to the
        guest plus a notification email to the admin.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateBookingRequest"
      responses:
        "201":
          description: Booking accepted
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/BookingResponse"
        "404":
          description: Villa slug not in allowlist
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "409":
          description: Dates conflict with an existing booking
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "422":
          description: Validation error
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

  /api/villas/{slug}/availability:
    get:
      operationId: getVillaAvailability
      tags: [bookings]
      summary: Get booked date ranges for a villa
      description: |
        Returns date ranges where the villa is already booked (approved or
        paid) within the [from, to) window. Pending bookings are NOT returned.
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
          description: Availability
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AvailabilityResponse"
        "404":
          description: Villa slug not in allowlist
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "422":
          description: Invalid query params
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

components:
  schemas:
    HealthResponse:
      type: object
      required: [status]
      properties:
        status:
          type: string
          enum: [ok]
          example: ok

    BookingStatus:
      type: string
      enum: [pending, approved, rejected, cancelled, paid]

    CreateBookingRequest:
      type: object
      required:
        - villa_slug
        - guest_name
        - guest_email
        - guest_phone
        - check_in
        - check_out
        - adults
      properties:
        villa_slug: { type: string, minLength: 1, maxLength: 64, example: casadana }
        guest_name: { type: string, minLength: 1, maxLength: 120, example: Jane Doe }
        guest_email: { type: string, format: email, maxLength: 255, example: jane@example.com }
        guest_phone: { type: string, minLength: 2, maxLength: 40, example: "+33123456789" }
        check_in:
          type: string
          format: date
          description: ISO date, YYYY-MM-DD
          example: "2026-07-04"
        check_out:
          type: string
          format: date
          description: ISO date, YYYY-MM-DD. Must be strictly after check_in.
          example: "2026-07-11"
        adults: { type: integer, minimum: 1, maximum: 20, example: 2 }
        children: { type: integer, minimum: 0, maximum: 20, default: 0, example: 0 }
        message: { type: string, maxLength: 2000, default: "", example: "" }

    BookingResponse:
      type: object
      required:
        - id
        - villa_slug
        - status
        - check_in
        - check_out
        - guest_name
        - guest_email
        - created_at
      properties:
        id: { type: string, format: uuid }
        villa_slug: { type: string }
        status: { $ref: "#/components/schemas/BookingStatus" }
        check_in: { type: string, format: date }
        check_out: { type: string, format: date }
        guest_name: { type: string }
        guest_email: { type: string, format: email }
        created_at: { type: string, format: date-time }

    BookedRange:
      type: object
      required: [check_in, check_out]
      properties:
        check_in: { type: string, format: date }
        check_out: { type: string, format: date }

    AvailabilityResponse:
      type: object
      required: [booked_ranges]
      properties:
        booked_ranges:
          type: array
          items: { $ref: "#/components/schemas/BookedRange" }

    ErrorResponse:
      type: object
      required: [error]
      properties:
        error:
          type: object
          required: [code, message]
          properties:
            code:
              type: string
              description: |
                Machine-readable error code. Known codes:
                `DATES_CONFLICT`, `UNKNOWN_VILLA`, `VALIDATION`,
                `INTERNAL`, `INVALID_STATUS`.
              example: DATES_CONFLICT
            message:
              type: string
              example: those dates are not available
```

- [ ] **Step 2: Lint the spec**

Verify the YAML is parseable. From the repo root run:

```bash
python3 -c "import yaml; yaml.safe_load(open('apps/api/openapi.yaml'))"
```

Expected: exit 0 (no output). If `python3` is unavailable, use any YAML validator (e.g., `bunx -y @stoplight/spectral-cli lint apps/api/openapi.yaml`).

---

## Phase C — `packages/api` workspace

### Task C1: Delete `packages/types`

**Files:**
- Delete: `packages/types/` (entire directory)
- Modify: `apps/web/package.json`

- [ ] **Step 1: Remove the dep from web**

In `apps/web/package.json`, remove the line:

```json
    "@casa-dana/types": "workspace:*",
```

- [ ] **Step 2: Remove the package directory**

Run from repo root:

```bash
rm -rf packages/types
```

- [ ] **Step 3: Refresh workspace**

Run from repo root:

```bash
bun install
```

Expected: install succeeds with no errors about missing `@casa-dana/types`.

---

### Task C2: Create `packages/api/package.json`

**Files:**
- Create: `packages/api/package.json`

- [ ] **Step 1: Write package manifest**

Create `packages/api/package.json`:

```json
{
  "name": "@casa-dana/api",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "exports": {
    ".": "./src/index.ts"
  },
  "scripts": {
    "generate": "orval --config orval.config.ts"
  },
  "dependencies": {
    "axios": "^1",
    "zod": "^3"
  },
  "peerDependencies": {
    "@tanstack/react-query": "^5",
    "react": "^19"
  },
  "devDependencies": {
    "orval": "^7",
    "typescript": "^5"
  }
}
```

- [ ] **Step 2: Install workspace**

Run from repo root: `bun install`
Expected: succeeds; `packages/api` now appears in the workspace.

---

### Task C3: Create `packages/api/tsconfig.json`

**Files:**
- Create: `packages/api/tsconfig.json`

- [ ] **Step 1: Write tsconfig**

Create `packages/api/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "jsx": "preserve",
    "strict": true,
    "noUnusedLocals": false,
    "noUnusedParameters": false,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "resolveJsonModule": true,
    "verbatimModuleSyntax": false,
    "allowSyntheticDefaultImports": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": false,
    "noEmit": true,
    "types": ["vite/client"]
  },
  "include": ["src"]
}
```

---

### Task C4: Write the custom axios client

**Files:**
- Create: `packages/api/src/client.ts`

- [ ] **Step 1: Write client**

Create `packages/api/src/client.ts`:

```ts
import Axios, { AxiosError, AxiosRequestConfig } from "axios"

export const AXIOS_INSTANCE = Axios.create({
  baseURL: (typeof import.meta !== "undefined" && (import.meta as any).env?.VITE_API_BASE_URL) || "",
  headers: { "Content-Type": "application/json" },
  withCredentials: true,
})

export type ApiErrorBody = { error: { code: string; message: string } }

export class ApiError extends Error {
  public code: string
  public status: number
  constructor(code: string, status: number, message: string) {
    super(message)
    this.name = "ApiError"
    this.code = code
    this.status = status
  }
  static fromAxios(e: AxiosError<ApiErrorBody>): ApiError {
    const status = e.response?.status ?? 0
    const body = e.response?.data?.error
    return new ApiError(body?.code ?? "NETWORK", status, body?.message ?? e.message)
  }
}

export const customAxios = <T>(config: AxiosRequestConfig): Promise<T> => {
  return AXIOS_INSTANCE({ ...config })
    .then((res) => res.data as T)
    .catch((e: AxiosError<ApiErrorBody>) => {
      throw ApiError.fromAxios(e)
    })
}
```

---

### Task C5: Write the Orval config

**Files:**
- Create: `packages/api/orval.config.ts`

- [ ] **Step 1: Write config**

Create `packages/api/orval.config.ts`:

```ts
import { defineConfig } from "orval"

export default defineConfig({
  casadana: {
    input: {
      target: "../../apps/api/openapi.yaml",
    },
    output: {
      mode: "tags-split",
      target: "src/generated/index.ts",
      schemas: "src/generated/schemas",
      client: "react-query",
      httpClient: "axios",
      indexFiles: true,
      prettier: false,
      override: {
        mutator: {
          path: "../client.ts",
          name: "customAxios",
        },
        useTypeOverInterface: true,
        query: {
          useQuery: true,
          useMutation: true,
          signal: true,
        },
      },
    },
  },
})
```

---

### Task C6: Generate the client

**Files:**
- Create (generated): `packages/api/src/generated/**`

- [ ] **Step 1: Run codegen**

Run from `packages/api/`:

```bash
bun run generate
```

Expected: `src/generated/` populated with at least `bookings.ts`, `health.ts`, `index.ts`, and a `schemas/` directory containing files for each schema.

- [ ] **Step 2: Inspect generated hook names**

Run from `packages/api/`:

```bash
grep -E "^export (function|const) use" src/generated/bookings.ts | head
```

Expected output includes hooks like `useCreateBooking` and `useGetVillaAvailability` (exact naming depends on Orval; if the names differ, note them — later tasks reference these exact names). If the names differ, **update the names in subsequent tasks before implementing**.

---

### Task C7: Write the package barrel

**Files:**
- Create: `packages/api/src/index.ts`

- [ ] **Step 1: Write barrel**

Create `packages/api/src/index.ts`:

```ts
export * from "./generated"
export * from "./generated/schemas"
export { customAxios, AXIOS_INSTANCE, ApiError } from "./client"
export type { ApiErrorBody } from "./client"
```

If `src/generated/index.ts` already re-exports schemas (depends on Orval's `indexFiles` behavior), the second line may produce a duplicate-export TS error — in that case remove the second line.

- [ ] **Step 2: Type-check the package**

Run from `packages/api/`:

```bash
bunx tsc --noEmit
```

Expected: exit 0.

---

## Phase D — React Query in `apps/web`

### Task D1: Install web dependencies

**Files:**
- Modify: `apps/web/package.json`

- [ ] **Step 1: Add the new deps**

Run from repo root:

```bash
bun add --cwd apps/web @tanstack/react-query @tanstack/react-query-devtools @casa-dana/api@workspace:*
```

- [ ] **Step 2: Verify install**

Run from repo root: `bun install`
Expected: exit 0. `apps/web/package.json` now lists `@tanstack/react-query`, `@tanstack/react-query-devtools`, and `@casa-dana/api`.

---

### Task D2: Create the QueryClient singleton

**Files:**
- Create: `apps/web/src/lib/query-client.ts`

- [ ] **Step 1: Write the file**

Create `apps/web/src/lib/query-client.ts`:

```ts
import { QueryClient } from "@tanstack/react-query"

import { ApiError } from "@casa-dana/api"

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      retry: (failureCount, err) => {
        if (err instanceof ApiError && err.status >= 400 && err.status < 500) return false
        return failureCount < 2
      },
      refetchOnWindowFocus: false,
    },
    mutations: { retry: false },
  },
})
```

---

### Task D3: Wrap the app in `QueryClientProvider`

**Files:**
- Modify: `apps/web/src/App.tsx`

- [ ] **Step 1: Replace App.tsx content**

Overwrite `apps/web/src/App.tsx` with:

```tsx
import { QueryClientProvider } from "@tanstack/react-query"
import { ReactQueryDevtools } from "@tanstack/react-query-devtools"
import { RouterProvider, createRouter } from "@tanstack/react-router"

import { queryClient } from "./lib/query-client"
import { routeTree } from "./routeTree.gen"

import "./globals.css"

const router = createRouter({ routeTree })

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}

export default App
```

---

### Task D4: Add dev env file

**Files:**
- Create: `apps/web/.env.development`

- [ ] **Step 1: Write env file**

Create `apps/web/.env.development`:

```
VITE_API_BASE_URL=http://localhost:8080
```

- [ ] **Step 2: Update gitignore if needed**

Inspect `apps/web/.gitignore` (or the repo root `.gitignore`) — `.env.development` should usually be committed (not secret). If `.env*` is broadly ignored, add an exception:

```
!.env.development
```

If `.env.development` is already not ignored, skip this step.

---

### Task D5: Verify the app still boots

**Files:** none

- [ ] **Step 1: Type-check + build**

Run from `apps/web/`:

```bash
bunx tsc --noEmit
```

Expected: exit 0. No compile errors. If there are errors involving `@casa-dana/api` imports, verify `packages/api/src/generated/` exists (Task C6) and `packages/api/src/index.ts` is correct (Task C7).

- [ ] **Step 2: Boot dev server (manual check)**

Run from repo root: `bun --filter @casa-dana/web dev`
Open `http://localhost:5173`.
Expected: app renders normally. Open browser DevTools — bottom-right shows the React Query DevTools button. Click it; no errors in console. Stop the dev server when verified.

---

## Phase E — Wire booking form

### Task E1: Thread `villaSlug` from page to booking component

**Files:**
- Modify: `apps/web/src/pages/villa-detail-page.tsx`
- Modify: `apps/web/src/components/sections/villa/villa-about.tsx` (if it forwards the booking prop to `VillaBooking`)
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Inspect how `VillaBooking` is rendered today**

Read `apps/web/src/components/sections/villa/villa-about.tsx` to confirm it's the parent of `VillaBooking` (`villa-detail-page.tsx:65` passes `booking` to `VillaAbout`). The thread will go:

```
villa-detail-page.tsx  ──┐ villaId
                         ▼
        VillaAbout (forwards prop)
                         ▼
        VillaBooking  (consumes villaSlug)
```

- [ ] **Step 2: Add `villaSlug` to VillaBooking props**

In `apps/web/src/components/sections/villa/villa-booking.tsx`, change the props interface:

```tsx
interface VillaBookingProps {
  villaSlug: string
  booking: VillaData["booking"]
}
```

And destructure in the component signature:

```tsx
export default function VillaBooking({ villaSlug, booking }: VillaBookingProps) {
```

- [ ] **Step 3: Forward through VillaAbout**

In `apps/web/src/components/sections/villa/villa-about.tsx`, find the `VillaBooking` usage and add `villaSlug` to its props interface and render. Specifically:

- Add `villaSlug: string` to the VillaAbout props interface
- Pass `villaSlug={villaSlug}` to `<VillaBooking ... />`

- [ ] **Step 4: Pass `villaId` from the page**

In `apps/web/src/pages/villa-detail-page.tsx`, update the `<VillaAbout ... />` call (currently at line 65) to:

```tsx
<VillaAbout villaSlug={villaId} about={villa.about} booking={villa.booking} />
```

- [ ] **Step 5: Type-check**

Run from `apps/web/`: `bunx tsc --noEmit`
Expected: exit 0. If errors, the prop interface in VillaAbout likely needs adjustment to match the new shape.

---

### Task E2: Add `name` field to the booking form

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Extend the form values type**

In `villa-booking.tsx`, update the `BookingFormValues` interface:

```tsx
interface BookingFormValues {
  name: string
  checkIn: Date
  checkOut: Date
  guests: number
  email: string
  tel: string
  description: string
}
```

- [ ] **Step 2: Add `name` to `defaultValues`**

In the `useForm` call, add `name: ""` to `defaultValues`:

```tsx
const { control, register, handleSubmit, watch, setValue, setError, formState } = useForm<BookingFormValues>({
  defaultValues: {
    name: "",
    checkIn: dateOnly(booking.defaultCheckIn),
    checkOut: dateOnly(booking.defaultCheckOut),
    guests: booking.defaultGuests,
    email: "",
    tel: "",
    description: "",
  },
})
```

Note: also pulling `setError` and `formState` into the destructure for later use.

- [ ] **Step 3: Add the name input above the email input**

In the form's input block (currently lines 309–347 of `villa-booking.tsx`, the `<div className="border-outline-variant mt-4 grid border border-b-0 bg-white">` section), prepend a new `<label>` for the name field:

```tsx
<label className="border-outline-variant block border-b px-4 py-3">
  <span className="text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase">
    {m.villa_booking_name_label()}
  </span>
  <Input
    type="text"
    autoComplete="name"
    placeholder={m.villa_booking_name_placeholder()}
    className={inputClassName}
    {...register("name", { required: true })}
  />
</label>
```

- [ ] **Step 4: Add the i18n keys**

In `apps/web/messages/en.json`, add:

```json
"villa_booking_name_label": "Full name",
"villa_booking_name_placeholder": "Your full name"
```

Add the same keys to `apps/web/messages/es.json` (`"Nombre completo"`, `"Tu nombre completo"`) and `apps/web/messages/fr.json` (`"Nom complet"`, `"Votre nom complet"`).

- [ ] **Step 5: Regenerate paraglide messages**

Run from `apps/web/`:

```bash
bun run paraglide
```

Expected: regenerates `apps/web/src/paraglide/messages.ts`. The `m.villa_booking_name_label` and `m.villa_booking_name_placeholder` accessors now exist.

- [ ] **Step 6: Type-check**

Run from `apps/web/`: `bunx tsc --noEmit`
Expected: exit 0.

---

### Task E3: Wire `useCreateBooking` mutation

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Import dependencies**

Add to the imports at the top of `villa-booking.tsx`:

```tsx
import { useQueryClient } from "@tanstack/react-query"
import { ApiError, useCreateBooking } from "@casa-dana/api"
import { format } from "date-fns"
```

(Note: the hook name shown is `useCreateBooking` — if Task C6 step 2 reported a different name from the actual Orval output, use that exact name.)

- [ ] **Step 2: Add mutation state**

Just below the `useForm` hook, add:

```tsx
const queryClient = useQueryClient()
const [submitted, setSubmitted] = useState(false)
const [topLevelError, setTopLevelError] = useState<string | null>(null)

const { mutate: createBooking, isPending } = useCreateBooking({
  mutation: {
    onSuccess: () => {
      setSubmitted(true)
      setTopLevelError(null)
      queryClient.invalidateQueries({ queryKey: ["/api/villas", villaSlug, "availability"] })
    },
    onError: (err) => {
      if (err instanceof ApiError) {
        if (err.code === "VALIDATION") {
          // Validation messages come in the form "field: tag; field: tag".
          // Best-effort: map common field names to react-hook-form fields.
          const fieldMap: Record<string, keyof BookingFormValues> = {
            GuestName: "name",
            GuestEmail: "email",
            GuestPhone: "tel",
            CheckIn: "checkIn",
            CheckOut: "checkOut",
            Adults: "guests",
          }
          for (const part of err.message.split(";")) {
            const [rawField, tag] = part.split(":").map((s) => s.trim())
            const field = fieldMap[rawField ?? ""]
            if (field) setError(field, { type: tag || "invalid", message: tag })
          }
          setTopLevelError(null)
        } else if (err.code === "DATES_CONFLICT") {
          setTopLevelError(m.villa_booking_error_dates_conflict())
        } else if (err.code === "UNKNOWN_VILLA") {
          setTopLevelError(m.villa_booking_error_unknown_villa())
        } else {
          setTopLevelError(m.villa_booking_error_generic())
        }
      } else {
        setTopLevelError(m.villa_booking_error_generic())
      }
    },
  },
})
```

- [ ] **Step 3: Replace the `onSubmit` body**

Find the existing `onSubmit` in `villa-booking.tsx`:

```tsx
const onSubmit = (values: BookingFormValues) => {
  console.log("booking request", values)
}
```

Replace with:

```tsx
const onSubmit = (values: BookingFormValues) => {
  createBooking({
    data: {
      villa_slug: villaSlug,
      guest_name: values.name,
      guest_email: values.email,
      guest_phone: values.tel,
      check_in: format(values.checkIn, "yyyy-MM-dd"),
      check_out: format(values.checkOut, "yyyy-MM-dd"),
      adults: values.guests,
      children: 0,
      message: values.description,
    },
  })
}
```

- [ ] **Step 4: Add the i18n keys for error messages**

In `apps/web/messages/en.json`, `es.json`, `fr.json`, add three keys each. For `en.json`:

```json
"villa_booking_error_dates_conflict": "Those dates were just taken. Please pick others.",
"villa_booking_error_unknown_villa": "We could not find this villa. Please refresh.",
"villa_booking_error_generic": "Something went wrong. Please try again.",
"villa_booking_success_title": "Thank you",
"villa_booking_success_body": "We received your request and will be in touch shortly."
```

Spanish (`es.json`):

```json
"villa_booking_error_dates_conflict": "Estas fechas acaban de reservarse. Elige otras, por favor.",
"villa_booking_error_unknown_villa": "No encontramos esta villa. Actualiza la página.",
"villa_booking_error_generic": "Algo salió mal. Inténtalo de nuevo.",
"villa_booking_success_title": "Gracias",
"villa_booking_success_body": "Recibimos tu solicitud y te contactaremos en breve."
```

French (`fr.json`):

```json
"villa_booking_error_dates_conflict": "Ces dates viennent d'être prises. Veuillez en choisir d'autres.",
"villa_booking_error_unknown_villa": "Villa introuvable. Veuillez actualiser la page.",
"villa_booking_error_generic": "Une erreur est survenue. Veuillez réessayer.",
"villa_booking_success_title": "Merci",
"villa_booking_success_body": "Nous avons bien reçu votre demande et reviendrons vers vous très bientôt."
```

Run `bun run paraglide` from `apps/web/` to regenerate the message accessors.

- [ ] **Step 5: Render UI states**

The submit button is currently at lines 349–355 of `villa-booking.tsx`. Replace its rendering with:

```tsx
{topLevelError && (
  <p className="border-error/30 bg-error-container/20 text-error mb-3 border px-3 py-2 text-[13px]">
    {topLevelError}
  </p>
)}

<Button
  type="submit"
  disabled={isPending}
  className="bg-primary text-on-primary hover:bg-primary-container disabled:opacity-60 mt-4 inline-flex h-auto w-full items-center justify-center gap-3 rounded-none px-6 py-[18px] font-mono text-[11px] tracking-[0.28em] uppercase"
>
  {isPending ? m.villa_booking_request_sending() : m.villa_booking_request_book()}
  {!isPending && <ArrowRight size={12} />}
</Button>
```

Add the new `villa_booking_request_sending` key to all three locale files:
- `en.json`: `"villa_booking_request_sending": "Sending..."`
- `es.json`: `"villa_booking_request_sending": "Enviando..."`
- `fr.json`: `"villa_booking_request_sending": "Envoi..."`

Run `bun run paraglide` again.

- [ ] **Step 6: Render success state**

At the very top of the component's return (just inside the `<aside ...>`, before the existing pricing header div), add a conditional that swaps the entire content when `submitted` is `true`:

```tsx
if (submitted) {
  return (
    <aside
      id="book"
      className="bg-background border-outline-variant editorial-shadow border p-6 md:sticky md:top-28 md:p-8"
    >
      <div className="text-center">
        <h3 className="font-display text-primary text-[28px] italic">
          {m.villa_booking_success_title()}
        </h3>
        <p className="text-on-surface-variant mt-3 text-[15px]">
          {m.villa_booking_success_body()}
        </p>
      </div>
    </aside>
  )
}
```

Place this `if (submitted)` block **after** the existing hooks (`useForm`, `useState`, etc.) but **before** the `return (...)` that renders the form. This is the standard pattern — early-return after hooks.

- [ ] **Step 7: Type-check**

Run from `apps/web/`: `bunx tsc --noEmit`
Expected: exit 0.

---

## Phase F — Wire availability query

### Task F1: Add the `useGetVillaAvailability` query

**Files:**
- Modify: `apps/web/src/components/sections/villa/villa-booking.tsx`

- [ ] **Step 1: Import the hook and date helpers**

Update the top imports in `villa-booking.tsx`:

```tsx
import { keepPreviousData, useQueryClient } from "@tanstack/react-query"
import { useGetVillaAvailability } from "@casa-dana/api"
import { addDays, addMonths, endOfMonth, format, parseISO, startOfMonth } from "date-fns"
```

(`useQueryClient` and `format` were already imported in Task E3 — merge the imports; do not duplicate.)

- [ ] **Step 2: Add the query**

Inside the component, after the existing `useState` and `useRef` calls but before the `cells` `useMemo`, add:

```tsx
const queryWindow = useMemo(() => {
  const from = startOfMonth(viewMonth)
  const to = endOfMonth(addMonths(viewMonth, 1))
  return { from: format(from, "yyyy-MM-dd"), to: format(to, "yyyy-MM-dd") }
}, [viewMonth])

const { data: availability } = useGetVillaAvailability(
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

- [ ] **Step 3: Build the blocked-nights set**

Just below the query, add:

```tsx
const blockedNights = useMemo(() => {
  const set = new Set<string>()
  for (const r of availability?.booked_ranges ?? []) {
    const start = parseISO(r.check_in)
    const end = parseISO(r.check_out)
    for (let d = start; d < end; d = addDays(d, 1)) {
      set.add(format(d, "yyyy-MM-dd"))
    }
  }
  return set
}, [availability])

const isBlocked = (date: Date) => blockedNights.has(format(date, "yyyy-MM-dd"))
```

- [ ] **Step 4: Guard `pickDate`**

Find the `pickDate` function in `villa-booking.tsx` (currently around line 123) and add a guard at the top:

```tsx
const pickDate = (date: Date) => {
  if (isBlocked(date)) return
  if (activeField === "in" || date < checkIn) {
    setValue("checkIn", date, { shouldDirty: true })
    if (checkOut <= date) {
      setValue("checkOut", new Date(date.getTime() + 7 * 86_400_000), { shouldDirty: true })
    }
    setActiveField("out")
  } else {
    setValue("checkOut", date, { shouldDirty: true })
    setActiveField(null)
  }
}
```

- [ ] **Step 5: Render blocked cells with strike-through**

Find the calendar cell render block (currently lines 241–270 of `villa-booking.tsx`). Replace the inner `cells.map((cell, i) => { ... })` body with:

```tsx
{cells.map((cell, i) => {
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
  const blocked = isBlocked(cell.date)
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
})}
```

- [ ] **Step 6: Type-check + build**

Run from `apps/web/`:

```bash
bunx tsc --noEmit
```

Expected: exit 0.

Run from `apps/web/`:

```bash
bun run build
```

Expected: vite build succeeds.

---

## Phase G — End-to-end smoke test

### Task G1: Boot the stack and submit a booking

**Files:** none

- [ ] **Step 1: Set up local env for the API**

If not already present, create `/Users/loancleris/Desktop/projet-perso/casadana/.env.dev` (or update existing) with at minimum:

```
POSTGRES_USER=casadana
POSTGRES_PASSWORD=changeme
POSTGRES_DB=casadana
POSTGRES_HOST=postgres
POSTGRES_PORT=5432

PORT=8080
JWT_SECRET=devsecret
RESEND_API_KEY=re_devplaceholder
MAIL_FROM=noreply@casa-dana.local
ADMIN_NOTIFY_EMAIL=admin@casa-dana.local
WEB_ORIGIN=http://localhost:5173
MIGRATE_ON_BOOT=true
```

- [ ] **Step 2: Start postgres + api**

From repo root:

```bash
docker compose -f .docker/docker-compose.yml --env-file .env.dev --profile dev up -d postgres api
```

Wait until both healthchecks pass: `docker compose -f .docker/docker-compose.yml ps`.

- [ ] **Step 3: Verify health and CORS preflight**

```bash
curl -i http://localhost:8080/api/health
```

Expected: HTTP 200 with `{"status":"ok"}`.

```bash
curl -i -X OPTIONS http://localhost:8080/api/bookings \
  -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type"
```

Expected: HTTP 200 or 204 with headers including:
- `Access-Control-Allow-Origin: http://localhost:5173`
- `Access-Control-Allow-Methods: GET, POST, ...`
- `Access-Control-Allow-Credentials: true`

If `Access-Control-Allow-Origin` is missing or `*`, the CORS middleware is misconfigured — recheck Task A2.

- [ ] **Step 4: Start the web dev server**

In a separate terminal:

```bash
bun --filter @casa-dana/web dev
```

Open http://localhost:5173. Navigate to a villa detail page (e.g. http://localhost:5173/villa/casadana).

- [ ] **Step 5: Submit a booking through the UI**

- Open the calendar; verify cells render with no errors.
- Pick a check-in / check-out range.
- Fill in: name, email, phone, message.
- Click "Request to Book".
- Expected: button shows "Sending..." briefly; on success, the aside swaps to the "Thank you" success card.
- Open browser DevTools → Network. The POST to `/api/bookings` returned 201. The body contains `{"id":"...","status":"pending",...}`.

- [ ] **Step 6: Verify availability query fires**

- Reopen the villa page (or click the calendar again to retrigger). DevTools → Network → there should be a `GET /api/villas/casadana/availability?from=...&to=...` request.
- The booking just submitted is `pending`, so it does NOT appear in `booked_ranges` (the SQL filters to `approved`/`paid`). This is expected. To verify strike-through visually, connect to the DB and manually update the booking to `approved`:

```bash
docker exec -it casadana-postgres psql -U casadana -d casadana \
  -c "UPDATE bookings SET status='approved' WHERE status='pending';"
```

Refresh the villa page, reopen the calendar. Expected: the booked dates appear with strike-through, the checkout day is still selectable, and clicks on struck-through cells do nothing.

- [ ] **Step 7: Verify dates-conflict UI**

Try to submit another booking with overlapping dates with the now-approved booking.
Expected: form stays open, an inline red error reads "Those dates were just taken. Please pick others."

- [ ] **Step 8: Stop the stack**

```bash
docker compose -f .docker/docker-compose.yml --env-file .env.dev --profile dev down
```

---

## Post-flight

- [ ] **Files touched (sanity list):**
  - Created:
    - `apps/api/openapi.yaml`
    - `packages/api/package.json`, `tsconfig.json`, `orval.config.ts`
    - `packages/api/src/client.ts`, `index.ts`
    - `packages/api/src/generated/**` (Orval output, committed)
    - `apps/web/src/lib/query-client.ts`
    - `apps/web/.env.development`
  - Modified:
    - `apps/api/internal/platform/httpserver/router.go`
    - `apps/api/cmd/server/casadana.go`
    - `apps/api/go.mod`, `go.sum`
    - `apps/web/package.json`
    - `apps/web/src/App.tsx`
    - `apps/web/src/pages/villa-detail-page.tsx`
    - `apps/web/src/components/sections/villa/villa-about.tsx`
    - `apps/web/src/components/sections/villa/villa-booking.tsx`
    - `apps/web/messages/{en,es,fr}.json`
    - `apps/web/src/paraglide/messages.ts` (regenerated)
  - Deleted:
    - `packages/types/` (entire directory)

- [ ] **Things deferred to later plans:**
  - Auth wiring (Plan 2) — refresh-cookie support is already enabled via `AllowCredentials: true` and `withCredentials: true`, but no auth headers are sent yet.
  - Reviews module (Plan 3) — its endpoints will be added to `openapi.yaml` and `bun run generate` will produce a `reviews.ts` hook file automatically.
  - `boneyard-js` skeleton screens.
  - Rate limits, CORS prod hardening, prod Dockerfile (Plan 4).
