-- Imports v1 (.NET / EF Core) "Reservations" into v2 `bookings`.
--
-- Supersedes the "no data migration" decision in ADR 0001. That decision rested
-- on villa_slug having no source value in v1; v1 only ever hosted Casa DaNa, so
-- the constant 'casadana' is correct and the blocker is gone. See ADR 0005.
--
-- Expects a staging table `v1_res` holding a verbatim copy of v1's
-- "Reservations" (see docs/runbooks/v1-data-import.md for how to load it).
-- Idempotent: re-running inserts nothing new, because v1's UUID primary keys are
-- carried over unchanged and ON CONFLICT DO NOTHING catches the repeat.

BEGIN;

WITH src AS (
    SELECT
        r."Id",
        -- v1 stored Brussels-local midnight as UTC, so the timestamps read 22:00
        -- (CEST) or 23:00 (CET) on the PREVIOUS day. A bare ::date cast therefore
        -- shifts 19 of the 25 rows one day early -- and silently, because the
        -- result is still a valid booking. The zone conversion is mandatory.
        (r."Start" AT TIME ZONE 'Europe/Brussels')::date AS check_in,
        (r."End"   AT TIME ZONE 'Europe/Brussels')::date AS check_out,
        lower(btrim(r."Email"))                          AS guest_email,
        -- v1 let names in with trailing and doubled spaces ("Christophe  Dessy").
        btrim(regexp_replace(r."FirstName" || ' ' || r."LastName", '\s+', ' ', 'g')) AS guest_name,
        btrim(r."Phone")                                 AS guest_phone,
        coalesce(btrim(r."Description"), '')             AS message,
        -- One v1 row carries 'Confirmed \n'. Plain btrim() strips SPACES ONLY and
        -- leaves the newline, so the CASE below misses it -- an explicit character
        -- set is required. No ELSE: an unmapped status yields NULL and trips the
        -- NOT NULL on `status`, aborting the whole transaction rather than
        -- importing a booking in the wrong state. That guard caught this exact row.
        CASE btrim(r."Status", E' \t\r\n')
            WHEN 'Confirmed' THEN 'approved'
            WHEN 'Pending'   THEN 'pending'
        END::booking_status                              AS status,
        -- v1 had a single NumberOfPersons with no adult/child split. Every guest
        -- is imported as an adult; `adults >= 1` also needs the GREATEST guard.
        GREATEST(r."NumberOfPersons", 1)::smallint       AS adults,
        r."Start"                                        AS created_at
    FROM v1_res r
),
-- v1's public form had no submit guard, so guests who clicked twice created
-- duplicate rows: 25 raw rows are 19 distinct requests. Dedup on
-- (email, check_in, check_out) and keep the largest party, which is the value
-- the guest most likely corrected upward on the retry.
deduped AS (
    SELECT DISTINCT ON (guest_email, check_in, check_out) *
    FROM src
    ORDER BY guest_email, check_in, check_out, adults DESC, "Id"
)
INSERT INTO bookings (
    id, villa_slug, guest_name, guest_email, guest_phone,
    check_in, check_out, adults, children, message, status,
    source, locale, created_at, updated_at
)
SELECT
    "Id",
    'casadana',          -- v1 hosted exactly one villa
    guest_name, guest_email, guest_phone,
    check_in, check_out, adults,
    0,                   -- no v1 source for a child count
    message, status,
    'direct',            -- these did arrive through the v1 website form
    'fr',                -- v1 was French-only; matches the 0010 default
    -- v1 never recorded when a request was submitted. The stay start date is
    -- used so the admin list sorts chronologically instead of collapsing onto
    -- the import timestamp. It is NOT a real submission time.
    created_at, created_at
FROM deduped
ON CONFLICT (id) DO NOTHING;

COMMIT;
