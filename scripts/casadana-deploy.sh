#!/usr/bin/env bash
#
# Deploy target for CI. Installed on the VPS as /usr/local/bin/casadana-deploy and wired
# up as the FORCED COMMAND for the CI deploy key in /root/.ssh/authorized_keys, so that
# key cannot run anything else -- not a shell, not an arbitrary docker command.
#
#   command="/usr/local/bin/casadana-deploy",restrict ssh-ed25519 AAAA... casadana-ci-deploy
#
# The only accepted request is `deploy <40-hex-commit-sha>`, read from SSH_ORIGINAL_COMMAND
# and pattern-matched before use. SSH_ORIGINAL_COMMAND is entirely attacker-controlled if
# the private key ever leaks, so the regex is a security boundary, not a convenience.
#
# See docs/runbooks/continuous-deployment.md. Updating this file in git does NOT update the
# host; re-run `install -m 700 scripts/casadana-deploy.sh /usr/local/bin/casadana-deploy`.
set -euo pipefail

REPO=/root/src/casadana
ENV_FILE=/root/casadana-demo.env
COMPOSE_FILE="$REPO/.docker/docker-compose.dokploy.yml"
STATE=/root/.casadana-deployed-tag
LOG=/var/log/casadana-deploy.log

log() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*" | tee -a "$LOG" >&2; }
compose() { docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" "$@"; }

# --- validate the request -----------------------------------------------------------
CMD="${SSH_ORIGINAL_COMMAND:-}"
if [[ ! $CMD =~ ^deploy\ [0-9a-f]{40}$ ]]; then
  log "REFUSED bad request: ${CMD:-<empty>}"
  echo "refused: expected 'deploy <40-hex-sha>'" >&2
  exit 64
fi
SHA=${CMD#deploy }
TAG="sha-$SHA"

PREV=$(cat "$STATE" 2>/dev/null || echo latest)
log "START tag=$TAG prev=$PREV"

# --- sync the checkout --------------------------------------------------------------
# The compose file, not just the images, comes from the repo -- a compose change would
# otherwise never reach the host. Pinning the checkout to the same sha CI built keeps the
# two in step.
#
# A dirty tree is a hard failure rather than a `reset --hard`: silently destroying someone's
# in-progress edit on the box is worse than a failed deploy. Clear it by hand, then re-run.
if [ -n "$(git -C "$REPO" status --porcelain)" ]; then
  log "REFUSED $REPO has uncommitted changes"
  git -C "$REPO" status --short >&2
  echo "refused: commit, stash or discard local changes on the host first" >&2
  exit 65
fi
git -C "$REPO" fetch --quiet origin
if ! git -C "$REPO" cat-file -e "$SHA^{commit}" 2>/dev/null; then
  log "REFUSED unknown commit $SHA"
  exit 66
fi
git -C "$REPO" reset --hard --quiet "$SHA"
log "checkout at $(git -C "$REPO" rev-parse --short HEAD)"

# --- roll forward -------------------------------------------------------------------
export IMAGE_TAG="$TAG"
compose pull api web
compose up -d --remove-orphans

# --- verify, and roll back if it did not come up ------------------------------------
# Probed on the loopback publications rather than through nginx, so a DNS or TLS problem
# is not misreported as a bad deploy. /api/health is the API's own route; the web check is
# the SPA index.
verify() {
  local i
  for i in $(seq 1 30); do
    if curl -fsS --max-time 3 -o /dev/null http://127.0.0.1:8080/api/health \
      && curl -fsS --max-time 3 -o /dev/null http://127.0.0.1:3001/; then
      return 0
    fi
    sleep 2
  done
  return 1
}

if verify; then
  echo "$TAG" >"$STATE"
  log "OK deployed $TAG"
  compose ps --format '{{.Name}}\t{{.Image}}\t{{.Status}}'
  exit 0
fi

log "FAILED health check after 60s; rolling back to $PREV"
compose logs --tail 40 api >&2 || true
export IMAGE_TAG="$PREV"
if compose up -d && verify; then
  log "ROLLED BACK to $PREV"
else
  log "ROLLBACK ALSO FAILED -- site is down, manual intervention required"
fi
exit 1
