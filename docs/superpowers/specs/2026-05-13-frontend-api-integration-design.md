# Casa Dana — Frontend ↔ Backend Integration Design

**Date:** 2026-05-13
**Status:** Approved — ready for implementation plan

## 1. Purpose & Scope

Wire the React frontend (`apps/web`) to the Go backend (`apps/api`) using a typed, OpenAPI-driven client with React Query hooks. The end state: the villa booking form submits real bookings, and the calendar visualizes booked dates from the server.

**In scope:**

- Author `apps/api/openapi.yaml` (hand-written) describing the three currently-running endpoints: `GET /api/health`, `POST /api/bookings`, `GET /api/villas/{slug}/availability`.
- Create a new workspace package `packages/api` (replacing the unused `packages/types`).
- Configure Orval to generate React Query hooks + zod schemas into `packages/api/src/generated/`.
- Add CORS middleware to the Go backend (pulled forward from Plan 4).
- Install and wire `@tanstack/react-query` (provider + DevTools) in `apps/web`.
- Replace the booking form's `console.log` with `useCreateBooking`.
- Wire the calendar in `villa-booking.tsx` to `useGetVillaAvailability`, striking through occupied nights and leaving the checkout day selectable.
- Map backend error codes (`DATES_CONFLICT`, `VALIDATION`, etc.) to user-facing UI states.

**Out of scope:**

- Auth + admin endpoints (Plan 2).
- Reviews endpoints (Plan 3).
- Rate limiting, graceful-shutdown hardening, prod Dockerfile (Plan 4).
- Backend villa CRUD (intentional — villa content stays static in the frontend).
- Splitting the `guests` field into adults + children (MVP sends `adults: guests, children: 0`).

## 2. Architecture Decisions

### 2.1 Codegen tool: Orval only

**Decision:** drop the unused `openapi-zod-client` setup. Orval handles everything we need from a single config: typed TypeScript, zod schemas (`zod: true`), React Query hooks (`client: 'react-query'`).

**Why not openapi-zod-client + manual hooks?** Two tools, two configs, manual glue. No upside.

**Why not hand-roll?** The 3-endpoint surface today grows to ~15-20 across Plans 2-3. Keeping types in sync manually with the server is annual-bug-tax territory.

### 2.2 OpenAPI spec: hand-written

**Decision:** `apps/api/openapi.yaml` is authored and maintained by hand.

**Why not `swaggo/swag` annotations?** Annotations get verbose for nested types and the generated YAML is less readable than what a human would write. Spec is small enough that hand-authoring pays for itself in clarity.

**Why not spec-first Go generation (`ogen`, `oapi-codegen`)?** Would mean replacing the hand-written chi handlers built in Plan 1 with generated stubs. Big refactor, low return.

**No sqlc-style "extract from code" tool exists** for HTTP handlers because Go's `net/http` handler doesn't carry compile-time type info for the wire shape — JSON decoding is dynamic, URL params are strings. The closest options are annotation-based (swaggo) or framework-tied (huma). Neither is worth it here.

### 2.3 Generated code lives in `packages/api`, not `apps/web`

**Decision:** new workspace package at `packages/api`, name `@casa-dana/api`. Delete `packages/types/` (currently empty placeholder). `apps/web` imports as `@casa-dana/api`.

**Why a separate package?** Aligns with the existing monorepo intent (`packages/*` workspace). Allows a future Node script (seed scripts, CLI) or admin-app workspace to consume the same client.

**Caveat:** Orval emits React Query hooks, which makes the package React-coupled. Only React consumers can use the generated hooks. The schemas + types portion is framework-neutral and reusable.

### 2.4 Generated code is committed

**Decision:** `packages/api/src/generated/` is committed to git.

**Why?** PRs show API contract changes alongside the YAML and backend changes. Reviewers see the diff. Local dev doesn't require running codegen to inspect types. CI runs `bun --filter @casa-dana/api generate` and `git diff --exit-code` to verify the committed output matches the spec.

### 2.5 CORS: prod is same-origin, dev needs allowlist

**Decision:** `WEB_ORIGIN` env var is a comma-separated allowlist of allowed origins, plumbed into the existing `httpserver.NewRouter` middleware stack via `github.com/go-chi/cors`.

**Why prod doesn't need CORS:** the Caddyfile reverse-proxies `/api/*` → `api:8080` and `/*` → `web:80` on the same `casa-dana.com` origin. Browser sees one origin → no preflight.

