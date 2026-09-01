# Runbook — Phase 4, point casa-dana.com at v2

Moves the public site from Vercel (v1 frontend) to the v2 stack on this host. This is the only
step in the whole migration that touches live traffic.

**Prerequisites**

- [ ] v1 booking history imported ([`v1-data-import.md`](v1-data-import.md)) — do this first;
      after cutover the v1 API is idle and it is easy to forget the data is still only there
- [ ] demo validated on `develop` ([`environments.md`](environments.md))
- [ ] fresh v1 dump taken **and copied off the box**
- [ ] a real `RESEND_API_KEY` in `/root/casadana-prod.env` — see the warning below
- [ ] apex DNS TTL lowered to 60s at least a day ahead

## Read this before planning the window

**1. `www` is canonical, not the apex.** Vercel answers `https://casa-dana.com/` with a
`308 → https://www.casa-dana.com/`. Bookmarks, links and search results point at `www`. v2 must
keep that shape: serve on `www`, redirect the apex. `WEB_ORIGIN` in `/root/casadana-prod.env`
must therefore be `https://www.casa-dana.com` — go-chi/cors compares it against the browser's
`Origin` header exactly, and the API is on the same origin, so a mismatch is silent until a
preflight happens.

**2. HSTS is already committed, for two years.** Both hostnames send:

```
strict-transport-security: max-age=63072000
```

Every returning visitor's browser has cached that. It means:

- **Plain HTTP is not a fallback.** The browser rewrites `http://` to `https://` itself.
- **A cert error is a dead end.** HSTS removes the "proceed anyway" button. There is no
  click-through.

So the window between "DNS points here" and "valid certificate installed" is a **hard outage**,
not a degraded experience. Keep it at zero if possible, and seconds if not.

`includeSubDomains` is **not** set, which is why `demo`/`api`/`dokploy` were never affected. Do
not add it — that would immediately bind every subdomain to the same rule.

**3. The certificate cannot be issued before the DNS move over HTTP-01.** The original plan said
to issue first with `--webroot`, and that is wrong: HTTP-01 means Let's Encrypt fetches
`http://casa-dana.com/.well-known/acme-challenge/<token>`, which today reaches **Vercel**, not
this host. Validation fails.

DNS-01 is the way round it, and it does not need Vercel or an API plugin — the TXT record is
added by hand at Hostinger. That is Option A.

## Option A — zero gap, via a DNS-01 challenge (preferred)

DNS-01 proves control of the domain through a TXT record instead of an HTTP fetch, so the
certificate can be issued **while Vercel is still serving the site**. Nothing about v2 touches
Vercel, and the DNS move afterwards has no gap at all.

The domain is on Hostinger (`ns1/ns2.dns-parking.com`), which has no maintained certbot plugin,
so the record goes in by hand. Two names means two TXT records.

**A1.** Start the issuance. It will pause and print the records to add:

```bash
certbot certonly --manual --preferred-challenges dns \
  -d www.casa-dana.com -d casa-dana.com
```

**A2.** At Hostinger, add each TXT record it names — `_acme-challenge.www.casa-dana.com` and
`_acme-challenge.casa-dana.com`.

**A3. Confirm propagation before pressing Enter.** This is the one step that actually fails in
practice: certbot does not check for you, and a validation attempted too early burns
rate-limit budget. In a second terminal:

```bash
dig +short TXT _acme-challenge.www.casa-dana.com @1.1.1.1
dig +short TXT _acme-challenge.casa-dana.com     @1.1.1.1
```

Each must print the exact string certbot gave. Only then press Enter.

**A4. Convert the lineage to webroot after the DNS move.** A `--manual` certificate **cannot
auto-renew** — certbot says so at issuance, and the failure mode is a silent expiry ~60 days
later. Once DNS points here, HTTP-01 works, so re-issue the same lineage over webroot:

```bash
certbot certonly --webroot -w /var/www/letsencrypt \
  --cert-name www.casa-dana.com -d www.casa-dana.com -d casa-dana.com
certbot renew --dry-run --no-random-sleep-on-renew    # must pass for all four lineages
```

Then delete the two `_acme-challenge` TXT records. **Do not skip A4.** It is the difference
between a certificate that renews itself and one that expires over a weekend.

## Option B — short outage, if DNS-01 is impractical

Move DNS first, then issue with `--webroot`. Everything else must be staged and tested
beforehand so only `certbot` plus a reload remain. Expect 1–3 minutes during which the site is
**unreachable** for anyone with HSTS cached. Do it at low traffic.

## Certificate rules that apply to both options

**Both names go on one lineage.** The apex block needs a valid certificate to serve its redirect
at all — HSTS means the browser reaches the apex over HTTPS before it ever sees the 308.

**`-d` is mandatory.** `/etc/letsencrypt/cli.ini` still carries a global
`domains = api.casa-dana.com`, so a bare `certonly` silently attaches v1's hostname to the new
certificate. Never `--nginx` — that installer rewrites v1's live `casadana.conf`.

