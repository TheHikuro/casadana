# Admin Auth + Reservations — Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Commit policy:** DO NOT run `git commit` during execution of this plan. Make file changes only. The user runs commits manually at their preferred boundaries. (Memory: `feedback-no-auto-commit`.)

**Goal:** Build the admin login page and the authenticated admin shell (sidebar, property switcher) with a fully working Reservations view, wired to the backend from `2026-07-02-admin-auth-and-reservations-backend.md` via the Orval-generated TanStack Query hooks.

**Architecture:** Split the root layout so `/admin` gets its own chrome instead of the public `Navbar`/`Footer`. A pathless `_authed` layout under `/admin` gate-checks the session (via `GET /api/admin/me`) in `beforeLoad` and renders the sidebar shell; `/admin/login` sits outside that gate. All data flows through the existing `@casa-dana/api` Orval hooks — no new fetch/axios code.

**Tech Stack:** React 19, TanStack Router (file-based routing), TanStack Query, react-hook-form, `@base-ui/react` primitives (already a dependency — `Dialog` is a new subpath import, not a new package), Tailwind v4 with this repo's MD3-style token set.

**Dependency:** Requires `2026-07-02-admin-auth-and-reservations-backend.md` to be implemented and its Task E2 (Orval regen) run first — this plan consumes `useAdminLogin`, `useAdminMe`, `useAdminLogout`, `useListAdminUsers`, `useCreateAdminUser`, `useDeleteAdminUser` from `packages/api/src/generated/admin/admin.ts`, and the updated `useListBookings`/`useCreateBooking`/`usePatchBooking`/`useDeleteBooking` (now with `villa_slug`/`source` support) from `packages/api/src/generated/bookings/bookings.ts`.

**Spec reference:** `docs/superpowers/specs/2026-07-02-admin-auth-and-reservations-design.md`

**Working dir for all paths below:** `apps/web/` unless stated otherwise.

## Global Constraints

- No `git commit` during execution — file changes only.
- No `any` types; use `Array<Type>` not `Type[]`; function components only; handler consts, not nested ternaries/ifs (per `CONTRIBUTING.md`).
- No new npm dependencies beyond what's already installed — `@base-ui/react/dialog` is a subpath of the already-installed `@base-ui/react` package (confirmed present in `node_modules`); `zod` is deliberately avoided for route search validation (a plain validator function is used instead) since it isn't currently a direct `apps/web` dependency.
- Two of the MD3 theme tokens the rest of `apps/web` sometimes references in copied shadcn boilerplate (`bg-destructive`, `text-muted-foreground`, etc.) are **not actually defined** in `globals.css`'s `@theme` block — only the MD3 set is (`primary`, `secondary`, `tertiary`, `error`, `surface*`, `on-*`, `outline*`). This plan only uses tokens confirmed present in `globals.css`. This is a pre-existing gap in the repo's UI kit, not something this plan fixes.
- No automated frontend tests are introduced — none exist in this repo today (confirmed absence of vitest/jest and any `*.test.ts*` file under `apps/web`). Verification is `tsc`/build + manual exercise against the dev server.
- Internal admin tool — no i18n/paraglide wiring for its copy (unlike the public site).

---

## Phase A — Root layout split

### Task A1: Add router context (`queryClient`) so route `beforeLoad` can read it

**Files:**
- Modify: `apps/web/src/routes/__root.tsx`
- Modify: `apps/web/src/App.tsx`

**Interfaces:**
- Produces: `RouterContext { queryClient: QueryClient }`, available as `context.queryClient` in every route's `beforeLoad`/`loader` — consumed by Task C2 (`_authed` layout) and Task C1 (login page).

- [ ] **Step 1: Give the root route a typed context**

Replace the full contents of `src/routes/__root.tsx`:

```tsx
import type { QueryClient } from "@tanstack/react-query"
import { Outlet, createRootRouteWithContext } from "@tanstack/react-router"

export interface RouterContext {
  queryClient: QueryClient
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => <Outlet />,
})
```

This removes the `Navbar`/`Footer` from the root — they move to `_public.tsx` in Task A2, so the public site's layout is unaffected once that task lands (doing this task alone temporarily breaks the public site's chrome; that's fine, Task A2 fixes it immediately after).

- [ ] **Step 2: Pass the context into the router**

In `src/App.tsx`, change:

```tsx
const router = createRouter({ routeTree })
```

to:

```tsx
const router = createRouter({ routeTree, context: { queryClient } })

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}
```

(`queryClient` is already imported at the top of this file from `./lib/query-client`.)

- [ ] **Step 3: Confirm it doesn't type-check yet (expected)**

Run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: FAILS — `routes/index.tsx` and `routes/villa/$villaId.tsx` still register at the old root without the new context requirement being an issue (this part is fine), but the app currently renders with no `Navbar`/`Footer` at all. This is a visual regression, not a type error, so `tsc` may actually pass here — if so, skip ahead; the real check is manual (open the dev server, confirm the public site currently has no nav/footer) and is fixed by Task A2.

---

### Task A2: Move public routes under a `_public` pathless layout