**Why dev needs it:** Vite dev server (`http://localhost:5173`) hits the API (`http://localhost:8080`). Different origins → CORS preflight required.

**Concrete values:**
- Dev: `WEB_ORIGIN=http://localhost:5173`
- Prod: `WEB_ORIGIN=https://casa-dana.com` (defensive — won't fire in same-origin flow, but blocks any unexpected cross-origin call)
- **Never** `*`.

**Frontend base URL:** `VITE_API_BASE_URL` env var.
- Dev: `VITE_API_BASE_URL=http://localhost:8080`
- Prod: `VITE_API_BASE_URL=""` (empty → relative URLs hit the same origin)

## 3. File Layout

```
casadana/
├── apps/
│   ├── api/
│   │   ├── openapi.yaml                        # ← NEW: hand-authored OpenAPI 3.0.3 spec
│   │   └── internal/platform/httpserver/
│   │       └── router.go                       # ← MODIFY: add CORS middleware
│   └── web/
│       ├── package.json                        # ← MODIFY: deps
│       ├── src/
│       │   ├── App.tsx                         # ← MODIFY: wrap with QueryClientProvider
│       │   ├── lib/
│       │   │   └── query-client.ts             # ← NEW: singleton QueryClient
│       │   ├── components/sections/villa/
│       │   │   └── villa-booking.tsx           # ← MODIFY: wire mutation + availability
│       │   └── pages/
│       │       └── villa-detail-page.tsx       # ← MODIFY: pass villaSlug down
│       └── .env.development                    # ← NEW: VITE_API_BASE_URL=http://localhost:8080
│
├── packages/
│   ├── types/                                  # ← DELETE
│   └── api/                                    # ← NEW workspace package
│       ├── package.json                        # name: "@casa-dana/api"
│       ├── tsconfig.json
│       ├── orval.config.ts                     # codegen config
│       └── src/
│           ├── index.ts                        # barrel re-export
│           ├── client.ts                       # axios instance + ApiError type
│           └── generated/                      # ← COMMITTED, regenerated by orval
│               ├── bookings.ts                 # useCreateBooking, useGetVillaAvailability
│               ├── health.ts                   # useGetHealth (rare use)
│               ├── schemas/                    # one TS file per OpenAPI schema
│               └── index.ts
│
└── .gitignore                                  # ← MODIFY: ensure node_modules/, etc.
```

**Removed from `apps/web/package.json`:** `"@casa-dana/types": "workspace:*"`.
**Added to `apps/web/package.json`:**
- `"@casa-dana/api": "workspace:*"`
- `"@tanstack/react-query": "^5"`
- `"@tanstack/react-query-devtools": "^5"`
- `"axios": "^1"` (Orval default HTTP client; or pivot to native fetch — see §6)

## 4. OpenAPI Spec

`apps/api/openapi.yaml`, OpenAPI 3.0.3. Components/operations mirror the implemented endpoints exactly.

### 4.1 Paths

| Path | Method | OperationId | Tag |
|---|---|---|---|
| `/api/health` | GET | `getHealth` | health |
| `/api/bookings` | POST | `createBooking` | bookings |
| `/api/villas/{slug}/availability` | GET | `getVillaAvailability` | bookings |

### 4.2 Schemas

- `HealthResponse` — `{ status: "ok" }`
- `CreateBookingRequest` — `{ villa_slug, guest_name, guest_email, guest_phone, check_in (date), check_out (date), adults (int >=1), children (int >=0, default 0), message (default "") }`
- `BookingResponse` — `{ id (uuid), villa_slug, status (enum), check_in, check_out, guest_name, guest_email, created_at (date-time) }`
- `AvailabilityResponse` — `{ booked_ranges: [{ check_in (date), check_out (date) }] }`
- `ErrorResponse` — `{ error: { code: string, message: string } }`
- `BookingStatus` (enum) — `pending | approved | rejected | cancelled | paid`

### 4.3 Error responses

Every endpoint declares:
- `400` — invalid JSON (rare; only when body parse fails before validator)
- `404` — `UNKNOWN_VILLA` (availability + booking when slug unknown)
- `409` — `DATES_CONFLICT` (booking only)
- `422` — `VALIDATION` (booking + availability with bad query)
- `500` — `INTERNAL`

All use the `ErrorResponse` schema.

### 4.4 Servers

```yaml
servers:
  - url: http://localhost:8080
    description: local dev
  - url: https://casa-dana.com
    description: prod
```

Orval generates against the spec, but the runtime base URL comes from `VITE_API_BASE_URL` via the custom axios client (§6).

## 5. Orval Configuration

`packages/api/orval.config.ts`:

```ts
import { defineConfig } from "orval"

export default defineConfig({
  casadana: {
    input: { target: "../../apps/api/openapi.yaml" },
    output: {
      mode: "tags-split",
      target: "src/generated/index.ts",
      schemas: "src/generated/schemas",
      client: "react-query",
      httpClient: "axios",
      override: {
        mutator: { path: "src/client.ts", name: "customAxios" },
        useTypeOverInterface: true,
        query: {
          useQuery: true,
          useMutation: true,
        },
      },
    },
  },
})
```

**Key choices:**
- `mode: 'tags-split'` — one file per OpenAPI tag (`bookings.ts`, `health.ts`). Easier diffs.
- `schemas: 'src/generated/schemas'` — one TS file per schema, importable individually.
- `httpClient: 'axios'` — Orval's most-supported client.
- `mutator` — every generated function calls our custom axios instance (§6) instead of inlining one. This is how we plumb `VITE_API_BASE_URL`, error mapping, and (later) auth headers.

**`packages/api/package.json`:**

```json
{
  "name": "@casa-dana/api",
  "private": true,
  "main": "./src/index.ts",
  "types": "./src/index.ts",
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

**`packages/api/src/index.ts`** (barrel):

```ts
export * from "./generated"
export { customAxios, ApiError } from "./client"
```

## 6. Custom axios client

`packages/api/src/client.ts`:

```ts
import Axios, { AxiosError, AxiosRequestConfig } from "axios"

export const AXIOS_INSTANCE = Axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? "",
  headers: { "Content-Type": "application/json" },
})

