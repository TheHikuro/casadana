# Runbook — continuous deployment to the VPS

Merging a PR to `main` builds both images, asserts the web bundle is same-origin clean,
then deploys to `147.93.89.239` and verifies the result — rolling back automatically if
the new build does not answer.

Workflow: `.github/workflows/deploy.yml`. Host script: `scripts/casadana-deploy.sh`.

## The chain

```
PR merged to main
  └─ job: build  (matrix: api, web)
       ├─ buildx → push ghcr.io/lcleris/casadana-{api,web}:latest
       │                                              :sha-<40-hex>
       └─ web only: assert the bundle embeds no absolute localhost URL   ← gate
  └─ job: deploy  (needs: build — so a failed assertion blocks the deploy)
       └─ ssh root@VPS "deploy <sha>"
            └─ forced command → /usr/local/bin/casadana-deploy
                 ├─ validate the request against ^deploy [0-9a-f]{40}$
                 ├─ refuse if the host checkout is dirty
                 ├─ git fetch && git reset --hard <sha>      (compose file must match images)
                 ├─ IMAGE_TAG=sha-<sha> docker compose pull api web && up -d
                 ├─ poll 127.0.0.1:8080/api/health and 127.0.0.1:3001/ for 60s
                 └─ on failure: redeploy the previous tag from /root/.casadana-deployed-tag
```

Images are tagged with the commit sha as well as `latest`, so **a deploy is reproducible and
a rollback is a tag change** rather than a rebuild. The tag actually running is recorded in
`/root/.casadana-deployed-tag`.

`.docker/docker-compose.dokploy.yml` reads `${IMAGE_TAG:-latest}`, so an interactive
`docker compose up -d` on the host still behaves as before.

## Why SSH with a forced command, and not the alternatives

The deploy key in `authorized_keys` is pinned to a single command:

```
command="/usr/local/bin/casadana-deploy",restrict ssh-ed25519 AAAA... casadana-ci-deploy
```

`command=` means the client's requested command is **discarded** — sshd always runs the
deploy script, and the original request is merely readable via `$SSH_ORIGINAL_COMMAND`.
`restrict` disables agent/port/X11 forwarding, PTY allocation and `~/.ssh/rc`. So a leaked
private key buys an attacker one thing: the ability to ask for a deploy of some commit sha.
It cannot open a shell.

That is why the script regex-validates `$SSH_ORIGINAL_COMMAND` before using it — the value
is fully attacker-controlled in that scenario. It is a security boundary, not input tidying.

Rejected alternatives:

- **A Dokploy webhook.** Would require the panel to be publicly reachable *and* the app to
  be created through the Dokploy UI. It also puts the deploy logic somewhere that is not in
  version control.
- **A poller on the VPS** (Watchtower or a timer diffing GHCR digests). Needs no inbound
  access and no key in GitHub, which is genuinely more secure — but it is not "deploy on
  merge", it re-deploys whenever a tag moves, and it has nowhere good to put the health-check
  rollback. Worth revisiting if the SSH key becomes a concern.
- **`docker context` / remote Docker API.** Exposing the Docker socket, even over TLS, is
  strictly more powerful than a forced command that can only run one script.

## Host-side prerequisites

Already in place unless noted:

| | |
|---|---|
| `/usr/local/bin/casadana-deploy` | mode 700, installed from `scripts/casadana-deploy.sh` |
| `/root/src/casadana` | git checkout on `main`, clean tree |
| `/root/casadana-demo.env` | mode 600, holds the runtime secrets |
| `/var/log/casadana-deploy.log` | mode 600, appended by every run |
| GHCR packages | **public** — the host needs no registry credential |
| `authorized_keys` entry | **must be added by hand** (see below) |

GitHub repo secrets (already set):

| Secret | Contents |
|---|---|
| `VPS_HOST` | `147.93.89.239` |
| `VPS_SSH_KEY` | the deploy private key (ed25519, no passphrase) |
| `VPS_KNOWN_HOSTS` | hashed host key, so the runner uses `StrictHostKeyChecking=yes` |

Host details live in secrets rather than in the workflow because **this repo is public**.

### Installing the deploy key

The one step that cannot be automated from here. On the VPS:

```bash
printf 'command="/usr/local/bin/casadana-deploy",restrict %s\n' \
  'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEIGj4LIZf01gsyFHZM0fgDuG86uBM/DqwoqQILnbgrf casadana-ci-deploy' \
  >> /root/.ssh/authorized_keys
chmod 600 /root/.ssh/authorized_keys
```

Fingerprint: `SHA256:NLuBPdzMrkDeD/c+yhCb38AxeNDzIIpG89u1gb3v4KI`

Until this is added, the build succeeds and the deploy step fails with
`Permission denied (publickey)`.

## Verify

```bash
# from the host, imitating what the runner sends
SSH_ORIGINAL_COMMAND="deploy $(git -C /root/src/casadana rev-parse HEAD)" /usr/local/bin/casadana-deploy

# the forced command must refuse anything else
SSH_ORIGINAL_COMMAND="whoami"    /usr/local/bin/casadana-deploy   # exit 64
SSH_ORIGINAL_COMMAND="deploy .." /usr/local/bin/casadana-deploy   # exit 64

tail -20 /var/log/casadana-deploy.log
cat /root/.casadana-deployed-tag
```

After a real run, confirm the containers moved to the sha tag:

```bash
docker compose -f .docker/docker-compose.dokploy.yml --env-file /root/casadana-demo.env \
  ps --format '{{.Name}}\t{{.Image}}\t{{.Status}}'
```

## Manual deploy and rollback

```bash
cd /root/src/casadana
export IMAGE_TAG=sha-<40-hex>
docker compose -f .docker/docker-compose.dokploy.yml --env-file /root/casadana-demo.env pull api web
docker compose -f .docker/docker-compose.dokploy.yml --env-file /root/casadana-demo.env up -d
```

To roll back, do the same with the previous sha — the images are still in GHCR, so nothing
needs rebuilding. Then `echo sha-<that> > /root/.casadana-deployed-tag` so the next automatic
rollback targets the right baseline.

Rolling back **code** is a normal `git revert` + merge; the pipeline redeploys.

## Traps

- **`deploy` also moves the host checkout.** `git reset --hard <sha>` runs against
  `/root/src/casadana`, because the compose file has to match the images. Do not keep
  uncommitted work there — the script refuses to deploy rather than discard it, so a stray
  edit blocks the pipeline until cleared.
- **A schema migration is not rolled back.** `MIGRATE_ON_BOOT=true` applies embedded
  migrations on API start, and reverting to an older image does *not* undo them. Any
  destructive migration needs its own plan; the automatic rollback only restores containers.
- **Postgres is in the same compose project.** `docker compose down -v` destroys the v2
  database. The deploy path never calls it; do not add it.
- **The health probe uses loopback, not the public URL.** A DNS or TLS fault must not be
  misreported as a bad build. Check the public URL separately after deploying.
- **`concurrency: cancel-in-progress: false`** on the deploy job is deliberate. Cancelling a
  deploy midway could leave containers half-updated. The build job cancels freely; the deploy
  queues.
- **Rotating the deploy key** means regenerating it, replacing the `authorized_keys` line and
  re-running `gh secret set VPS_SSH_KEY`. The old key keeps working until removed from
  `authorized_keys`.
- Unrelated but adjacent: this host has `PermitRootLogin yes` **and**
  `PasswordAuthentication yes`. The deploy key is not what exposes the box — root password
  SSH does. Worth turning off independently of this pipeline.
