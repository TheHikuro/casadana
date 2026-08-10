# Runbook — deploy v2 to `demo.casa-dana.com`

Phase 3 of the v1 → v2 migration. Read [ADR 0002](../adr/0002-nginx-fronts-dokploy-traefik-disabled.md)
and [ADR 0003](../adr/0003-same-origin-api-routing.md) first; they explain *why* the
steps below look unusual. [ADR 0001](../adr/0001-v2-rewrite-and-v1-decommission.md)
covers what happens afterwards.

Host: `147.93.89.239`. Every step is additive — v1 (`api.casa-dana.com`) keeps serving
throughout. Nothing here edits `casadana.conf` or touches the v1 database.

## Already done on the host (verified 2026-08-10 20:45 UTC)

| | State |
|---|---|
| v1 | `active`, `NRestarts=0`, serving live traffic |
| swap | 4 GB at `/swapfile`, in `/etc/fstab`, `vm.swappiness=10` |
| `casadana.service` drop-in | `Restart=always`, `RestartSec=5`, `OOMScoreAdjust=-500`, ordered after `postgresql.service` |
| backups | `/root/backups` (mode 700), nightly `03:30` via `/etc/cron.d/casadana-backup`, 30-day retention, restore-verified |
| Dokploy | installed, Swarm active, `dokploy` + `dokploy-postgres` running |
| `dokploy-traefik` | stopped, `--restart=no` — nginx owns 80/443 |
| UI port 3000 | firewalled to loopback (iptables `INPUT` + `DOCKER-USER`, persisted) |
| DNS | `demo.casa-dana.com` A → `147.93.89.239`, resolving |
| TLS | cert issued via `certonly --webroot`, expires 2026-11-08, renews as `authenticator = webroot` with **no installer** so it can never rewrite v1's config |
| nginx vhost | `demo-casadana.conf` **enabled**, reloaded (not restarted); serves 502 until containers exist |
| certbot `cli.ini` | `renew-by-default` and `nginx` globals disabled — see below |

Both certs pass `certbot renew --dry-run` together.

### The certbot `cli.ini` globals (fixed 2026-08-10)

`/etc/letsencrypt/cli.ini` held two settings that applied to *every* certbot command
on the host. Backup at `/root/backups/cli.ini.bak-2026-08-10`.

`renew-by-default = True` forced `--force-renewal` on every run. `certbot.timer` fires
twice daily, so `api.casa-dana.com` was reissued continuously instead of at the
30-days-remaining threshold: **487 lineages** in `/etc/letsencrypt/archive` and **1428
`urn:ietf:params:acme:error:rateLimited` / "too many certificates"** rejections in the
logs. It never caused an outage — enough attempts succeeded to stay ahead of expiry —
but it kept the account pinned against Let's Encrypt's 5-duplicate-certs-per-week
limit, and that budget is shared with every other `casa-dana.com` name.

`nginx = True` forced the nginx authenticator *and installer* globally. It made
`certonly --webroot` fail outright with `Too many flags setting
configurators/installers/authenticators 'nginx' -> 'webroot'`, and would have let the
nginx installer rewrite v1's live `casadana.conf` on any bare `certonly`. It was
redundant: `renewal/api.casa-dana.com.conf` records `authenticator = nginx` and
`installer = nginx` in its own `[renewalparams]`, which is what `certbot renew`
actually uses — verified by dry-run after the change.

`domains = api.casa-dana.com` is still set. Harmless for renewal (only consulted when
a command passes no `-d`), but **never run a bare `certonly` on this host** or the new
cert picks up v1's hostname. Pass `-d` explicitly. This matters at Phase 4, which
issues apex + `www`.

`OOMScoreAdjust=-500` is loaded but not yet applied to the running process — systemd
sets it at fork time, so it engages on the next restart. `Restart=always` is already
in effect (systemd re-reads it when the process exits).

## Status: deployed 2026-08-10 22:05 UTC

All three containers are up and the site is live. Everything in the *Verify* section
below passes except outbound mail, which needs a real `RESEND_API_KEY`.

