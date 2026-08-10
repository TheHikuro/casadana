# 0002 — nginx fronts 80/443; Dokploy's Traefik stays parked

- **Status:** Accepted (has an explicit exit condition — see below)
- **Date:** 2026-08-10
- **Applies to host:** `147.93.89.239`

## Context

Dokploy was installed to build and run v2 while the v1 .NET API stays live. Three things want ports 80/443 on this one host:

1. **nginx** — already there, terminating TLS for `api.casa-dana.com` (v1)
2. **Dokploy's Traefik** — the installer starts it on 80/443 by default
3. **the `caddy` service in `.docker/docker-compose.yml`** — also declares `80:80` and `443:443`

Only one can bind. The deciding fact is in v1's certificate renewal config:

```
# /etc/letsencrypt/renewal/api.casa-dana.com.conf
authenticator = nginx
installer = nginx
```

**v1's TLS renewal is bound to nginx.** Take nginx off port 80 and `certbot.timer` keeps running without ever renewing. The certificate expires **2026-11-08**, first renewal attempt around 2026-10-09 — so removing nginx produces a v1 outage roughly two months later, long after anyone would connect it to the change. That is the worst failure shape available, and it is the reason nginx stays.

## Decision

**nginx keeps 80/443 and remains the only TLS terminator. Dokploy's Traefik is stopped and prevented from restarting. v2 containers publish to loopback only, and nginx reverse-proxies to them.**

```
Internet :80/:443
   │
   nginx  (certbot, config for v1 UNCHANGED)
   ├── api.casa-dana.com   → 127.0.0.1:5000     v1 .NET, systemd
   └── demo.casa-dana.com
         ├── /api/*        → 127.0.0.1:8080     v2 Go API
         └── /*            → 127.0.0.1:3001     v2 web (nginx in container)
```

Dokploy is used only as a build/deploy/logs UI. It routes nothing.

### How Traefik is parked

The installer creates it with **`docker run --restart always`**, *not* `docker service create` — so `docker service scale dokploy-traefik=0` does **not** work on it:

```bash
docker stop dokploy-traefik
docker update --restart=no dokploy-traefik   # required, or it returns on reboot
```

Verify with `ss -tlnp | grep -E ':(80|443) '` — only nginx should appear.

### Loopback binding is mandatory, not cosmetic

**Docker bypasses ufw.** It writes its own iptables chains, so a published port is reachable from the internet even when `ufw status` says inactive or a ufw rule appears to deny it. On this host `ufw` is inactive and the INPUT policy is ACCEPT.

Therefore every v2 port binding is written `127.0.0.1:<host>:<container>`. A bare `8080:8080` publishes the API — and a bare `5432:5432` would publish a database — straight to the internet. `.docker/docker-compose.dokploy.yml` follows this; `.docker/docker-compose.yml` (dev) does not and must not be used on this host.

### Port numbers

| Port | Owner | Note |
|---|---|---|
| 80, 443 | nginx | TLS for v1 and v2 |
| 3000 | **Dokploy UI** | firewalled; see below |
| 3001 | v2 web container | 3001 *because* Dokploy owns 3000 — the dev compose's `3000:80` would collide |
| 8080 | v2 Go API | loopback only |
| 5000 | v1 .NET | loopback only, unchanged |
| 5432 | host Postgres (v1) | loopback only; v2's Postgres has **no** host port |

### The Dokploy UI is firewalled

The installer publishes the UI on `0.0.0.0:3000`, and Dokploy's first-run flow lets whoever arrives first create the admin account. Confirmed publicly reachable immediately after install, so:

```bash
iptables -I INPUT 1 -i lo -p tcp --dport 3000 -j ACCEPT
iptables -I INPUT 2      -p tcp --dport 3000 -j DROP
iptables -I DOCKER-USER 1 -p tcp --dport 3000 -j DROP   # covers the DNAT/forward path
```

Persisted via `iptables-persistent` to `/etc/iptables/rules.v4`. Access the UI through an SSH tunnel:

```bash
ssh -L 3000:localhost:3000 root@147.93.89.239   # then http://localhost:3000
```

Note that testing this from the host itself with its own public IP is misleading — the kernel routes that via `lo` and it hits the ACCEPT rule. Test from a separate network namespace, e.g. `docker run --rm postgres:16 timeout 5 bash -c 'echo > /dev/tcp/147.93.89.239/3000'`.

## Alternatives rejected

- **Give Traefik 80/443 and route v1 through it.** Cleanest end state, and v1 would tolerate it (it reads no `X-Forwarded-*` headers). Rejected *for now* only because of the certbot binding: it forces migrating a live certificate to Traefik's ACME during the window when v1 must stay stable. Deferred, not dismissed.
- **Containerise v1 into Dokploy.** Rejected — see [0001](0001-v2-rewrite-and-v1-decommission.md) "Host facts": DataProtection keys outside the app directory, `PGHOST=localhost` breaking inside a container, `.env` baking into the image with no `.dockerignore`, and migrations running on every start. Substantial risk for an app with a scheduled deletion date.
- **Use the repo's `caddy` service.** Same port conflict, and `Caddyfile` targets `casa-dana.com`, not the demo subdomain.

## Consequences

- Two proxy layers exist on the box, but only one binds a public port. Traefik idles as an exited container.
- TLS for v2 is issued by certbot, not Dokploy. Use `certbot certonly --webroot -w /var/www/letsencrypt`, **not** `--nginx`, so certbot does not rewrite the existing `casadana.conf`. The port-80 server block has no `server_name`, so it is the default server and already answers ACME challenges for any hostname.
- Dokploy's domain/routing features go unused; adding a domain in its UI does nothing while Traefik is stopped. Routing changes mean editing nginx.
- The UI is only reachable over SSH, which is deliberate.

## Exit condition

**This decision is scoped to the v1 coexistence period.** Once [0001](0001-v2-rewrite-and-v1-decommission.md) step 4 completes — v1 gone, `api.casa-dana.com` certificate deleted — no certbot-to-nginx binding remains and nothing else needs port 80. Traefik may then take over:

```bash
systemctl disable --now nginx
docker update --restart=always dokploy-traefik
docker start dokploy-traefik
```

…after moving each vhost to a Dokploy domain entry (remembering the `/api` path rule in [0003](0003-same-origin-api-routing.md)) and removing the loopback port bindings. `Caddyfile` is kept in the repo as a record of that intended topology.

Until then, **do not "simplify" this by removing nginx.** Read the Context section first.
