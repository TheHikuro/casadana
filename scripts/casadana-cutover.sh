#!/usr/bin/env bash
# Phase 4 cutover, run AFTER the A records are repointed at Hostinger.
#
# The window that matters is between DNS landing and the certificate existing: HSTS
# (max-age=63072000, set by Vercel) means a missing cert is a hard failure with no
# click-through, not a warning. So this does every remaining step back to back instead of
# leaving them to be typed.
#
# Does nothing until DNS actually points here, so it is safe to start before the edit.
set -euo pipefail

IP=147.93.89.239
NS=ns1.dns-parking.com
VHOST=/etc/nginx/sites-available/casadana-prod.conf
LINK=/etc/nginx/sites-enabled/casadana-prod.conf
say() { printf '\n=== %s\n' "$*"; }

say "waiting for casa-dana.com + www to point at $IP (authoritative: $NS)"
# Poll the authoritative server: Let's Encrypt resolves from authority too, so this is the
# same view the CA will get. A recursive resolver could still be serving the 4-hour-TTL
# Vercel answer and would make us wait for no reason.
for i in $(seq 1 120); do
  a=$(dig +short A casa-dana.com     @$NS | tail -1)
  w=$(dig +short A www.casa-dana.com @$NS | tail -1)
  [ "$a" = "$IP" ] && [ "$w" = "$IP" ] && { echo "  both point here"; break; }
  printf '  apex=%-15s www=%-15s waiting...\r' "${a:-none}" "${w:-none}"
  sleep 10
  [ "$i" = 120 ] && { echo; echo "TIMEOUT: DNS never updated. Nothing was changed."; exit 1; }
done

say "issuing the certificate (webroot, instant)"
# -d is mandatory: cli.ini still carries a global `domains = api.casa-dana.com`, so a bare
# certonly would silently attach v1's hostname. Never --nginx: it rewrites v1's live config.
certbot certonly --webroot -w /var/www/letsencrypt --non-interactive --agree-tos \
  --cert-name www.casa-dana.com -d www.casa-dana.com -d casa-dana.com

say "enabling the production vhost"
ln -sfn "$VHOST" "$LINK"
# reload, never restart -- v1 keeps serving. Roll the symlink back if the config is bad,
# otherwise a syntax error takes v1 down with us.
if ! nginx -t; then rm -f "$LINK"; echo "nginx config rejected; vhost NOT enabled"; exit 1; fi
systemctl reload nginx

say "switching WEB_ORIGIN to production and redeploying"
sed -i 's|^WEB_ORIGIN=.*|WEB_ORIGIN=https://www.casa-dana.com|' /root/casadana-prod.env
SSH_ORIGINAL_COMMAND="deploy prod $(git -C /root/src/casadana rev-parse HEAD)" \
  /usr/local/bin/casadana-deploy

say "verifying"
for u in https://www.casa-dana.com/ https://casa-dana.com/ \
         https://www.casa-dana.com/api/health https://demo.casa-dana.com/ \
         https://api.casa-dana.com/; do
  printf '  %-40s %s\n' "$u" "$(curl -sI -o /dev/null -w '%{http_code}' --max-time 15 "$u")"
done
echo
echo "expected: www 200, apex 308, health 200, demo 200, v1 404 (Kestrel's own)"
echo "v1 service: $(systemctl is-active casadana)"
