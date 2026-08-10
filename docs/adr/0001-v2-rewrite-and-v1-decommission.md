# 0001 — v2 rewrite, and how v1 gets decommissioned

- **Status:** Accepted
- **Date:** 2026-08-10
- **Applies to host:** `147.93.89.239` (Ubuntu 24.04, 3.8 GB RAM, 48 GB disk)

## Context

Casa Dana v1 has run in production since February 2025:

| | v1 | v2 |
|---|---|---|
| API | .NET 9 + EF Core, `casa-dana-api.dll` | Go 1.26 + sqlc |
| Repo | `github.com/TheHikuro/casa-dana-api` | `github.com/lcleris/casadana` (this one) |
| Runs as | `casadana.service` (systemd), `/var/www/casa-dana-api`, port 5000 | Docker containers via Dokploy |
| Public at | `api.casa-dana.com` behind nginx | `demo.casa-dana.com`, later `casa-dana.com` |
| Frontend | separate deploy on **Vercel** (`casa-dana.com` → `76.76.21.21`) | `apps/web`, same origin as the API (see [0003](0003-same-origin-api-routing.md)) |
| Database | **host** Postgres 16, db `casadana` (8 MB) | dedicated `postgres:16-alpine` container + volume |

v2 is a full rewrite, not a port. Both must run side by side until v2 is proven, then v1 and every .NET dependency get removed from the host.

## Decision

### 1. v1 data is archived, not migrated

The two schemas are **structurally incompatible**, and the gaps need information that no longer exists:

| v1 `Reservations` | v2 `bookings` | portable? |
|---|---|---|
| `NumberOfPersons` (int) | `adults` + `children` (smallint each) | **no** — the split was never recorded |
| — | `villa_slug` TEXT NOT NULL | **no** — v1 is single-villa, there is no source value |
| `Price` (numeric, per reservation) | not on `bookings`; pricing lives in `pricing` / `price_overrides` | **no** |
| `Start` / `End` (timestamptz) | `check_in` / `check_out` (DATE) | lossy |
| `FirstName` + `LastName` | `guest_name` (TEXT) | trivial |
| `Status` varchar(20) `'Confirmed'`/`'Pending'` | `booking_status` enum: `pending`,`approved`,`rejected`,`cancelled`,`paid` | needs remapping |
| `Description` | `message` | trivial |
| *(none)* | `created_at` / `updated_at` | unavailable |
| `AspNetUsers` (ASP.NET Identity, PBKDF2) | `admin_users` (own `adminauth`) | **no** — hashes are not portable |
| `Calendars` (4 rows) | no equivalent | n/a |

Writing an ETL would mean inventing `adults`/`children` splits and a `villa_slug`. The dataset does not justify it.

**v1 data as of 2026-08-10** (the whole database is 8 MB):

- 25 reservations, 4 calendars, **1** admin user, 1 EF migration-history row
- 13 `Confirmed` — earliest start 2025-03-20, **latest end 2026-08-04, i.e. all already in the past**
- 11 `Pending` — **7 still active or future, all ending by 2026-08-25**
- Data quirk: one `Status` value contains an embedded newline.

> **Zero-loss cutover window: after 2026-08-25.** After that date no v1 booking is still live, so starting v2 with an empty `bookings` table loses nothing operationally. Cutting over *before* then requires resolving those 7 pending requests in v1 or re-entering them by hand.

The v2 admin account is created fresh through `adminauth`. Nothing carries over.

### 2. Backups are the only surviving record

The host had **no backups of any kind** until 2026-08-10. Now in place:

- `/usr/local/bin/casadana-backup.sh` — `pg_dump -Fc` + a CSV of `Reservations`, 30-day retention
- `/etc/cron.d/casadana-backup` — daily 03:30, logs to `/var/log/casadana-backup.log`
- Output in `/root/backups` (mode 700; dumps contain guest emails, phone numbers and password hashes)

A restore was verified on 2026-08-10 into a scratch database: 25/4/1 rows and 10 tables, matching live.

> **Keep the final pre-decommission dump for at least a year, off the box.** Once step 4 below runs, it is the only copy of Casa Dana's booking history.

### 3. Order of operations

