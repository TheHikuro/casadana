# 0004 — GitHub Actions owns deploys; Dokploy is an observation surface

- **Status:** Accepted
- **Date:** 2026-08-10

## Context

[0002](0002-nginx-fronts-dokploy-traefik-disabled.md) installed Dokploy and then took away
the two things it normally does: Traefik was stopped so nginx could keep 80/443 for v1's
certbot renewal, and routing was moved into hand-written nginx vhosts. What remained was the
expectation that Dokploy would still be the thing that *deploys* — a Docker Compose
application pointed at this repo, redeploying on a webhook.

That is not what was built. `.github/workflows/deploy.yml` now builds both images, asserts
the web bundle is same-origin clean, and then SSHes to the host to run
`/usr/local/bin/casadana-deploy`, which pulls the sha-pinned images, runs `up -d`, polls
health and rolls back on failure. See
[`docs/runbooks/continuous-deployment.md`](../runbooks/continuous-deployment.md).

So there are two candidate deploy controllers for one stack, and the question is which owns
it. Leaving it implicit is the actual hazard: the natural reading of "we set up Dokploy for
v2" is that the stack should be adopted into Dokploy, and doing that later looks like
finishing the job rather than breaking it.

## Decision

**GitHub Actions is the only thing that deploys. Do not create a Dokploy application for
this stack.**

Dokploy stays installed and is now reachable at `dokploy.casa-dana.com`
([runbook](../runbooks/dokploy-panel-subdomain.md)) as a **read-mostly operations surface**:
container status, logs, stats, an exec terminal, and the Swarm/Docker view of the host. Those
are genuinely useful and conflict with nothing.

### Why not adopt the stack into Dokploy

Both controllers would own the same containers, by the same names, from the same compose
file:

- Dokploy maintains **its own git checkout** and redeploys from it. `casadana-deploy` does
  `git reset --hard <sha>` on `/root/src/casadana`. The two would disagree about which commit
  is deployed, with no single source of truth.
- Both call `docker compose up -d` on project `casadana` with `container_name:` pinned. A
  Dokploy redeploy would silently revert a CI deploy, and vice versa. Whichever ran last
  wins, and neither records that it clobbered the other.
- The compose file selects images with `${IMAGE_TAG:-latest}`. CI pins `sha-<commit>`;
  Dokploy would supply nothing and get `latest`. **A Dokploy redeploy would therefore quietly
  move a pinned deployment back to `latest`**, defeating the rollback design and making the
  running version unknowable from the recorded tag.
- The health-check-and-rollback logic lives in the deploy script. Dokploy would not run it.

None of this is a defect in Dokploy. It is what happens when two orchestrators share one
stack.

### Why the pipeline was built this way rather than on Dokploy's webhook

- **A webhook needs the panel publicly reachable**, which is a real exposure increase
  ([runbook](../runbooks/dokploy-panel-subdomain.md) covers it). The SSH path needs no inbound
  HTTP at all.
- **The deploy logic belongs in version control.** Steps configured through a web UI are not
  reviewable, not diffable, and not recoverable if the panel's own database is lost.
- **The build must gate the deploy.** The web bundle assertion exists because a leaked
  `VITE_API_BASE_URL` produces a site that loads and then fails every request
  ([0003](0003-same-origin-api-routing.md)) — undetectable from the server side. A webhook
  fired on push cannot express "only if the artefact passed its check".

The deploy key is confined by `command="/usr/local/bin/casadana-deploy",restrict` in
`authorized_keys`, so it cannot open a shell — verified by attempting `whoami`, a read of the
env file, and a bare shell request, all refused with exit 64. That is a narrower grant than
either a public admin panel or an exposed Docker socket.

## Consequences

- **Deploys require GitHub to be up.** For a demo this is acceptable; the manual path is two
  `docker compose` commands and is documented in the runbook.
- **`/root/src/casadana` is owned by the pipeline.** It gets `git reset --hard` on every
  deploy. Uncommitted work there blocks deploys (the script refuses rather than discarding),
  which is the intended trade.
- **Dokploy's UI will show containers it did not create.** That is expected. Use it to look,
  not to redeploy. Its Domains tab in particular must stay unused — it writes Traefik labels,
  and Traefik is deliberately stopped ([0002](0002-nginx-fronts-dokploy-traefik-disabled.md)).
- **Dokploy has earned its keep even without deploying**: it supplies logs, stats and a
  terminal behind one TLS + basic-auth front door, which otherwise means an SSH session.

## Rules for future changes

- Do not create a Dokploy application for `casadana` while the Actions pipeline exists. To
  switch, delete the deploy job and the `authorized_keys` entry **first**, so there is never a
  window with two controllers.
- **This ADR is not a reason to keep Dokploy forever.** After
  [0002](0002-nginx-fronts-dokploy-traefik-disabled.md)'s exit condition (v1 decommissioned,
  no certbot/nginx constraint left), handing 80/443 to Traefik and running the stack as a real
  Dokploy application becomes coherent, because Dokploy would then own routing *and* deploys
  instead of half of each. That is a deliberate migration: retire this pipeline in the same
  change, and supersede this ADR.
- If the API is rewritten in Rust, nothing here changes — the pipeline deals in image tags and
  `/api/health`, not languages. Keep the image names and the health route, or update the
  deploy script's `verify()` alongside.
