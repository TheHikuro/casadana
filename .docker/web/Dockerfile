FROM oven/bun:1-alpine AS builder

WORKDIR /app

COPY package.json bun.lock ./
COPY packages ./packages
COPY apps/web ./apps/web
COPY .docker/web/nginx.conf ./nginx.conf

RUN bun install --frozen-lockfile

# VITE_API_BASE_URL must be EMPTY: nginx serves the SPA and proxies /api on the same
# origin (docs/adr/0003-same-origin-api-routing.md).
#
# It is set explicitly here rather than left unset, which is not the same thing. bun
# auto-loads apps/web/.env.development into process.env whenever NODE_ENV is unset --
# and it is unset in this stage -- and Vite deliberately gives process.env priority
# over .env files. So the "unset -> empty string fallback" in packages/api/src/client.ts
# never ran: every production bundle was built with
#   baseURL: "http://localhost:8080"
# which made the browser call localhost instead of the same origin, producing a CORS
# (and mixed-content) failure on https://demo.casa-dana.com. Observed 2026-08-10.
#
# bun does not override a variable that is already present, so this wins. .env.development
# is also excluded via .dockerignore; both are kept as independent safeguards.
RUN VITE_API_BASE_URL="" bun --cwd apps/web build

FROM nginx:alpine

COPY --from=builder /app/apps/web/dist /usr/share/nginx/html
COPY --from=builder /app/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
