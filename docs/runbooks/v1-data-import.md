# Runbook — import v1 reservations into v2

Moves booking history out of the v1 .NET database (host Postgres, db `casadana`) into v2
(`casadana-postgres` container, db `casadana_v2`). Rationale and the full list of what is and
is not imported: [ADR 0005](../adr/0005-v1-data-is-imported-after-all.md).

**Run this before Phase 5.** `dropdb casadana` is irreversible and the import cannot be redone
afterwards.

## What it does

`scripts/v1-import.sql` reads a staging table `v1_res` — a verbatim copy of v1's `"Reservations"` —
and inserts into `bookings`. It is **idempotent**: v1's UUID primary keys are carried over, so a
second run inserts nothing.

Expected on this dataset: **25 rows in → 18 bookings out** (14 `approved`, 4 `pending`).
The gap is duplicate form submissions, not dropped data.

## Procedure

**1. Fresh dump first.** Non-negotiable — this is the last moment the source exists.

```bash
sudo -u postgres pg_dump -Fc casadana > /root/backups/casadana-pre-import-$(date +%F).dump
```

**2. Export `"Reservations"` to CSV.** Via stdout: the `postgres` user cannot write to most
directories, and `\copy ... to '<path>'` fails with a bare `Permission denied`.

```bash
sudo -u postgres psql -At -d casadana -c "copy (
  select \"Id\",\"NumberOfPersons\",\"Start\",\"End\",\"Price\",\"Phone\",\"Email\",
         \"FirstName\",\"LastName\",coalesce(\"Description\",''),\"Status\"
  from \"Reservations\") to stdout csv" > /root/v1_reservations.csv
```

One row's `Status` contains a newline, so the file has **more lines than rows**. That is correct
CSV quoting — do not "fix" it.

**3. Stage it inside the v2 container.**

```bash
docker exec -i casadana-postgres psql -q -U casadana_app -d casadana_v2 -c '
CREATE TABLE v1_res (
  "Id" uuid, "NumberOfPersons" int, "Start" timestamptz, "End" timestamptz,
  "Price" numeric, "Phone" text, "Email" text, "FirstName" text,
  "LastName" text, "Description" text, "Status" text);'

docker exec -i casadana-postgres psql -q -U casadana_app -d casadana_v2 \
  -c '\copy v1_res from stdin csv' < /root/v1_reservations.csv

docker exec casadana-postgres psql -U casadana_app -d casadana_v2 -tAc 'select count(*) from v1_res'
# -> 25
```

**4. Run the import.** `ON_ERROR_STOP=1` matters: the script is one transaction and an unmapped
status is *designed* to abort it.

```bash
docker exec -i casadana-postgres psql -q -v ON_ERROR_STOP=1 -U casadana_app -d casadana_v2 \
  < /root/src/casadana/scripts/v1-import.sql
```

**5. Verify before dropping the staging table.**

```bash
docker exec -i casadana-postgres psql -U casadana_app -d casadana_v2 -P pager=off <<'SQL'
-- every imported date equals the v1 Brussels-local date
SELECT count(*) AS date_mismatches FROM bookings b JOIN v1_res r ON r."Id"=b.id
WHERE b.check_in  <> (r."Start" AT TIME ZONE 'Europe/Brussels')::date
   OR b.check_out <> (r."End"   AT TIME ZONE 'Europe/Brussels')::date;

-- no distinct v1 request was dropped by the dedup
SELECT count(*) AS dropped FROM (
  SELECT DISTINCT lower(btrim("Email")) e,
         ("Start" AT TIME ZONE 'Europe/Brussels')::date ci,
         ("End"   AT TIME ZONE 'Europe/Brussels')::date co FROM v1_res) v
LEFT JOIN bookings b ON b.guest_email=v.e AND b.check_in=v.ci AND b.check_out=v.co
WHERE b.id IS NULL;

SELECT status, count(*) FROM bookings WHERE villa_slug='casadana' GROUP BY 1;
SQL
```

`date_mismatches` and `dropped` must both be **0**. If either is not, roll back (below) and stop.

**6. Clean up.**

```bash
docker exec casadana-postgres psql -U casadana_app -d casadana_v2 -c 'DROP TABLE v1_res'
shred -u /root/v1_reservations.csv     # contains guest names, emails and phone numbers
```

Then check the admin dashboard: the bookings list should show 18 Casa DaNa entries dated
2025-03-21 through 2026-08-26.

## Rollback

The import only ever inserts, and only rows whose ids came from v1, so it is precisely reversible:

```bash
docker exec casadana-postgres psql -U casadana_app -d casadana_v2 \
  -c 'DELETE FROM bookings WHERE id IN (SELECT "Id" FROM v1_res)'
```

Do this **before** dropping `v1_res` — it is the only list of which ids were imported. If the table
is already gone, re-stage it from the CSV or the dump.

## Traps

- **Do not cast the timestamps with a bare `::date`.** v1 stored Brussels-local midnight as UTC, so
  a plain cast moves **21 of 25** stays one day earlier — and the result still looks like a valid
  booking, so nothing catches it downstream. The script converts via
  `AT TIME ZONE 'Europe/Brussels'` everywhere.
- **`btrim()` does not strip newlines**, only spaces. One v1 status is `'Confirmed \n'`. The script
  passes an explicit character set, `btrim(x, E' \t\r\n')`.
- **Leave the duplicate handling alone unless you re-check it.** The dedup keeps the largest party
  per `(email, check_in, check_out)`. Changing the tie-break changes guest counts.
- **Two imported stays overlap** (Aug 2025 and Aug 2026). Those are genuine v1 double-bookings, not
  an import bug. Both are in the past.
- **`created_at` is the stay start date, not a submission time** — v1 never recorded one.
- Do not run the import against the demo stack expecting it to reach production once the two are
  split; they have separate databases.