export type ApiErrorBody = { error: { code: string; message: string } }

export class ApiError extends Error {
  constructor(
    public code: string,
    public status: number,
    message: string,
  ) {
    super(message)
    this.name = "ApiError"
  }
  static fromAxios(e: AxiosError<ApiErrorBody>): ApiError {
    const status = e.response?.status ?? 0
    const body = e.response?.data?.error
    return new ApiError(body?.code ?? "NETWORK", status, body?.message ?? e.message)
  }
}

export const customAxios = <T>(config: AxiosRequestConfig): Promise<T> => {
  return AXIOS_INSTANCE({ ...config }).then((res) => res.data as T).catch((e: AxiosError<ApiErrorBody>) => {
    throw ApiError.fromAxios(e)
  })
}
```

Every generated query/mutation throws `ApiError` on failure. React Query exposes it via `query.error` / `mutation.error` typed as `ApiError | null` once we set the `<QueryClient>` default `mutationKeyHashFn` typing (handled by `QueryCache.defaultOptions`).

## 7. React Query setup

`apps/web/src/lib/query-client.ts`:

```ts
import { QueryClient } from "@tanstack/react-query"

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      retry: (failureCount, err: any) => {
        // Don't retry 4xx
        if (err?.status >= 400 && err?.status < 500) return false
        return failureCount < 2
      },
      refetchOnWindowFocus: false,
    },
    mutations: { retry: false },
  },
})
```

`apps/web/src/App.tsx` (revised): wrap existing router with `QueryClientProvider`:

```tsx
import { QueryClientProvider } from "@tanstack/react-query"
import { ReactQueryDevtools } from "@tanstack/react-query-devtools"
import { RouterProvider } from "@tanstack/react-router"
import { queryClient } from "./lib/query-client"
import { router } from "./routes"

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}
```

(The exact existing App structure determines how to splice these in — implementation plan will handle.)

## 8. Wiring the booking form

### 8.1 Props change

`villa-booking.tsx` currently takes `{ booking: VillaData["booking"] }`. Add `villaSlug: string`. Propagate from `villa-detail-page.tsx` (which already has the route param via TanStack Router).

### 8.2 Mutation

```tsx
import { useCreateBooking } from "@casa-dana/api"

const { mutate: createBooking, isPending, error } = useCreateBooking({
  mutation: {
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["/api/villas", villaSlug, "availability"],
      })
      setSubmitted(true)
    },
    onError: (err) => {
      // ApiError typing
      if (err.code === "VALIDATION") {
        // parse messages, setError on react-hook-form fields
      }
    },
  },
})

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