The stack was brought up with `docker compose` directly, not through Dokploy:

```bash
docker compose -f .docker/docker-compose.dokploy.yml --env-file /root/casadana-demo.env up -d
```

`/root/casadana-demo.env` (mode 600, **not** in the repo) holds the live values.
Adopting this into Dokploy later is cosmetic — same compose file, same images.

### Two bugs this deploy surfaced

**No CA trust store in the api image.** The final stage is `FROM scratch`, which ships
no `ca-certificates.crt`, so `crypto/tls` could not verify *any* outbound peer. Every
Resend call failed with `x509: certificate signed by unknown authority` — a valid
`RESEND_API_KEY` would have failed identically. It is WARN-level and the booking still
returns 201, so mail was dropped silently. Fixed by copying the bundle out of the
builder stage. The proof of the fix is that the log line changes from the x509 error to
Resend's own `[ERROR]: API key is invalid`: a reply from Resend means the HTTPS request
now completes. Any future rewrite that produces a scratch/distroless image must carry
the same `COPY`.

**Permanently `(unhealthy)` web container.** The healthcheck used
`wget -q --spider http://localhost:80`, but `nginx.conf` has a bare `listen 80;`
(IPv4 only) while the image's `/etc/hosts` maps `localhost` to `::1` too, and BusyBox
wget tries `::1` first. It reported `Connection refused` while the site served 200 on
both `127.0.0.1:3001` and the public URL. Fixed to `http://127.0.0.1:80` in both
compose files. Worth knowing generally: a `localhost` healthcheck against an
IPv4-only listener is a false negative, and under Swarm it would restart-loop a
perfectly good container.

The api service has **no** healthcheck by design — a scratch image has no shell and no
`wget`/`curl`, so every form of `test:` is unrunnable. Probe it from the host instead.

## Reference — first-time setup

Kept for rebuilding from scratch, and because Phase 4 repeats most of it.

### 1. Images

The host has locally-built `ghcr.io/lcleris/casadana-api:latest` and
`-web:latest`, so a deploy can proceed **without CI**. Compose uses a local image when
one is present under that tag.

For a reproducible pipeline, push `main` (5 commits ahead) and let
`.github/workflows/deploy.yml` publish them:

```bash
cd /root/src/casadana && git push origin main
docker pull ghcr.io/lcleris/casadana-api:latest   # `denied` = not published yet
```

If the GHCR packages are private, give Dokploy a PAT with `read:packages`. Note that a
pull then *replaces* the local image, so make sure CI has actually built the merge
first — otherwise a stale registry image overwrites a good local one.

### 2. Secrets, then deploy

There is no age key on the host, so `make decrypt` cannot run there. Decrypt locally
and paste into Dokploy's environment UI. Reach the UI over a tunnel — it is not
publicly reachable:

```bash
ssh -L 3000:localhost:3000 root@147.93.89.239   # then http://localhost:3000
```

The Dokploy admin account was claimed on 2026-08-10.

Required (the API refuses to boot if any is missing — see
`apps/api/internal/platform/config/config.go`):

```
POSTGRES_USER  POSTGRES_PASSWORD  POSTGRES_DB
JWT_SECRET             # generate fresh, do not reuse v1's
RESEND_API_KEY
MAIL_FROM              # must be on a Resend-verified domain
ADMIN_NOTIFY_EMAIL
WEB_ORIGIN=https://demo.casa-dana.com     # exact, scheme included
```

`POSTGRES_HOST` is `postgres` (the service name) and is set in the compose file, not
here — pointing it at `localhost` would reach *v1's* database.

Create a Docker Compose application in Dokploy against this repo and
`.docker/docker-compose.dokploy.yml`. Migrations are embedded and apply on API start
(`MIGRATE_ON_BOOT`, default true), so the schema needs no manual step. Optionally load
`apps/api/internal/db/seed_dev.sql`.

### 3. Bootstrapping the first admin

