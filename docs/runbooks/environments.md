# Runbook — the two environments

One host, one compose file, two independent stacks.

| | **prod** | **demo** |
|---|---|---|
| Branch | `main` | `develop` |
| Hostname | `www.casa-dana.com` | `demo.casa-dana.com` |
| Compose project (`STACK_NAME`) | `casadana` | `casadana-demo` |
| Containers | `casadana-{web,api,postgres}` | `casadana-demo-{web,api,postgres}` |
| Loopback ports (web / api) | 3001 / 8080 | 3011 / 8090 |
| Database | `casadana_v2` | `casadana_demo` |
| Volume | `casadana_postgres_data` | `casadana-demo_postgres_data` |
| Checkout | `/root/src/casadana` | `/root/src/casadana-demo` |
| Env file | `/root/casadana-prod.env` | `/root/casadana-demo.env` |
| Deployed tag | `/root/.casadana-deployed-tag-prod` | `/root/.casadana-deployed-tag-demo` |

Both are driven by `.docker/docker-compose.dokploy.yml`; everything above that differs comes
from the `--env-file`. There is no per-environment compose file, deliberately — two files
drift, and the drift is only discovered when one environment breaks in a way the other did not.

## Why the split exists

Before 2026-09-01 there was one stack and `main` deployed straight to `demo.casa-dana.com`.
Once the apex points at this host, `main` is production, so merging needs somewhere safe to
land first. `develop` is that place.

**The stack that is now production is the same one that served the demo since 2026-08-10** — it
kept its volume, its `STACK_NAME`, its ports and its admin user. The *demo* is the newly built,
disposable one. That direction is deliberate: production runs the configuration with three weeks
of uptime behind it, not the one created this afternoon.

## Rules

- **Never point both env files at the same `STACK_NAME`.** It is the compose project name, which
  is what namespaces the volume — identical names mean demo writes into production's database.
  The compose file declares it as `${STACK_NAME:?...}` with no default so an unset value is an
  error rather than a guess.
- **Never reuse ports.** 3000 is Dokploy's UI. Check with `ss -tlnp` after any change, and
  confirm every binding is `127.0.0.1:` — Docker bypasses ufw, so a bare mapping is public.
- **Secrets are independent.** `JWT_SECRET` and `POSTGRES_PASSWORD` differ, so a demo token
  cannot authenticate against production and a demo leak cannot reach guest data.
- **One image pair serves both.** The web bundle embeds no hostname ([ADR 0003](../adr/0003-same-origin-api-routing.md)),
  so the digest that passed CI on `develop` is the same artefact `main` promotes. Do not add a
  per-environment build.
- **`develop` is the merge target.** Merge feature work into `develop`, verify on the demo, then
  merge `develop` into `main`. A hotfix straight to `main` is possible but skips that check.

## Deploying

Automatic: push to `develop` → demo, merge to `main` → prod. Both go through
[`continuous-deployment.md`](continuous-deployment.md); the only difference is the argument the
runner sends, `deploy demo <sha>` or `deploy prod <sha>`.

By hand on the host:

```bash
ENV=demo   # or prod
REPO=/root/src/casadana$([ $ENV = demo ] && echo -demo)
IMAGE_TAG=$(cat /root/.casadana-deployed-tag-$ENV) \
  docker compose -f $REPO/.docker/docker-compose.dokploy.yml \
                 --env-file /root/casadana-$ENV.env ps
```

## Adding a third environment

Copy an env file, change `STACK_NAME`, both ports and `POSTGRES_DB`, generate fresh secrets, add
a checkout, add the branch to `.github/workflows/deploy.yml`, and add the name to the
alternation in `scripts/casadana-deploy.sh`. That alternation is a security boundary — the
environment name selects a file path, so it is matched against a closed list and never
interpolated unvalidated.

## Traps

- **The deploy script refuses a dirty checkout** (exit 65). Editing files directly in
  `/root/src/casadana*` blocks the next deploy until the tree is clean. That is intended.
- **The two checkouts must not be merged into one.** Both run `git reset --hard`; sharing a tree
  would let a demo deploy move production's working copy onto a develop commit — and the compose
  file is read from that tree, so the next prod deploy would apply develop's definition.
- **The script re-checks branch ancestry** (exit 69): a sha is only deployable to the environment
  whose branch contains it. If CI is ever mis-edited, the deploy fails instead of shipping
  unreviewed code to production.
- **`RESEND_API_KEY` is a placeholder in both environments.** Mail fails at send time. Harmless
  on demo; on production it means no booking confirmation reaches a guest or the owner.