**Form change:** the current form has no `guest_name` field — the API requires one. The implementation plan adds a name input above the email row. `BookingFormValues` gains a `name: string` field with `register("name", { required: true })`.

### 8.3 UI states

- `isPending`: submit button shows spinner + "Sending..."
- `error?.code === "DATES_CONFLICT"`: inline message on the date row, "These dates were just taken. Please pick others."
- `error?.code === "VALIDATION"`: `setError` on each invalid field via `react-hook-form`
- Generic error: banner at top of form
- Success: replace form with a "Thank you — we'll be in touch" confirmation card

## 9. Wiring the calendar

### 9.1 Availability query

When the calendar opens (`activeField !== null`), or when `viewMonth` changes, query availability for a 2-month window centered on the visible month:

```tsx
import { useGetVillaAvailability } from "@casa-dana/api"

const from = startOfMonth(viewMonth)
const to = endOfMonth(addMonths(viewMonth, 1))

const { data: availability } = useGetVillaAvailability(
  villaSlug,
  { from: format(from, "yyyy-MM-dd"), to: format(to, "yyyy-MM-dd") },
  {
    query: {
      enabled: activeField !== null,
      // keep previous data while month changes for less flicker
      placeholderData: (prev) => prev,
    },
  },
)
```

### 9.2 Occupied-night set

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

### 9.3 Cell rendering

Modify the existing `cells.map(...)` render in `villa-booking.tsx`. For each `cell.date`:

- If `isBlocked(cell.date)`: render with `text-on-surface-variant/40 line-through cursor-not-allowed`, no `onClick`.
- Otherwise: existing behavior (selectable, range highlight, etc.).

`pickDate(date)` early-returns if `isBlocked(date)`.

### 9.4 Semantic correctness

This matches the backend's half-open interval:
- Booking `[2026-07-04, 2026-07-11)` → blocked nights: 4, 5, 6, 7, 8, 9, 10.
- 2026-07-11 (checkout day) is selectable as a new check-in. Sat→Sat handoff works.
- Conservative simplification: the check-IN day of an existing booking is also struck through, even though technically a new booking *could* check out that day without conflict. Accepted for MVP UX clarity.

## 10. Backend CORS

`apps/api/internal/platform/httpserver/router.go` — add CORS middleware before existing middleware stack:

```go
import "github.com/go-chi/cors"

func NewRouter(log *slog.Logger, webOrigin string) chi.Router {
    r := chi.NewRouter()

    allowed := strings.Split(webOrigin, ",")
    for i, s := range allowed { allowed[i] = strings.TrimSpace(s) }
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   allowed,
        AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
        ExposedHeaders:   []string{"Link"},
        AllowCredentials: true, // needed for Plan 2 refresh-cookie flow
        MaxAge:           300,
    }))

    // ...rest of middleware unchanged
}
```

`cmd/server/casadana.go`: pass `cfg.WebOrigin` to `httpserver.NewRouter(log, cfg.WebOrigin)`.

`config.Config.WebOrigin` already exists and is required (see Plan 1 spec §9).

## 11. Build & CI hooks

`packages/api/package.json` script: `"generate": "orval --config orval.config.ts"`.

`apps/web/package.json`:
- Add `"prebuild": "bun --filter @casa-dana/api generate"` so production builds always regenerate (paranoid but cheap).
- Dev workflow: `bun --filter @casa-dana/api generate` after editing `openapi.yaml`.

CI (future):
- Step 1: `bun --filter @casa-dana/api generate`
- Step 2: `git diff --exit-code packages/api/src/generated` (fails the build if committed output is out of sync)

## 12. Out-of-Scope, Confirmed

- Auth (Plan 2). The axios mutator is structured so an auth-header interceptor slots in cleanly later.
- Reviews endpoint (Plan 3).
- Form field redesign beyond adding `guest_name`.
- A separate `adults` + `children` form UI (sending `adults: guests, children: 0` for MVP).
- Server-side i18n of error messages.
- Optimistic updates on booking creation (would need rollback logic on `DATES_CONFLICT` — leave for later).
- WebSocket / SSE live availability updates.
- `boneyard-js` (skeleton-screen library). Defer to Plan 2/3 when actual list/table UIs (reviews list, admin bookings table) need them. Current scope only has a button-level loading state and a near-instant calendar fetch — both handled with a single boolean, no library needed.
