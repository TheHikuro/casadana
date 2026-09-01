#!/usr/bin/env bash
#
# Deploy target for CI. Installed on the VPS as /usr/local/bin/casadana-deploy and wired
# up as the FORCED COMMAND for the CI deploy key in /root/.ssh/authorized_keys, so that
# key cannot run anything else -- not a shell, not an arbitrary docker command.
#
#   command="/usr/local/bin/casadana-deploy",restrict ssh-ed25519 AAAA... casadana-ci-deploy
#
# The only accepted request is `deploy <env> <40-hex-commit-sha>`, read from
# SSH_ORIGINAL_COMMAND and pattern-matched before use. SSH_ORIGINAL_COMMAND is entirely
# attacker-controlled if the private key ever leaks, so the regex is a security boundary,
# not a convenience. <env> is matched against a closed alternation, never interpolated into
# a path before validation -- otherwise `deploy ../../etc <sha>` would pick the env file.
#
# TWO ENVIRONMENTS, ONE SCRIPT:
#   prod  <- main     -> www.casa-dana.com
#   demo  <- develop  -> demo.casa-dana.com
#
# They must not share a checkout. Both do `git reset --hard`, so one tree would mean a demo
# deploy silently moving production's working copy onto a develop commit -- and since the
# COMPOSE FILE is read from that tree, the next prod deploy would apply develop's compose
# definition. Hence a directory per environment.
#
# See docs/runbooks/continuous-deployment.md and docs/runbooks/environments.md. Updating this
# file in git does NOT update the host; re-run
#   install -m 700 scripts/casadana-deploy.sh /usr/local/bin/casadana-deploy
set -euo pipefail

LOG=/var/log/casadana-deploy.log
log() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*" | tee -a "$LOG" >&2; }

# --- validate the request -----------------------------------------------------------
CMD="${SSH_ORIGINAL_COMMAND:-}"
if [[ $CMD =~ ^deploy\ (prod|demo)\ ([0-9a-f]{40})$ ]]; then
  ENVNAME="${BASH_REMATCH[1]}"
  SHA="${BASH_REMATCH[2]}"
elif [[ $CMD =~ ^deploy\ ([0-9a-f]{40})$ ]]; then
  # Pre-split form, from before demo moved to its own branch. Production is the only thing
  # it could ever have meant. Kept so an in-flight or replayed old workflow run cannot fail
  # confusingly; remove once no such run can reach this host.
  ENVNAME=prod
  SHA="${BASH_REMATCH[1]}"
  log "NOTE legacy request without an environment; assuming prod"
else
  log "REFUSED bad request: ${CMD:-<empty>}"
  echo "refused: expected 'deploy <prod|demo> <40-hex-sha>'" >&2
  exit 64
fi

case "$ENVNAME" in
  prod) REPO=/root/src/casadana      ; BRANCH=main    ;;
  demo) REPO=/root/src/casadana-demo ; BRANCH=develop ;;
esac
ENV_FILE="/root/casadana-$ENVNAME.env"
STATE="/root/.casadana-deployed-tag-$ENVNAME"
COMPOSE_FILE="$REPO/.docker/docker-compose.dokploy.yml"
TAG="sha-$SHA"

for f in "$ENV_FILE" "$COMPOSE_FILE"; do
  [ -r "$f" ] || { log "REFUSED missing $f"; echo "refused: $f not readable" >&2; exit 67; }
done

# The verify() probes need the loopback ports, which differ per environment and are defined
# in the env file. Read rather than source it: the file holds secrets and arbitrary values,
# and nothing here needs them in the environment.
envget() { sed -n "s/^$1=//p" "$ENV_FILE" | tail -1; }
API_PORT=$(envget API_PORT); WEB_PORT=$(envget WEB_PORT)
if [[ ! $API_PORT =~ ^[0-9]{2,5}$ || ! $WEB_PORT =~ ^[0-9]{2,5}$ ]]; then
  log "REFUSED $ENV_FILE lacks usable API_PORT/WEB_PORT"
  exit 68
fi

compose() { docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" "$@"; }

PREV=$(cat "$STATE" 2>/dev/null || echo latest)
log "START env=$ENVNAME tag=$TAG prev=$PREV ports=$API_PORT/$WEB_PORT"

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
# Guards against a prod deploy of a develop-only commit (and vice versa) if the workflow is
# ever mis-edited: CI is what maps branch to environment, so this only catches a mistake,
# but the mistake it catches is deploying unreviewed code to production.
if ! git -C "$REPO" merge-base --is-ancestor "$SHA" "origin/$BRANCH" 2>/dev/null; then
  log "REFUSED $SHA is not an ancestor of origin/$BRANCH (env=$ENVNAME)"
  echo "refused: commit is not on origin/$BRANCH" >&2
  exit 69
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
    if curl -fsS --max-time 3 -o /dev/null "http://127.0.0.1:$API_PORT/api/health" \
      && curl -fsS --max-time 3 -o /dev/null "http://127.0.0.1:$WEB_PORT/"; then
      return 0
    fi
    sleep 2
  done
  return 1
}

if verify; then
  echo "$TAG" >"$STATE"
  log "OK deployed $TAG to $ENVNAME"
  compose ps --format '{{.Name}}\t{{.Image}}\t{{.Status}}'
  exit 0
fi

log "FAILED health check after 60s on $ENVNAME; rolling back to $PREV"
compose logs --tail 40 api >&2 || true
export IMAGE_TAG="$PREV"
if compose up -d && verify; then
  log "ROLLED BACK $ENVNAME to $PREV"
else
  log "ROLLBACK ALSO FAILED -- $ENVNAME is down, manual intervention required"
fi
exit 1