**Files:**
- Create: `apps/web/src/routes/_public.tsx`
- Create: `apps/web/src/routes/_public/index.tsx` (moved from `routes/index.tsx`)
- Create: `apps/web/src/routes/_public/villa/$villaId.tsx` (moved from `routes/villa/$villaId.tsx`)
- Delete: `apps/web/src/routes/index.tsx`
- Delete: `apps/web/src/routes/villa/$villaId.tsx`

- [ ] **Step 1: Create the pathless layout**

`src/routes/_public.tsx`:

```tsx
import { Outlet, createFileRoute } from "@tanstack/react-router"

import Footer from "@/components/footer/footer"
import Navbar from "@/components/header/navbar"

export const Route = createFileRoute("/_public")({
  component: PublicLayout,
})

function PublicLayout() {
  return (
    <div className="flex min-h-screen flex-col">
      <Navbar />
      <main className="grow">
        <Outlet />
      </main>
      <Footer />
    </div>
  )
}
```

- [ ] **Step 2: Move the home route**

Create `src/routes/_public/index.tsx` with the exact content of the current `src/routes/index.tsx`, only changing the path string:

```tsx
import { createFileRoute } from "@tanstack/react-router"

import HomePage from "@/pages/home-page"

export const Route = createFileRoute("/_public/")({
  component: HomePage,
})
```

Delete the old `src/routes/index.tsx`.

- [ ] **Step 3: Move the villa detail route**

Create `src/routes/_public/villa/$villaId.tsx` with the exact content of the current `src/routes/villa/$villaId.tsx`, only changing the path string:

```tsx
import { createFileRoute } from "@tanstack/react-router"

import VillaDetailPage from "@/pages/villa-detail-page"

export const Route = createFileRoute("/_public/villa/$villaId")({
  component: VillaDetailPage,
})
```

Delete the old `src/routes/villa/$villaId.tsx` (and the now-empty `src/routes/villa/` directory).

- [ ] **Step 4: Regenerate the route tree and verify**