**There is no API route that can do this.** `POST /api/admin/users` sits inside the
`r.Group` guarded by `RequireAdminSession` (`apps/api/internal/adminauth/http.go`), so
creating an admin requires already being one. Only `/api/admin/login` and
`/api/admin/logout` are unauthenticated. v1's ASP.NET Identity PBKDF2 hash cannot be
reused either.

So the first row goes in by hand, with a bcrypt hash at the same cost the app uses
(`bcrypt.DefaultCost`, `adminauth/domain.go:34`):

```bash
docker exec casadana-postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "insert into admin_users (email, password_hash) values ('you@example.com', '<bcrypt hash>');"
```

Then verify through the public URL — the response must set `HttpOnly; Secure; SameSite=Lax`:

```bash
curl -si -X POST https://demo.casa-dana.com/api/admin/login \
  -H 'Content-Type: application/json' -d '{"email":"...","password":"..."}' | grep -i set-cookie
```

Note there is **no password-change endpoint** — only create, list and delete. To rotate
a password: log in, create a second admin, delete the first.

## Verify

```bash
curl -s  https://demo.casa-dana.com/api/health       # {"status":"ok"}
curl -sI https://demo.casa-dana.com/                 # SPA index
curl -sI https://demo.casa-dana.com/some/deep/route  # 200 via SPA fallback
curl -sI https://api.casa-dana.com/                  # v1 UNAFFECTED
ss -tlnp | grep -E '3001|8080'                       # 127.0.0.1 only, never 0.0.0.0
free -h
```

That last binding check is not paranoia: **Docker bypasses ufw**, so a bare
`8080:8080` anywhere in a compose file publishes the API to the internet even with the
firewall enabled. Re-run it after every compose change.

Then in a browser: admin login (cookie must be `Secure` and survive a reload), create a
booking, confirm the Resend mail fires.

The only valid `villa_slug` values are **`casadana`** and **`casacasay`** — no hyphens.
The list is hardcoded in `apps/api/internal/villaslug/catalog.go`, which mirrors
`apps/web/src/constants/villas.const.ts` and must be edited by hand when a villa is
added. Anything else gets `404 UNKNOWN_VILLA`. There is no `/api/villas` endpoint and
no `villas` table, so the catalog is not discoverable at runtime.

A booking `POST` also requires `guest_phone`; omitting it is a `422`, not a `400`.

## Traps

- **Never run `/var/www/casa-dana-api/docker-compose.yml`** on this host — it publishes
  `5432:5432`, exposing Postgres and colliding with the host instance.
- **Never start a second v1 instance** against the live database: `MigrateDatabase()`
  and `SeedUsers.CreateAdminUser` run on every start.
- **Never build an image from the v1 repo** — it has no `.dockerignore`, so `COPY . .`
  bakes `.env` into a layer.
- **Do not set `VITE_API_BASE_URL`.** The empty fallback is load-bearing (ADR 0003).
- `/root/backups` stays mode 700 — the dumps contain guest PII and password hashes.
- Cutover (Phase 4) should wait until **after 2026-08-25**, when the last 7 pending v1
  reservations have ended. Before then, those 7 must be resolved in v1 or re-entered by
  hand in v2 — no data is migrated.

Until containers exist, all three demo paths return **502**. That is the correct
response and confirms nginx is resolving the vhost and dialling `127.0.0.1:3001` /
`:8080` — not a config error.

## Rollback

| Change | Undo |
|---|---|
| deployed app | stop it in Dokploy; `docker compose down -v` also drops the v2 database |
| nginx vhost | `rm /etc/nginx/sites-enabled/demo-casadana.conf && systemctl reload nginx` |
| demo cert | `certbot delete --cert-name demo.casa-dana.com` |
| `cli.ini` edits | `cp /root/backups/cli.ini.bak-2026-08-10 /etc/letsencrypt/cli.ini` |
| DNS | remove the `demo` A record |

None of these touch v1. After any nginx or certbot change, re-run
`certbot renew --dry-run --no-random-sleep-on-renew` — it must report success for
**both** lineages.
