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
| nginx vhost | staged at `/etc/nginx/sites-available/demo-casadana.conf`, **deliberately not symlinked** |

> The vhost is unlinked on purpose: it references a certificate that does not exist yet,
> so enabling it before step 3 makes `nginx -t` fail and takes v1 down with it.

`OOMScoreAdjust=-500` is loaded but not yet applied to the running process — systemd
sets it at fork time, so it engages on the next restart. `Restart=always` is already
in effect (systemd re-reads it when the process exits).

## Blockers — these need credentials or DNS, in this order

### 1. Push the branch

Two commits exist only on the host, at `/root/src/casadana`. CI must run before there
is anything to deploy.

```bash
cd /root/src/casadana
git push -u origin chore/dokploy-deploy-and-adrs
```

Then merge to `main` — `.github/workflows/deploy.yml` triggers on `main` and publishes
`ghcr.io/lcleris/casadana-api:latest` and `-web:latest`. Confirm on the host:

```bash
docker pull ghcr.io/lcleris/casadana-api:latest   # `denied` = not published yet
```

If the GHCR packages are private, give Dokploy a PAT with `read:packages`.

### 2. DNS

`demo.casa-dana.com` A → `147.93.89.239`. Leave `casa-dana.com` and `www` on Vercel.

```bash
getent hosts demo.casa-dana.com   # must resolve before step 3
```

### 3. TLS

`certonly --webroot`, **never `--nginx`** — the nginx installer rewrites
`casadana.conf`, which is v1's live config.

```bash
certbot certonly --webroot -w /var/www/letsencrypt -d demo.casa-dana.com
```

The port-80 default server already answers the ACME challenge for any hostname via
`/etc/nginx/snippets/letsencrypt-acme-challenge.conf`, so no config change is needed
to get the challenge served.

### 4. Enable the vhost

```bash
ln -s /etc/nginx/sites-available/demo-casadana.conf /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx    # reload, not restart — v1 keeps serving
```

If `nginx -t` fails, `rm` the symlink and reload again; v1 is unaffected either way.

### 5. Secrets, then deploy

There is no age key on the host, so `make decrypt` cannot run there. Decrypt locally
and paste into Dokploy's environment UI. Reach the UI over a tunnel — it is not
publicly reachable:

```bash
ssh -L 3000:localhost:3000 root@147.93.89.239   # then http://localhost:3000
```

The admin account is still unclaimed; whoever opens it first becomes admin.

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
`apps/api/internal/db/seed_dev.sql`. Create the v2 admin through the app's own
`adminauth` path — v1's ASP.NET Identity hash cannot be reused.

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

## Rollback

| Step | Undo |
|---|---|
| 1–2 | nothing to undo (repo + DNS only) |
| 3 | `certbot delete --cert-name demo.casa-dana.com` |
| 4 | `rm /etc/nginx/sites-enabled/demo-casadana.conf && systemctl reload nginx` |
| 5 | stop the Dokploy app; `docker compose down -v` drops the v2 database |

None of these touch v1.