Option B's issuance, run only after DNS has moved:

```bash
certbot certonly --webroot -w /var/www/letsencrypt \
  -d www.casa-dana.com -d casa-dana.com
```

## Install the vhost

`/etc/nginx/sites-available/casadana-prod.conf` — a **new** file. Do not edit `casadana.conf`
(v1) or `demo-casadana.conf`.

```nginx
server {
    listen 443 ssl;
    http2 on;
    server_name casa-dana.com;

    ssl_certificate     /etc/letsencrypt/live/www.casa-dana.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/www.casa-dana.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    # Preserves the redirect Vercel served, so existing links and SEO are unaffected.
    return 308 https://www.casa-dana.com$request_uri;
}

server {
    listen 443 ssl;
    http2 on;
    server_name www.casa-dana.com;

    ssl_certificate     /etc/letsencrypt/live/www.casa-dana.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/www.casa-dana.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    # Matches what Vercel already committed browsers to. Dropping it would not un-commit
    # them -- their cached policy runs for two years either way -- so the only effect of
    # omitting it is that the policy silently expires. No includeSubDomains.
    add_header Strict-Transport-Security "max-age=63072000" always;

    # No X-Robots-Tag here: unlike the demo, this SHOULD be indexed.

    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;   # required: COOKIE_SECURE=true

    client_max_body_size 10m;

    # Order matters: /api/ before /. NO trailing slash on proxy_pass -- the Go API serves
    # its routes under /api, and a slash makes nginx strip the prefix so every route 404s
    # (ADR 0003). The X-Forwarded-For line above is what the review rate limiter counts on.
    location /api/ { proxy_pass http://127.0.0.1:8080; }
    location /     { proxy_pass http://127.0.0.1:3001; }
}
```

Test it against the running stack **before** DNS moves — this needs no public DNS:

```bash
nginx -t
curl -s -o /dev/null -w '%{http_code}\n' -H 'Host: www.casa-dana.com' http://127.0.0.1:3001/
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/api/health
```

Then enable (**reload**, never restart — v1 keeps serving):

```bash
ln -s /etc/nginx/sites-available/casadana-prod.conf /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx || rm /etc/nginx/sites-enabled/casadana-prod.conf
```

## Set the production origin

`WEB_ORIGIN` is still `https://demo.casa-dana.com` from when this stack served the demo:

```bash
sed -i 's|^WEB_ORIGIN=.*|WEB_ORIGIN=https://www.casa-dana.com|' /root/casadana-prod.env
SSH_ORIGINAL_COMMAND="deploy prod $(git -C /root/src/casadana rev-parse HEAD)" \
  /usr/local/bin/casadana-deploy
```

## Move DNS

At Hostinger, for `casa-dana.com`:

| Record | From | To |
|---|---|---|
| `@` A | `76.76.21.21` | `147.93.89.239` |
| `www` CNAME | `cname.vercel-dns.com` | **delete**, replace with A → `147.93.89.239` |

```bash
watch -n5 'dig +short casa-dana.com @1.1.1.1; dig +short www.casa-dana.com @1.1.1.1'
```

## Verify

```bash
curl -sI https://www.casa-dana.com/            | head -1     # 200
curl -sI https://casa-dana.com/                | head -1     # 308 -> www
curl -s   https://www.casa-dana.com/api/health               # ok
curl -sI https://www.casa-dana.com/some/deep/route | head -1 # 200, SPA fallback
curl -sI https://demo.casa-dana.com/           | head -1     # 200, unaffected
curl -sI https://api.casa-dana.com/            | head -1     # v1 still up (rollback path)
certbot renew --dry-run --no-random-sleep-on-renew           # all four lineages
```

In a browser: admin login (cookie `Secure`, survives a reload), the imported bookings visible,
and a test booking end to end.

## Rollback

DNS back to Vercel — `@` A → `76.76.21.21`, `www` CNAME → `cname.vercel-dns.com`. With a 60s TTL
this takes minutes. v1's API and the Vercel deployment must stay untouched until the grace period
ends, which is exactly why Phase 5 waits.

Nothing on this host needs undoing; the vhost can stay enabled, since with DNS elsewhere it
receives nothing.

## After the grace period (1–2 weeks)

Delete the Vercel project, then proceed to Phase 5
([ADR 0001](../adr/0001-v2-rewrite-and-v1-decommission.md)).

**v2 uses nothing from Vercel — but keep the project until the grace period ends anyway.** It is
the rollback path. Deleting it early means a bad cutover has nowhere to fall back to, since the
v1 frontend exists only there.

**Renewal follows DNS.** Once the apex points here the lineage renews over webroot (A4). If the
apex is ever moved away again while the lineage still exists, renewal fails silently and surfaces
as an outage ~60 days later.
