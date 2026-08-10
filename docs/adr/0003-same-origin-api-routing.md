# 0003 — Web and API must share one origin

- **Status:** Accepted
- **Date:** 2026-08-10

## Context

The frontend resolves its API base URL like this:

```ts
// packages/api/src/client.ts
baseURL: (typeof import.meta !== "undefined" && (import.meta as any).env?.VITE_API_BASE_URL) || "",
withCredentials: true,
```

Two consequences follow from the build, not from configuration:

1. **`.docker/web/Dockerfile` declares no `ARG`/`ENV`.** Vite inlines env vars at build time, so with nothing passed, `VITE_API_BASE_URL` is undefined and `baseURL` compiles to `""` — every request goes to a **relative** path such as `/api/bookings`, resolved against whatever origin served the page.
2. **`.docker/web/nginx.conf` does not proxy `/api`.** It only serves static files with an SPA fallback (`try_files $uri $uri/ /index.html`). It has no `proxy_pass` and no upstream.

So the web container cannot reach the API by itself, and the browser will send API requests to the page's own origin. Auth is cookie-based (`withCredentials: true`, plus `COOKIE_SECURE` on the API), which same-origin satisfies without `SameSite=None` or a CORS allow-list.

## Decision

**Whatever serves the frontend must also route `/api/*` to the API, under the same scheme and hostname.** The reverse proxy in front is responsible for the split — there is no fallback if it is missing.

Current implementation ([0002](0002-nginx-fronts-dokploy-traefik-disabled.md)), in `demo-casadana.conf`:

```nginx
location /api/ { proxy_pass http://127.0.0.1:8080; }   # NO trailing slash
location /     { proxy_pass http://127.0.0.1:3001; }
```

The repo's `Caddyfile` expresses the same shape for the eventual production topology:

```
casa-dana.com {
    reverse_proxy /api/* api:8080
    reverse_proxy /* web:80
}
```

### Two traps

**No trailing slash on `proxy_pass`.** The Go API serves its routes *under* `/api` — the health check is `/api/health` (see the healthcheck in `.docker/docker-compose.yml`). Writing `proxy_pass http://127.0.0.1:8080/;` makes nginx strip the `/api` prefix, and **every** API route 404s. Without the slash, the original URI is passed through intact.

**Do not set `VITE_API_BASE_URL`.** The empty fallback is load-bearing. Setting it to an absolute URL (e.g. `https://api.casa-dana.com`) makes every request cross-origin and drags back the `SameSite`/CORS/preflight problems that v1 had, for no benefit. It exists only as an escape hatch for local development against a remote API.

## Rules for future changes

- Any new deploy target — Traefik after [0002](0002-nginx-fronts-dokploy-traefik-disabled.md)'s exit condition, a preview environment, a CDN — must implement the same two-route split. Pointing a hostname at the web container alone yields a site that renders and then fails every request.
- **This constrains the planned Rust rewrite of the API.** Any replacement must keep serving under the `/api` prefix and keep `/api/health` (the container healthcheck depends on it). Changing the prefix means changing the proxy config and the healthcheck together.
- If the frontend ever needs to be hosted separately (Vercel, static CDN) while the API stays here, then `VITE_API_BASE_URL` must be added as a build `ARG` in `.docker/web/Dockerfile` *and* the API needs an explicit CORS allow-list *and* cookies need `SameSite=None; Secure`. That is a deliberate architecture change — write a new ADR superseding this one rather than patching around it.
- Adding `/api` proxying inside `.docker/web/nginx.conf` is an alternative (it would make the web container self-sufficient), but it hard-codes the API's address into the web image and breaks the dev setup where the API is reached differently. Not chosen.

## Consequences

- The web image is environment-independent: the same artefact works on `demo.casa-dana.com` and `casa-dana.com` with no rebuild, because it embeds no hostname. This is what lets CI build once and Dokploy deploy the same digest everywhere.
- The reverse proxy becomes load-bearing for correctness, not just TLS. A misordered or trailing-slashed `location` block is a total outage of API functionality while the site still appears to load.
- Cookie auth needs `X-Forwarded-Proto` set by the proxy so the API marks cookies `Secure` correctly behind TLS termination.
