# Runbook — expose the Dokploy panel on `dokploy.casa-dana.com`

Puts the Dokploy UI behind nginx + TLS + basic auth so it can be managed from a browser
instead of an SSH tunnel.

Host config lives outside the repo (it is host state, not code):

| Path | Purpose |
|---|---|
| `/etc/nginx/sites-available/dokploy-casadana.conf` | the vhost — **staged, not enabled** |
| `/etc/nginx/conf.d/websocket-upgrade.conf` | `$connection_upgrade` map, already active |
| `/etc/nginx/.htpasswd-dokploy` | bcrypt basic-auth file, `root:www-data 640` |
| `/root/casadana-dokploy-basicauth.txt` | the credentials, mode 600 |

## Understand the exposure before enabling this

Dokploy controls Docker on this host. Anyone who reaches an authenticated session can start
a privileged container, mount the host filesystem, and read v1's live database, the backup
dumps and every secret on the box. It is not "just a deploy UI" — it is root.

Until this vhost is enabled, the panel is reachable only over an SSH tunnel:

```bash
ssh -L 3000:localhost:3000 root@147.93.89.239   # then http://localhost:3000
```

That remains the lower-risk option and keeps working regardless. Enabling the vhost trades
some of that safety for convenience, which is why basic auth sits in front: nginx rejects
unauthenticated requests before Dokploy is reached, so its login form is never exposed to
scanners, and a Dokploy auth bug is not directly reachable from the internet.

**Port 3000 stays firewalled either way.** Docker publishes it on `0.0.0.0` and Docker
bypasses ufw, so these iptables rules are the only thing keeping it off the raw internet:

```
-A INPUT -i lo -p tcp --dport 3000 -j ACCEPT
-A INPUT       -p tcp --dport 3000 -j DROP
-A DOCKER-USER -p tcp --dport 3000 -j DROP
```

Do not remove them. The vhost reaches the panel over loopback, which the first rule permits.

> Testing `curl http://147.93.89.239:3000` **from the host itself** returns 200 and proves
> nothing — host-to-own-public-IP traffic goes over `lo` and matches the ACCEPT. Test from
> another machine.

## Enable it

**1. DNS.** Add `dokploy.casa-dana.com` A → `147.93.89.239`. Wait for it:

```bash
dig +short dokploy.casa-dana.com A     # must print 147.93.89.239
```

**2. Pre-flight the ACME path** before invoking certbot, so a failed validation does not
spend Let's Encrypt rate-limit budget:

```bash
echo ping > /var/www/letsencrypt/.well-known/acme-challenge/ping.txt
curl -s http://dokploy.casa-dana.com/.well-known/acme-challenge/ping.txt   # -> ping
rm /var/www/letsencrypt/.well-known/acme-challenge/ping.txt
```

**3. Certificate.** Webroot, no installer:

```bash
certbot certonly --webroot -w /var/www/letsencrypt -d dokploy.casa-dana.com
```

`-d` is mandatory. `/etc/letsencrypt/cli.ini` still carries a global
`domains = api.casa-dana.com`, so a bare `certonly` silently attaches v1's hostname to the
new certificate. Never `--nginx` — that installer rewrites v1's live `casadana.conf`.

**4. Enable the vhost.** Reload, never restart, so v1 keeps serving:

```bash
ln -s /etc/nginx/sites-available/dokploy-casadana.conf /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx || rm /etc/nginx/sites-enabled/dokploy-casadana.conf
```

**5. Verify.**

```bash
curl -so /dev/null -w '%{http_code}\n' https://dokploy.casa-dana.com/            # 401
curl -so /dev/null -w '%{http_code}\n' -u loan:<pw> https://dokploy.casa-dana.com/  # 200
curl -so /dev/null -w '%{http_code}\n' https://api.casa-dana.com/                # v1 unchanged
curl -so /dev/null -w '%{http_code}\n' https://demo.casa-dana.com/               # 200
certbot renew --dry-run --no-random-sleep-on-renew                               # all lineages
```

Then in a browser confirm the **build-log view and the container terminal** work — those are
the WebSocket paths, and they are the part most likely to break behind a proxy.

## Traps

- **WebSockets need the `Upgrade`/`Connection` pair.** Without it the UI loads and looks
  healthy while log streaming and the terminal hang with no error. That is what
  `conf.d/websocket-upgrade.conf` and `proxy_http_version 1.1` are for. `proxy_buffering off`
  matters too, or progress output arrives in chunks long after the fact.
- **`proxy_read_timeout` is 3600s deliberately.** The 60s default cuts long builds off
  mid-stream and makes a working deploy look like a failure.
- **Do not use Dokploy's own Domains tab** to add this hostname. It writes Traefik labels,
  and `dokploy-traefik` is deliberately stopped with `--restart=no` because nginx must keep
  80/443 for v1's certbot renewal (ADR 0002). Routing is nginx's job on this host.
- **Basic auth credentials are not the Dokploy login.** Two separate prompts; only the first
  is managed here.
- Rotate the basic-auth password with `htpasswd -B /etc/nginx/.htpasswd-dokploy loan`
  (`-B` = bcrypt; the default MD5-based format is weak).

## Rollback

```bash
rm /etc/nginx/sites-enabled/dokploy-casadana.conf && systemctl reload nginx
certbot delete --cert-name dokploy.casa-dana.com
```

Neither touches v1 or the demo. The SSH tunnel keeps working throughout.