1. Backups verified and copied off-box *(done 2026-08-10)*
2. Dokploy installed, nginx keeps 80/443 — see [0002](0002-nginx-fronts-dokploy-traefik-disabled.md) *(done 2026-08-10)*
3. v2 on `demo.casa-dana.com`, validated
4. Cutover: apex + `www` DNS off Vercel onto this host; keep v1 reachable 1–2 weeks as rollback
5. Decommission (below)

### 4. Decommission runbook

Only after v2 has run clean through the grace period, and with a **verified** dump in hand.

```bash
# a. final backup FIRST, verified and off-box
/usr/local/bin/casadana-backup.sh
# verify: createdb resttest && pg_restore -d resttest /root/backups/casadana-<date>.dump

# b. stop the app
systemctl disable --now casadana.service
rm /etc/systemd/system/casadana.service
rm -rf /etc/systemd/system/casadana.service.d   # hardening drop-in, see "Host facts" below
systemctl daemon-reload

# c. nginx vhost + certificate
rm /etc/nginx/sites-enabled/casadana.conf
nginx -t && systemctl reload nginx
certbot delete --cert-name api.casa-dana.com

# d. .NET toolchain (~1 GB; /root/.nuget alone is 922 MB)
apt purge -y 'dotnet-sdk-*' 'aspnetcore-runtime-*' 'dotnet-runtime-*' 'dotnet-host*'
apt autoremove -y
rm -rf /usr/lib/dotnet /root/.dotnet /root/.nuget /root/.aspnet /var/www/casa-dana-api

# e. database LAST
sudo -u postgres dropdb casadana
sudo -u postgres dropuser casadana
```

Then remove the `api.casa-dana.com` DNS A record, and update `casadana-backup.sh` (it backs up a database that no longer exists) or delete it along with its cron file.

## Host facts that exist nowhere else

These were found by inspecting the running server. They are **not** recoverable from either repository once the host is cleaned.

- **`/root/.aspnet/DataProtection-Keys/` — 6 XML keys, oldest Feb 2025.** ASP.NET Identity encrypts password-reset and email-confirmation tokens with these. They are outside `/var/www/casa-dana-api`, so any move or containerisation of v1 silently loses them: JWT logins keep working (separate `JWT_SECRET_KEY`), and the breakage only shows up when a guest clicks a reset link. Step (d) deletes them deliberately — fine, because v1 is gone by then.
- **v1 runs migrations and reseeds the admin on *every* start.** `Program.cs` calls `app.MigrateDatabase()` and `SeedUsers.CreateAdminUser(...)` unconditionally. Never run a second v1 instance against the live database — two processes will race on EF migrations.
- **Never run `/var/www/casa-dana-api/docker-compose.yml` on this host.** It publishes `5432:5432`, which would expose Postgres to the internet and collide with the host instance.
- **v1's `Dockerfile` has no `.dockerignore`.** `COPY . .` would bake `.env` — Postgres password, `JWT_SECRET_KEY`, a Gmail app password and `Authentication__RootPassword` — into an image layer. Never push an image built from that repo.
- **v1 CORS is pinned to `https://www.casa-dana.com`** (`Extensions/CorsExtensions.cs`) with `AllowCredentials`. Note `www`, not the apex. At cutover, decide which host is canonical and 301 the other; v2 is same-origin so CORS stops mattering.
- **v1 is proxy-agnostic:** it never calls `UseForwardedHeaders` and reads no `X-Forwarded-*`. That is why moving the proxy in front of it is safe.
- **`casadana.service` has a hardening drop-in** at `/etc/systemd/system/casadana.service.d/override.conf` adding `Restart=always`, `OOMScoreAdjust=-500` and ordering after `postgresql.service`. The original unit had **no `Restart=`**, so any crash left the API down until noticed.
- v1 footprint for scale: **143 MB RSS** for the API, 31 MB for Postgres. v1 was never the memory problem.

## Consequences

- v1's booking history survives only as a `pg_dump` plus a CSV. Restoring it means standing up .NET 9 and EF Core again — treat it as cold archive, not a rollback path.
- After step 4 there is no rollback to v1. The rollback boundary is step 4, not the DNS cutover.
- Guests with in-flight v1 password-reset links are unaffected in practice: there is exactly 1 admin user and no guest accounts.
- The host reclaims ~1 GB and loses two .NET SDKs (8.0.129, 9.0.119).
- Once v1 is gone, [0002](0002-nginx-fronts-dokploy-traefik-disabled.md)'s constraint lifts and the proxy layer can be reconsidered.
