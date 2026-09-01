# Architecture Decision Records

Short documents recording decisions that are **expensive or dangerous to rediscover** — especially ones whose reasons live outside this repository (production host state, DNS, certificates, the legacy .NET app).

Format: `NNNN-kebab-case-title.md`, with Context / Decision / Consequences. Records are append-only. A decision that is later reversed gets a new ADR that supersedes the old one; the old file stays, marked `Superseded by NNNN`.

| # | Title | Status |
|---|---|---|
| [0001](0001-v2-rewrite-and-v1-decommission.md) | v2 rewrite, and how v1 gets decommissioned | Accepted — data-migration part superseded by [0005](0005-v1-data-is-imported-after-all.md) |
| [0002](0002-nginx-fronts-dokploy-traefik-disabled.md) | nginx fronts 80/443; Dokploy's Traefik stays parked | Accepted |
| [0003](0003-same-origin-api-routing.md) | Web and API must share one origin | Accepted |
| [0004](0004-github-actions-owns-deploys-dokploy-observes.md) | GitHub Actions owns deploys; Dokploy is an observation surface | Accepted |
| [0005](0005-v1-data-is-imported-after-all.md) | v1 booking data is imported after all | Accepted — supersedes part of [0001](0001-v2-rewrite-and-v1-decommission.md) |

## Runbooks

ADRs record *why*. Step-by-step *how* lives in [`docs/runbooks/`](../runbooks/):

- [`deploy-demo-subdomain.md`](../runbooks/deploy-demo-subdomain.md) — bring v2 up on
  `demo.casa-dana.com` while v1 keeps serving.
- [`continuous-deployment.md`](../runbooks/continuous-deployment.md) — how a merge reaches the
  VPS, and how to roll back.
- [`environments.md`](../runbooks/environments.md) — prod vs demo: branches, ports, databases,
  and what must never be shared between them.
- [`dokploy-panel-subdomain.md`](../runbooks/dokploy-panel-subdomain.md) — expose the Dokploy
  UI on `dokploy.casa-dana.com`, and the exposure that buys.
- [`v1-data-import.md`](../runbooks/v1-data-import.md) — move v1's booking history into v2,
  before Phase 5 destroys the source.

## Why these exist

The v2 stack replaces a .NET 9 API that has been in production since February 2025 on a single VPS. Several facts needed to shut that API down safely — where its encryption keys live, what its TLS renewal is bound to, how its data gets out — are discoverable only by inspecting the running server, and will vanish the moment it is deleted. ADR 0001 is the record that makes the deletion safe.
