# 0005 — v1 booking data is imported after all

- **Status:** Accepted — supersedes the "no data migration" decision in [0001](0001-v2-rewrite-and-v1-decommission.md)
- **Date:** 2026-09-01

## Context

[0001](0001-v2-rewrite-and-v1-decommission.md) decided that v1's data would be **archived, not
migrated**, and listed five columns as the reason. That table was correct about four of them and
wrong about the one that actually decided the outcome:

| v1 `Reservations` | v2 `bookings` | 0001 said | actually |
|---|---|---|---|
| — | `villa_slug` NOT NULL | **no v1 source value** | **wrong** — v1 only ever hosted Casa DaNa, so the value is the constant `'casadana'` |
| `NumberOfPersons` int | `adults` + `children` | unsplittable | true, but survivable: import everyone as an adult |
| `Price` numeric | *(separate pricing tables)* | not portable | true — and it stays out (below) |
| `Start`/`End` timestamptz | `check_in`/`check_out` DATE | lossy | true, and the loss is desirable |
| `AspNetUsers` PBKDF2 | `admin_users` | hashes not portable | true — the admin was recreated by hand |

`villa_slug` was the only genuine blocker: a NOT NULL column with no defensible source value. It
was never really absent, only implicit, because v1 had no concept of a second villa. Once the
constant is supplied the remaining gaps are all *lossy but defensible*, not *impossible*, and
"archive it" stops being the only option.

The rest of 0001 — the decommission runbook, the DataProtection keys, the nuget cache, the
dump-retention rule — is unaffected and still governs Phase 5.

## Decision

**Import v1's reservations into v2** using [`scripts/v1-import.sql`](../../scripts/v1-import.sql),
documented in [`docs/runbooks/v1-data-import.md`](../runbooks/v1-data-import.md).

Result, verified against a scratch database and then **run on production 2026-09-01**:
**25 v1 rows → 18 bookings** (14 `approved`, 4 `pending`), no request dropped, every date
identical to the v1 local date, party sizes unchanged, and the admin API returning all of them.
The imported ids are listed in `/root/backups/v2/v1-imported-ids-2026-09-01.txt`, and
`/root/backups/casadana-pre-import-2026-09-01.dump` is the source snapshot taken immediately
beforehand.

It was run twice. The first run landed in `casadana_v2` — the database the stack had used as the
demo — which turned out to hold demo bookings, demo audit history and a second admin alongside
three weeks of real pricing configuration. Production was to contain v1's data and nothing else,
so the stack was repointed at a fresh `casadana_prod` and the import re-run there. `casadana_v2`
was abandoned rather than dropped, so it is still in the same volume.

**The pricing configuration was then copied across** — `villa_pricing_settings`, `season_rules`
and `price_overrides` only, by a `pg_dump --data-only -t` of those three tables. It was three
weeks of deliberate work and it exists nowhere else, so keeping it was worth the one exception to
"v1 data only"; bookings, audit history and admin users were not copied. Verified afterwards that
the precedence chain resolves: base €85 → season rule €130 → per-date override €250.
A readable copy also sits at `/root/backups/v2/demo-config-for-reentry-2026-09-01.txt`.

### What is imported, and what is deliberately not

- **`id` is carried over unchanged.** v1's primary keys are already UUIDs, so every imported row
  is traceable to the archived dump by key alone. This is the provenance record, and it is also
  what makes the script idempotent via `ON CONFLICT (id) DO NOTHING`.
- **`source` is `'direct'`, not a marker like `'v1-import'`.** `source` is a closed enum —
  `oneof=direct airbnb booking_com` in `booking/http.go`, regenerated into
  `packages/api/src/generated/schemas/bookingSource.ts` — so an out-of-band value would fail admin
  round-trips and break the frontend's typed union. These bookings did arrive through the v1
  website form, so `direct` is also simply true.
- **`Price` is not imported.** v2 has no price column on `bookings`; the quoted total lives in
  `price_overrides` / `season_rules` / `villa_pricing_settings`, which model *rates*, not
  *what one guest was quoted*. Back-deriving a nightly rate from a v1 total would invent pricing
  history. The totals stay in the archived dump and the CSV.
- **`Calendars` (4 rows) is not imported.** It is stale pricing config — two of the four rows are
  overlapping duplicates of the same 2026 range, and one expired in 2025. It maps cleanly onto
  `season_rules`, but re-entering three rates in the admin UI is less work than reviewing an
  import, and produces a state someone actually chose.
- **`created_at` is synthetic.** v1 never recorded when a request was submitted. The stay's start
  date is used so the admin list sorts chronologically instead of collapsing every row onto the
  import timestamp. **It is not a submission time** and must not be read as one.

### Three things the data itself forced

**1. The timestamps are Brussels-local midnight stored as UTC.** They read `22:00` (CEST) or
`23:00` (CET) *on the previous day*. A bare `::date` cast shifts **21 of the 25 rows one day
early** — silently, because the result is still a valid booking. Every conversion goes through
`AT TIME ZONE 'Europe/Brussels'`. One row spans the March 2025 DST boundary and lands on local
midnight at both ends, which is what confirms the interpretation rather than assuming it.

**2. One status is `'Confirmed \n'`.** Plain `btrim()` strips spaces but **not** newlines, so the
first version of the mapping missed it. The `CASE` has no `ELSE`, so the unmapped value became
NULL and tripped the NOT NULL on `status`, aborting the transaction — the guard is what surfaced
the bug, and it stays.

**3. v1's booking form had no submit guard.** Guests who clicked twice created duplicate rows:
11 `Pending` rows are 4 distinct requests. Dedup is on `(email, check_in, check_out)`, keeping the
largest party — the value a guest most plausibly corrected upward on the retry.

## Consequences

- **v1's booking history is live in v2**, not just in a dump. The dump retention rule from 0001
  still applies: `Price`, the raw duplicate rows and the original `Status` strings exist only there.
- **Two overlapping stays are imported** (Carolina GEMA / Morgan Martin in Aug 2025, and the two
  Aug 2026 requests). These are real v1 double-bookings, not import artefacts. v2 does not enforce
  non-overlap at the database level — `bookings_overlap_idx` is a plain index, not an exclusion
  constraint — so they import cleanly. All are in the past, so availability is unaffected.
- **Nothing imported is in the future.** The last stay ended 2026-08-26; the import ran after that.
  The "zero-loss cutover window after 2026-08-25" from 0001 has closed with nothing outstanding.
- **The import must run before `dropdb casadana`** in Phase 5, and Phase 5's step ordering already
  puts the drop last. Re-running the import after the drop is impossible.
- **This does not revive the ETL question for anything else.** Admin users, pricing and calendars
  remain hand-recreated.