Run: `bun --filter @casa-dana/web dev` (start it, or if already running, save any file to trigger the router plugin) — this regenerates `routeTree.gen.ts` automatically. Watch the terminal for router-plugin errors (a common one: the path string inside `createFileRoute(...)` not matching the file's location exactly — if so, fix the string to match, per Steps 2/3 above, and re-save).

Then run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: no errors.

Open `http://localhost:5173/` in a browser (or use the `mcp__plugin_playwright_playwright__browser_navigate` tool if running headless verification) and confirm the home page renders with `Navbar`/`Footer` exactly as before this task. Visit `/villa/<any-existing-slug>` and confirm the same.

---

## Phase B — Shared UI primitives

### Task B1: `Toast` primitive

**Files:**
- Create: `apps/web/src/components/ui/toast.tsx`

**Interfaces:**
- Produces: `ToastProvider`, `useToast() -> { toast: (message: string) => void }` — consumed by Task D2 (`ReservationTable`), Task D3 (`AddReservationDialog`), Task C1 (login error, optional).

- [ ] **Step 1: Write the component**

```tsx
import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react"

import { cn } from "@/lib/utils"

interface ToastContextValue {
  toast: (message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

const TOAST_DURATION_MS = 1800

export function ToastProvider({ children }: { children: ReactNode }) {
  const [message, setMessage] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const toast = useCallback((next: string) => {
    setMessage(next)
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setMessage(null), TOAST_DURATION_MS)
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div
        role="status"
        aria-live="polite"
        className={cn(
          "fixed right-6 bottom-6 z-[300] rounded-lg bg-inverse-surface px-4 py-2.5 text-sm text-inverse-on-surface shadow-editorial transition-all duration-200",
          message ? "translate-y-0 opacity-100" : "pointer-events-none translate-y-2 opacity-0",
        )}
      >
        {message}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) {
    throw new Error("useToast must be used within a ToastProvider")
  }
  return ctx
}
```

- [ ] **Step 2: Type-check**

Run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: no new errors (this file isn't imported anywhere yet, so it should just compile standalone).

---

### Task B2: `Dialog` primitive

**Files:**
- Create: `apps/web/src/components/ui/dialog.tsx`

**Interfaces:**
- Produces: `Dialog`, `DialogTrigger`, `DialogClose`, `DialogContent`, `DialogTitle` — consumed by Task D3 (`AddReservationDialog`).

- [ ] **Step 1: Write the component**

Follows the exact same wrapper convention as the existing `src/components/ui/select.tsx` (Portal/Popup around a `@base-ui/react/*` primitive family — `@base-ui/react/dialog` is already present in `node_modules` as part of the already-installed `@base-ui/react` package, confirmed via `ls node_modules/@base-ui/react/dialog`):

```tsx
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog"

import { cn } from "@/lib/utils"

const Dialog = DialogPrimitive.Root
const DialogTrigger = DialogPrimitive.Trigger
const DialogClose = DialogPrimitive.Close

function DialogPortal({ ...props }: DialogPrimitive.Portal.Props) {
  return <DialogPrimitive.Portal data-slot="dialog-portal" {...props} />
}

function DialogBackdrop({ className, ...props }: DialogPrimitive.Backdrop.Props) {
  return (
    <DialogPrimitive.Backdrop
      data-slot="dialog-backdrop"
      className={cn(
        "fixed inset-0 z-[150] bg-inverse-surface/40 data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0",
        className,
      )}
      {...props}
    />
  )
}

function DialogContent({ className, children, ...props }: DialogPrimitive.Popup.Props) {
  return (
    <DialogPortal>
      <DialogBackdrop />
      <DialogPrimitive.Popup
        data-slot="dialog-content"
        className={cn(
          "fixed top-1/2 left-1/2 z-[150] w-full max-w-[480px] -translate-x-1/2 -translate-y-1/2 rounded-xl bg-surface p-6 shadow-editorial",
          className,
        )}
        {...props}
      >
        {children}
      </DialogPrimitive.Popup>
    </DialogPortal>
  )
}

function DialogTitle({ className, ...props }: DialogPrimitive.Title.Props) {
  return (
    <DialogPrimitive.Title
      data-slot="dialog-title"
      className={cn("mb-4 text-base font-semibold text-on-surface", className)}
      {...props}
    />
  )
}

export { Dialog, DialogTrigger, DialogClose, DialogPortal, DialogBackdrop, DialogContent, DialogTitle }
```

- [ ] **Step 2: Type-check**

Run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: no new errors. If `DialogPrimitive.Popup.Props`/`.Backdrop.Props`/`.Title.Props`/`.Portal.Props` type names don't match what `@base-ui/react/dialog`'s `index.parts.d.ts` actually exports, run `cat node_modules/@base-ui/react/dialog/index.parts.d.ts` (or the individual `popup/DialogPopup.d.ts` etc.) to get the exact exported type names and adjust — the *shape* (Root/Trigger/Close/Portal/Backdrop/Popup/Title) is confirmed present from `ls node_modules/@base-ui/react/dialog/`, only exact prop-type names might need a small adjustment to match this specific installed version.

---

## Phase C — Admin route tree

### Task C1: Login page

**Files:**
- Create: `apps/web/src/routes/admin/login.tsx`

**Interfaces:**
- Consumes: `useAdminLogin`, `getAdminMeQueryOptions`, `ApiError` from `@casa-dana/api` (produced by the backend plan's Task E2).

- [ ] **Step 1: Write the route**

```tsx
import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"

import { ApiError, getAdminMeQueryOptions, useAdminLogin } from "@casa-dana/api"
import { Button } from "@/components/ui/button"
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

export const Route = createFileRoute("/admin/login")({
  beforeLoad: async ({ context }) => {
    const alreadyAuthed = await context.queryClient
      .fetchQuery(getAdminMeQueryOptions())
      .then(() => true)
      .catch(() => false)
    if (alreadyAuthed) {
      throw redirect({ to: "/admin/reservations" })
    }
  },
  component: AdminLoginPage,
})

interface LoginFormValues {
  email: string
  password: string
}

function AdminLoginPage() {
  const navigate = useNavigate()
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<LoginFormValues>({
    defaultValues: { email: "", password: "" },
  })

  const { mutate: login, isPending } = useAdminLogin({
    mutation: {
      onSuccess: () => {
        navigate({ to: "/admin/reservations" })
      },
      onError: (err) => {
        const message =
          err instanceof ApiError && err.code === "INVALID_CREDENTIALS"
            ? "Incorrect email or password."
            : "Something went wrong. Try again."
        setError("password", { type: "invalid", message })
      },
    },
  })

  const onSubmit = (values: LoginFormValues) => {
    login({ data: values })
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-primary px-4">
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="w-full max-w-sm rounded-xl bg-surface p-10 shadow-editorial"
      >
        <p className="mb-1.5 font-mono text-[11px] tracking-[0.22em] text-on-surface-variant uppercase">
          Casa DaNa &amp; CasAy
        </p>
        <h1 className="mb-6 text-xl font-bold text-on-surface">Admin access</h1>

        <Field className="mb-3">
          <FieldLabel htmlFor="email">Email</FieldLabel>
          <Input id="email" type="email" autoComplete="username" {...register("email", { required: true })} />
        </Field>
        <Field className="mb-1">
          <FieldLabel htmlFor="password">Password</FieldLabel>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            {...register("password", { required: true })}
          />
        </Field>
        <FieldError errors={[errors.password]} className="mb-3 min-h-4" />

        <Button type="submit" disabled={isPending} className="w-full justify-center">
          {isPending ? "Signing in…" : "Enter dashboard"}
        </Button>
      </form>
    </div>
  )
}
```

- [ ] **Step 2: Regenerate routes and type-check**

Run: `bun --filter @casa-dana/web dev` (trigger route regeneration), then `bun --filter @casa-dana/web exec tsc --noEmit`.
Expected: no errors, assuming the backend plan's Task E2 already produced `useAdminLogin`/`getAdminMeQueryOptions`/`ApiError` — if `getAdminMeQueryOptions` doesn't exist under that exact name, run `grep -n "export const getAdminMe\|export const useAdminMe" ../../packages/api/src/generated/admin/admin.ts` (adjust the relative path as needed) to find the actual generated name (it's derived from the OpenAPI `operationId: adminMe`, so `getAdminMeQueryOptions`/`useAdminMe`/`getAdminMeQueryKey` are the expected names, but confirm against what Orval actually emitted).

---

### Task C2: Authed admin layout — auth gate + sidebar shell

**Files:**
- Create: `apps/web/src/routes/admin/_authed.tsx`
- Create: `apps/web/src/routes/admin/_authed/index.tsx`
- Create: `apps/web/src/components/admin/admin-sidebar.tsx`

**Interfaces:**
- Produces: the `AdminSidebar` component and the `/admin/_authed` pathless layout wrapping every other admin route in `ToastProvider` — consumed by Task D1 (Reservations route) and Task C3 (ComingSoon routes).
- Consumes: `getAdminMeQueryOptions`, `useAdminLogout` from `@casa-dana/api`; `ToastProvider` from Task B1.

- [ ] **Step 1: Write the sidebar**

```tsx
import { useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { CalendarDays, History, LogOut, Star, Tag, UserCog } from "lucide-react"

import { useAdminLogout } from "@casa-dana/api"

const NAV_ITEMS = [
  { to: "/admin/reservations", label: "Reservations", icon: CalendarDays },
  { to: "/admin/pricing", label: "Pricing", icon: Tag },
  { to: "/admin/reviews", label: "Reviews", icon: Star },
  { to: "/admin/owner", label: "Owner & access", icon: UserCog },
  { to: "/admin/history", label: "History", icon: History },
] as const

export default function AdminSidebar() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { mutate: logout } = useAdminLogout({
    mutation: {
      onSuccess: () => {
        queryClient.clear()
        navigate({ to: "/admin/login" })
      },
    },
  })

  const handleLogout = () => logout()

  return (
    <aside className="flex h-screen flex-col gap-6 bg-primary p-4 text-on-primary">
      <div className="px-2">
        <p className="text-sm font-bold">Casa Admin</p>
        <p className="mt-1 font-mono text-[9.5px] tracking-[0.2em] text-on-primary/60 uppercase">
          Internal · not public
        </p>
      </div>

      <nav className="flex flex-1 flex-col gap-0.5">
        {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
          <Link
            key={to}
            to={to}
            className="flex items-center gap-2.5 rounded-md px-3 py-2.5 text-[13.5px] font-medium text-on-primary/75 hover:bg-primary-container hover:text-on-primary"
            activeProps={{ className: "bg-white text-primary hover:bg-white hover:text-primary" }}
          >
            <Icon className="size-4 shrink-0" />
            {label}
          </Link>
        ))}
      </nav>

      <div className="flex flex-col gap-1.5">
        <a
          href="/"
          target="_blank"
          rel="noopener noreferrer"
          className="rounded-md px-2 py-1.5 text-[12.5px] text-on-primary/65 hover:text-on-primary"
        >
          ↗ View public site
        </a>
        <button
          type="button"
          onClick={handleLogout}
          className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-[12.5px] text-on-primary/65 hover:text-on-primary"
        >
          <LogOut className="size-3.5" />
          Log out
        </button>
      </div>
    </aside>
  )
}
```

- [ ] **Step 2: Write the authed layout**

`src/routes/admin/_authed.tsx`:

```tsx
import { Outlet, createFileRoute, redirect } from "@tanstack/react-router"

import { getAdminMeQueryOptions } from "@casa-dana/api"
import AdminSidebar from "@/components/admin/admin-sidebar"
import { ToastProvider } from "@/components/ui/toast"

export const Route = createFileRoute("/admin/_authed")({
  beforeLoad: async ({ context }) => {
    try {
      await context.queryClient.fetchQuery(getAdminMeQueryOptions())
    } catch {
      throw redirect({ to: "/admin/login" })
    }
  },
  component: AuthedAdminLayout,
})

function AuthedAdminLayout() {
  return (
    <ToastProvider>
      <div className="grid min-h-screen grid-cols-[236px_1fr]">
        <AdminSidebar />
        <main className="px-10 py-8">
          <Outlet />
        </main>
      </div>
    </ToastProvider>
  )
}
```

- [ ] **Step 3: Write the `/admin` index redirect**

`src/routes/admin/_authed/index.tsx`:

```tsx
import { createFileRoute, redirect } from "@tanstack/react-router"

export const Route = createFileRoute("/admin/_authed/")({
  beforeLoad: () => {
    throw redirect({ to: "/admin/reservations" })
  },
})
```

- [ ] **Step 4: Regenerate routes and type-check**

Run: `bun --filter @casa-dana/web dev` (trigger regeneration), then `bun --filter @casa-dana/web exec tsc --noEmit`.
Expected: no errors. If the router plugin complains about the `createFileRoute("/admin/_authed")`/`("/admin/_authed/")` path strings not matching their file locations, check `routeTree.gen.ts`'s generated ids for these files and correct the string literals to match exactly.

---

### Task C3: `ComingSoon` placeholder + its 4 routes

**Files:**
- Create: `apps/web/src/components/admin/coming-soon.tsx`
- Create: `apps/web/src/routes/admin/_authed/pricing.tsx`
- Create: `apps/web/src/routes/admin/_authed/reviews.tsx`
- Create: `apps/web/src/routes/admin/_authed/owner.tsx`
- Create: `apps/web/src/routes/admin/_authed/history.tsx`

- [ ] **Step 1: Write the shared placeholder**

```tsx
interface ComingSoonProps {
  title: string
}

export default function ComingSoon({ title }: ComingSoonProps) {
  return (
    <div>
      <h2 className="text-2xl font-bold text-on-surface">{title}</h2>
      <div className="mt-6 rounded-lg border border-outline-variant bg-surface px-5 py-16 text-center text-[13.5px] text-on-surface-variant">
        Coming soon.
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Write the 4 route files**

`src/routes/admin/_authed/pricing.tsx`:

```tsx
import { createFileRoute } from "@tanstack/react-router"

import ComingSoon from "@/components/admin/coming-soon"

export const Route = createFileRoute("/admin/_authed/pricing")({
  component: () => <ComingSoon title="Pricing" />,
})
```

`src/routes/admin/_authed/reviews.tsx`:

```tsx
import { createFileRoute } from "@tanstack/react-router"

import ComingSoon from "@/components/admin/coming-soon"

export const Route = createFileRoute("/admin/_authed/reviews")({
  component: () => <ComingSoon title="Reviews" />,
})
```

`src/routes/admin/_authed/owner.tsx`:

```tsx
import { createFileRoute } from "@tanstack/react-router"

import ComingSoon from "@/components/admin/coming-soon"

export const Route = createFileRoute("/admin/_authed/owner")({
  component: () => <ComingSoon title="Owner & access" />,
})
```

`src/routes/admin/_authed/history.tsx`:

```tsx
import { createFileRoute } from "@tanstack/react-router"

import ComingSoon from "@/components/admin/coming-soon"

export const Route = createFileRoute("/admin/_authed/history")({
  component: () => <ComingSoon title="History" />,
})
```

- [ ] **Step 3: Regenerate and type-check**

Run: `bun --filter @casa-dana/web dev` then `bun --filter @casa-dana/web exec tsc --noEmit`.
Expected: no errors.

---

## Phase D — Reservations view

### Task D1: Reservations route — search params, stats, pagination shell

**Files:**
- Create: `apps/web/src/routes/admin/_authed/reservations.tsx`

**Interfaces:**
- Produces: the `/admin/reservations` route with validated search (`property`, `status?`, `page`) — consumed by Task D2/D3 (which receive these as props).
- Consumes: `ReservationTable` (Task D2), `AddReservationDialog` (Task D3), `useListBookings` from `@casa-dana/api`.

- [ ] **Step 1: Write the route**

```tsx
import { keepPreviousData } from "@tanstack/react-query"
import { createFileRoute, useNavigate } from "@tanstack/react-router"

import { type BookingStatus, useListBookings } from "@casa-dana/api"
import AddReservationDialog from "@/components/admin/add-reservation-dialog"
import ReservationTable from "@/components/admin/reservation-table"
import { Button } from "@/components/ui/button"

const PROPERTIES = ["casadana", "casacasay"] as const
type Property = (typeof PROPERTIES)[number]
const STATUSES: Array<BookingStatus> = ["pending", "approved", "rejected", "cancelled", "paid"]
const PAGE_SIZE = 8

interface ReservationsSearch {
  property: Property
  status?: BookingStatus
  page: number
}

function isProperty(value: unknown): value is Property {
  return typeof value === "string" && (PROPERTIES as ReadonlyArray<string>).includes(value)
}

function isBookingStatus(value: unknown): value is BookingStatus {
  return typeof value === "string" && (STATUSES as ReadonlyArray<string>).includes(value)
}

function validateReservationsSearch(search: Record<string, unknown>): ReservationsSearch {
  const property = isProperty(search.property) ? search.property : "casadana"
  const status = isBookingStatus(search.status) ? search.status : undefined
  const page = Number(search.page)
  return { property, status, page: Number.isFinite(page) && page >= 1 ? page : 1 }
}

export const Route = createFileRoute("/admin/_authed/reservations")({
  validateSearch: validateReservationsSearch,
  component: ReservationsPage,
})

const PROPERTY_LABELS: Record<Property, string> = {
  casadana: "Casa DaNa",
  casacasay: "Casa CasAy",
}

function ReservationsPage() {
  const { property, status, page } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const { data } = useListBookings(
    { villa_slug: property, status, page, limit: PAGE_SIZE },
    { query: { placeholderData: keepPreviousData } },
  )
  const { data: totalsAll } = useListBookings({ villa_slug: property, limit: 1 })
  const statusTotals = useStatusTotals(property)
  const maxPage = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1

  const goToPage = (nextPage: number) => {
    navigate({ search: (prev) => ({ ...prev, page: nextPage }) })
  }

  const switchProperty = (nextProperty: Property) => {
    navigate({ search: (prev) => ({ ...prev, property: nextProperty, page: 1 }) })
  }

  return (
    <div>
      <div className="mb-7 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-on-surface">Reservations</h2>
          <p className="mt-1 text-[13.5px] text-on-surface-variant">
            Requests and confirmed stays for {PROPERTY_LABELS[property]}.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex gap-1 rounded-lg bg-surface-container p-1">
            {PROPERTIES.map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => switchProperty(p)}
                className={
                  p === property
                    ? "rounded-md bg-primary px-3 py-1.5 text-[13px] font-medium text-on-primary"
                    : "rounded-md px-3 py-1.5 text-[13px] font-medium text-on-surface-variant"
                }
              >
                {PROPERTY_LABELS[p]}
              </button>
            ))}
          </div>
          <AddReservationDialog property={property} status={status} page={page} />
        </div>
      </div>

      <div className="mb-5 grid grid-cols-2 gap-4 md:grid-cols-6">
        <StatTile label="Total" value={totalsAll?.total ?? 0} />
        {STATUSES.map((s) => (
          <StatTile key={s} label={s} value={statusTotals[s] ?? 0} />
        ))}
      </div>

      <div className="rounded-lg border border-outline-variant bg-surface">
        <div className="border-b border-outline-variant px-5 py-4">
          <h3 className="text-[14.5px] font-semibold text-on-surface">All reservations</h3>
        </div>
        <ReservationTable bookings={data?.bookings ?? []} property={property} status={status} page={page} />
        {data && data.total > PAGE_SIZE && (
          <div className="flex items-center justify-center gap-4 border-t border-outline-variant px-5 py-3.5">
            <Button type="button" variant="outline" size="sm" disabled={page <= 1} onClick={() => goToPage(page - 1)}>
              ‹ Prev
            </Button>
            <span className="text-[12.5px] text-on-surface-variant">
              Page {page} of {maxPage}
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={page >= maxPage}
              onClick={() => goToPage(page + 1)}
            >
              Next ›
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}

function useStatusTotals(property: Property): Partial<Record<BookingStatus, number>> {
  const pendingQuery = useListBookings({ villa_slug: property, status: "pending", limit: 1 })
  const approvedQuery = useListBookings({ villa_slug: property, status: "approved", limit: 1 })
  const rejectedQuery = useListBookings({ villa_slug: property, status: "rejected", limit: 1 })
  const cancelledQuery = useListBookings({ villa_slug: property, status: "cancelled", limit: 1 })
  const paidQuery = useListBookings({ villa_slug: property, status: "paid", limit: 1 })
  return {
    pending: pendingQuery.data?.total,
    approved: approvedQuery.data?.total,
    rejected: rejectedQuery.data?.total,
    cancelled: cancelledQuery.data?.total,
    paid: paidQuery.data?.total,
  }
}

function StatTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-outline-variant bg-surface px-4 py-3.5">
      <p className="text-[11px] font-semibold tracking-[0.06em] text-on-surface-variant uppercase">{label}</p>
      <p className="mt-1.5 font-mono text-2xl font-bold text-on-surface">{value}</p>
    </div>
  )
}
```

This issues 6 `useListBookings` calls total (1 unfiltered-count-only + 5 per-status-count-only, each `limit: 1` so payload is trivial, plus the 1 real paginated call) — per the design spec §7.3's stat-row approach, reusing the existing endpoint with no backend change.

- [ ] **Step 2: Confirm it fails to type-check (expected, until D2/D3 exist)**

Run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: FAILS — `@/components/admin/add-reservation-dialog` and `@/components/admin/reservation-table` don't exist yet. Fixed by Tasks D2/D3.

---

### Task D2: `ReservationTable` — status actions, delete

**Files:**
- Create: `apps/web/src/components/admin/reservation-table.tsx`

**Interfaces:**
- Consumes: `BookingResponse`, `BookingStatus`, `getListBookingsQueryKey`, `usePatchBooking`, `useDeleteBooking` from `@casa-dana/api`; `useToast` from Task B1.
- Produces: `<ReservationTable bookings status property page />` — consumed by Task D1.

- [ ] **Step 1: Write the component**

```tsx
import { useQueryClient } from "@tanstack/react-query"
import { Trash2 } from "lucide-react"

import {
  type BookingResponse,
  type BookingStatus,
  getListBookingsQueryKey,
  useDeleteBooking,
  usePatchBooking,
} from "@casa-dana/api"
import { Button } from "@/components/ui/button"
import { useToast } from "@/components/ui/toast"

const NEXT_STATUSES: Record<BookingStatus, Array<{ status: BookingStatus; label: string }>> = {
  pending: [
    { status: "approved", label: "Approve" },
    { status: "rejected", label: "Reject" },
    { status: "cancelled", label: "Cancel" },
  ],
  approved: [
    { status: "paid", label: "Mark paid" },
    { status: "cancelled", label: "Cancel" },
  ],
  paid: [{ status: "cancelled", label: "Cancel" }],
  rejected: [],
  cancelled: [],
}

const STATUS_BADGE_CLASSES: Record<BookingStatus, string> = {
  pending: "bg-secondary-container text-on-secondary-container",
  approved: "bg-primary-container text-on-primary-container",
  paid: "bg-primary text-on-primary",
  rejected: "bg-error-container text-on-error-container",
  cancelled: "bg-surface-container-high text-on-surface-variant",
}

interface ReservationTableProps {
  bookings: Array<BookingResponse>
  property: "casadana" | "casacasay"
  status?: BookingStatus
  page: number
}

export default function ReservationTable({ bookings, property, status, page }: ReservationTableProps) {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const listQueryKey = getListBookingsQueryKey({ villa_slug: property, status, page, limit: 8 })

  const { mutate: patchStatus } = usePatchBooking({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: listQueryKey })
        toast("Status updated")
      },
      onError: () => toast("Could not update status"),
    },
  })

  const { mutate: deleteBooking } = useDeleteBooking({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: listQueryKey })
        toast("Reservation deleted")
      },
      onError: () => toast("Could not delete reservation"),
    },
  })

  const handleDelete = (id: string) => {
    if (window.confirm("Delete this reservation?")) {
      deleteBooking({ id })
    }
  }

  if (bookings.length === 0) {
    return (
      <div className="px-5 py-10 text-center text-[13.5px] text-on-surface-variant">
        No reservations yet — new "Request to Book" submissions from the public site will land here.
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[720px] border-collapse text-[13px]">
        <thead>
          <tr className="border-b border-outline-variant bg-surface-container-low text-left text-[10.5px] font-semibold tracking-[0.08em] text-on-surface-variant uppercase">
            <th className="px-5 py-2.5">Guest</th>
            <th className="px-5 py-2.5">Dates</th>
            <th className="px-5 py-2.5">Guests</th>
            <th className="px-5 py-2.5">Source</th>
            <th className="px-5 py-2.5">Status</th>
            <th className="px-5 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {bookings.map((b) => (
            <tr key={b.id} className="border-b border-outline-variant last:border-0">
              <td className="px-5 py-3">
                <p className="font-semibold text-on-surface">{b.guest_name}</p>
                <p className="mt-0.5 font-mono text-[11px] text-on-surface-variant">{b.guest_email}</p>
              </td>
              <td className="px-5 py-3 font-mono text-on-surface">
                {b.check_in} → {b.check_out}
              </td>
              <td className="px-5 py-3 text-on-surface">{b.adults + b.children}</td>
              <td className="px-5 py-3 text-on-surface-variant capitalize">{b.source}</td>
              <td className="px-5 py-3">
                <div className="flex flex-wrap items-center gap-1.5">
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-1 text-[11.5px] font-semibold ${STATUS_BADGE_CLASSES[b.status]}`}
                  >
                    {b.status}
                  </span>
                  {NEXT_STATUSES[b.status].map(({ status: next, label }) => (
                    <Button
                      key={next}
                      type="button"
                      variant="outline"
                      size="xs"
                      onClick={() => patchStatus({ id: b.id, data: { status: next } })}
                    >
                      {label}
                    </Button>
                  ))}
                </div>
              </td>
              <td className="px-5 py-3">
                <button
                  type="button"
                  onClick={() => handleDelete(b.id)}
                  aria-label="Delete reservation"
                  className="rounded-md p-1.5 text-on-surface-variant hover:bg-error-container hover:text-on-error-container"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

- [ ] **Step 2: Type-check**

Run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: still fails on the missing `add-reservation-dialog` import from Task D1 — confirm the *only* remaining error is that one (i.e. this file itself compiles cleanly). Fixed by Task D3.

---

### Task D3: `AddReservationDialog`

**Files:**
- Create: `apps/web/src/components/admin/add-reservation-dialog.tsx`

**Interfaces:**
- Consumes: `ApiError`, `getListBookingsQueryKey`, `useCreateBooking` from `@casa-dana/api`; `Dialog`/`DialogTrigger`/`DialogClose`/`DialogContent`/`DialogTitle` from Task B2; `useToast` from Task B1.
- Produces: `<AddReservationDialog property status page />` — consumed by Task D1.

- [ ] **Step 1: Write the component**

```tsx
import { useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { useForm } from "react-hook-form"

import { ApiError, type BookingStatus, getListBookingsQueryKey, useCreateBooking } from "@casa-dana/api"
import { Button } from "@/components/ui/button"
import { Dialog, DialogClose, DialogContent, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useToast } from "@/components/ui/toast"

interface AddReservationFormValues {
  guestName: string
  guestEmail: string
  guestPhone: string
  checkIn: string
  checkOut: string
  adults: number
  source: "direct" | "airbnb" | "booking_com"
}

interface AddReservationDialogProps {
  property: "casadana" | "casacasay"
  status?: BookingStatus
  page: number
}

export default function AddReservationDialog({ property, status, page }: AddReservationDialogProps) {
  const [open, setOpen] = useState(false)
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<AddReservationFormValues>({
    defaultValues: {
      guestName: "",
      guestEmail: "",
      guestPhone: "",
      checkIn: "",
      checkOut: "",
      adults: 2,
      source: "direct",
    },
  })

  const listQueryKey = getListBookingsQueryKey({ villa_slug: property, status, page, limit: 8 })

  const { mutate: createBooking, isPending } = useCreateBooking({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: listQueryKey })
        toast("Reservation added")
        setOpen(false)
        reset()
      },
      onError: (err) => {
        if (err instanceof ApiError && err.code === "DATES_CONFLICT") {
          setError("checkOut", { type: "conflict", message: "Those dates overlap an existing reservation." })
        } else {
          toast("Could not add reservation")
        }
      },
    },
  })

  const onSubmit = (values: AddReservationFormValues) => {
    createBooking({
      data: {
        villa_slug: property,
        guest_name: values.guestName,
        guest_email: values.guestEmail,
        guest_phone: values.guestPhone,
        check_in: values.checkIn,
        check_out: values.checkOut,
        adults: Number(values.adults),
        children: 0,
        source: values.source,
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button type="button" />}>Add reservation</DialogTrigger>
      <DialogContent>
        <DialogTitle>Add reservation</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-3">
          <Field>
            <FieldLabel htmlFor="guestName">Guest name</FieldLabel>
            <Input id="guestName" {...register("guestName", { required: true })} />
          </Field>
          <Field>
            <FieldLabel htmlFor="guestEmail">Email</FieldLabel>
            <Input id="guestEmail" type="email" {...register("guestEmail", { required: true })} />
          </Field>
          <Field>
            <FieldLabel htmlFor="guestPhone">Phone</FieldLabel>
            <Input id="guestPhone" {...register("guestPhone", { required: true })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field>
              <FieldLabel htmlFor="checkIn">Check-in</FieldLabel>
              <Input id="checkIn" type="date" {...register("checkIn", { required: true })} />
            </Field>
            <Field>
              <FieldLabel htmlFor="checkOut">Check-out</FieldLabel>
              <Input id="checkOut" type="date" {...register("checkOut", { required: true })} />
              <FieldError errors={[errors.checkOut]} />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Field>
              <FieldLabel htmlFor="adults">Guests</FieldLabel>
              <Input
                id="adults"
                type="number"
                min={1}
                max={10}
                {...register("adults", { required: true, valueAsNumber: true })}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="source">Source</FieldLabel>
              <select
                id="source"
                {...register("source")}
                className="h-8 rounded-lg border border-outline-variant bg-transparent px-2.5 text-sm text-on-surface"
              >
                <option value="direct">Direct</option>
                <option value="airbnb">Airbnb</option>
                <option value="booking_com">Booking.com</option>
              </select>
            </Field>
          </div>
          <div className="mt-2 flex justify-end gap-2.5">
            <DialogClose render={<Button type="button" variant="outline" />}>Cancel</DialogClose>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Saving…" : "Save reservation"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
```

- [ ] **Step 2: Full type-check**

Run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: no errors now that D1/D2/D3 all exist. If `DialogTrigger`/`DialogClose`'s `render` prop doesn't accept a JSX element the way `select.tsx`'s `SelectPrimitive.Icon`'s `render` prop did, check `node_modules/@base-ui/react/dialog/trigger/DialogTrigger.d.ts` and `close/DialogClose.d.ts` for the exact prop and adjust (e.g. it may need `render={(props) => <Button type="button" {...props} />}` instead of a bare element, depending on this base-ui version's `useRender` API).

---

## Phase E — Full verification

### Task E1: Build and manual QA

**Files:** none (verification only)

- [ ] **Step 1: Type-check and build**

Run: `bun --filter @casa-dana/web exec tsc --noEmit`
Expected: no errors.

Run: `bun --filter @casa-dana/web build`
Expected: succeeds (this also runs `tsc` per the `build` script in `package.json`, then `vite build`).

- [ ] **Step 2: Bootstrap the first admin account (one-time, manual — see backend plan's Task F1 Step 5)**

Before any of the following manual checks are possible, an `admin_users` row must exist. If the backend plan's bootstrap step wasn't already done, do it now: connect to the dev Postgres and run

```sql
INSERT INTO admin_users (id, email, password_hash)
VALUES (uuid_generate_v4(), 'you@example.com', '<bcrypt-hash-of-your-password>');
```

Generate the bcrypt hash with a one-off Go snippet (`go run` a small `main.go` calling `bcrypt.GenerateFromPassword`) or any bcrypt CLI tool — do not hand-roll a hash.

- [ ] **Step 3: Manual QA against the dev server**

Start both the API (`mise run docker-dev` or equivalent) and the web dev server (`bun --filter @casa-dana/web dev`). Then, in a browser (or via the `mcp__plugin_playwright_playwright__browser_navigate`/`browser_click`/`browser_type`/`browser_snapshot` tools for automated verification):

1. Visit `/admin` while logged out → redirected to `/admin/login`.
2. Submit wrong credentials → inline error shown, stays on the login page.
3. Submit the bootstrapped admin's correct credentials → redirected to `/admin/reservations`, sidebar renders, property switcher defaults to Casa DaNa.
4. Reload the page → still authenticated (session cookie persists), lands back on Reservations (not bounced to login).
5. Switch property to Casa CasAy → table/stat row refetch and update; URL search param `?property=casacasay` reflects the switch.
6. Click "Add reservation", fill the form, submit → toast "Reservation added", new row appears, stat counts update.
7. Try adding a reservation with dates that overlap the one just created → inline conflict error on the Check-out field, no toast, form stays open.
8. On a `pending` row, click "Approve" → status badge updates to `approved`, "Mark paid"/"Cancel" become the new action buttons, toast "Status updated".
9. Click delete on a row, confirm the browser `confirm()` dialog → row disappears, toast "Reservation deleted".
10. Click "Pricing" / "Reviews" / "Owner & access" / "History" in the sidebar → each renders the "Coming soon" placeholder without erroring.
11. Click "Log out" → redirected to `/admin/login`; then try navigating directly to `/admin/reservations` → bounced back to `/admin/login` (session actually cleared, not just a client-side redirect).
12. Visit the public site (`/`, `/villa/<slug>`) → confirm `Navbar`/`Footer` still render exactly as before this work (regression check for the `_public` layout split in Task A2).

Report any step that fails with the exact error observed — do not mark this task complete on a partial pass.
