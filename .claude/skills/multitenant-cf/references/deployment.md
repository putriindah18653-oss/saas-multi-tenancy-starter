# Deployment — VPS (Golang) + Cloudflare Workers (Vue)

## Overview

```
VPS Ubuntu
├── Nginx (reverse proxy + SSL)
├── Option A: Golang API + Worker as systemd services, PostgreSQL + Redis local/managed
└── Option B: Docker Compose for API + Worker + PostgreSQL + Redis

Cloudflare
├── Worker PUBLIC (vue-public) — server-render (SSR) tenant sites + proxy public API
└── Worker DASHBOARD (vue-dashboard) — serve the Vue SPA + auth proxy + API proxy
```

---

## Option B: Docker Compose Backend Stack

Use this when you want the VPS backend, database, and Redis managed as one container stack. This is valid for a single-instance VPS, but treat it as a production database deployment: persistent volumes, backups, upgrades, and secrets matter.

Container layout:

```text
Docker host / VPS
├── nginx / cloudflared on the host, or a separate edge container
├── api          → ./cmd/api
├── worker       → ./cmd/worker, includes libvips-tools for AVIF conversion
├── postgres     → persistent volume, single database yourdb
├── redis        → persistent volume, asynq broker/cache
└── migrator     → one-shot migration job, uses owner/migrator DB role only
```

### Dockerfile — API + Worker Targets

```dockerfile
# Dockerfile
# syntax=docker/dockerfile:1.7

FROM golang:1.23-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# If internal/db is not committed, install sqlc and run sqlc generate here instead.
# Preferred: commit generated sqlc output and verify it in CI.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM debian:bookworm-slim AS api
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata wget \
  && rm -rf /var/lib/apt/lists/*
RUN useradd --system --uid 10001 --create-home app
WORKDIR /app
COPY --from=build /out/api /app/api
USER app
EXPOSE 8080 9090
ENTRYPOINT ["/app/api"]

FROM debian:bookworm-slim AS worker
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata libvips-tools \
  && rm -rf /var/lib/apt/lists/*
RUN useradd --system --uid 10001 --create-home app
WORKDIR /app
COPY --from=build /out/worker /app/worker
USER app
ENTRYPOINT ["/app/worker"]
```

Why two runtime targets: the API stays small, while the worker image carries `libvips-tools` because media conversion shells out to the `vips` CLI.

### `.dockerignore`

```dockerignore
.git
bin
dist
coverage
node_modules
.tmp
.env
.env.*
*.log
```

### Database Bootstrap

Postgres' `/docker-entrypoint-initdb.d` scripts run only when the database volume is first created. Use this only for base roles/database creation. Schema migrations still run separately via the `migrator` service.

```bash
#!/usr/bin/env bash
# docker/postgres/001-init-roles.sh
# Runs as POSTGRES_USER on first volume initialization only.
set -euo pipefail

: "${APP_DB_PASSWORD:?APP_DB_PASSWORD is required}"
: "${PLATFORM_DB_PASSWORD:?PLATFORM_DB_PASSWORD is required}"
: "${MIGRATOR_DB_PASSWORD:?MIGRATOR_DB_PASSWORD is required}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<'SQL'
CREATE DATABASE yourdb;
SQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname yourdb \
  -v app_password="$APP_DB_PASSWORD" \
  -v platform_password="$PLATFORM_DB_PASSWORD" \
  -v migrator_password="$MIGRATOR_DB_PASSWORD" <<'SQL'
CREATE ROLE app_user LOGIN PASSWORD :'app_password' NOSUPERUSER NOBYPASSRLS;
CREATE ROLE platform_user LOGIN PASSWORD :'platform_password' NOSUPERUSER BYPASSRLS;
CREATE ROLE migrator_user LOGIN PASSWORD :'migrator_password' NOSUPERUSER NOBYPASSRLS;

GRANT CONNECT ON DATABASE yourdb TO app_user, platform_user, migrator_user;
GRANT CREATE, USAGE ON SCHEMA public TO migrator_user;
GRANT USAGE ON SCHEMA public TO app_user, platform_user;

-- After migrations create tables, migration SQL must grant table privileges to app/platform roles:
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO platform_user;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO platform_user;
SQL
```

> Keep RLS rules from `database-schema.md`: migrations must not run as `app_user`; tenant handlers use `app_user`; platform cross-tenant handlers use `platform_user`; every tenant table must `ENABLE` and `FORCE ROW LEVEL SECURITY` with `USING` + `WITH CHECK` policies.

### `docker-compose.yml`

```yaml
services:
  postgres:
    image: postgres:16-bookworm
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD}
      POSTGRES_DB: postgres
      APP_DB_PASSWORD: ${APP_DB_PASSWORD:?set APP_DB_PASSWORD}
      PLATFORM_DB_PASSWORD: ${PLATFORM_DB_PASSWORD:?set PLATFORM_DB_PASSWORD}
      MIGRATOR_DB_PASSWORD: ${MIGRATOR_DB_PASSWORD:?set MIGRATOR_DB_PASSWORD}
      TZ: Asia/Jakarta
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./docker/postgres:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d yourdb"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks: [backend]

  redis:
    image: redis:7-bookworm
    restart: unless-stopped
    command: ["redis-server", "--appendonly", "yes", "--requirepass", "${REDIS_PASSWORD:?set REDIS_PASSWORD}"]
    environment:
      REDIS_PASSWORD: ${REDIS_PASSWORD:?set REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a $$REDIS_PASSWORD ping | grep PONG"]
      interval: 10s
      timeout: 5s
      retries: 10
    networks: [backend]

  migrator:
    image: migrate/migrate:v4.17.1
    profiles: ["migrate"]
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./migrations:/migrations:ro
    command:
      [
        "-path", "/migrations",
        "-database", "postgres://migrator_user:${MIGRATOR_DB_PASSWORD}@postgres:5432/yourdb?sslmode=disable",
        "up"
      ]
    networks: [backend]

  api:
    build:
      context: .
      target: api
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      APP_ENV: production
      PORT: 8080
      INTERNAL_PORT: 9090
      DB_URL: postgres://app_user:${APP_DB_PASSWORD}@postgres:5432/yourdb?sslmode=disable
      PLATFORM_DB_URL: postgres://platform_user:${PLATFORM_DB_PASSWORD}@postgres:5432/yourdb?sslmode=disable
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379/0
      JWT_SECRET: ${JWT_SECRET:?set JWT_SECRET}
      ACCESS_TTL: 15m
      REFRESH_IDLE_TTL: 12h
      REFRESH_ABSOLUTE_TTL: 24h
      INTERNAL_SSO_SECRET_CURRENT: ${INTERNAL_SSO_SECRET_CURRENT:?set INTERNAL_SSO_SECRET_CURRENT}
      INTERNAL_SSO_SECRET_PREVIOUS: ${INTERNAL_SSO_SECRET_PREVIOUS:-}
      CF_ACCOUNT_ID: ${CF_ACCOUNT_ID}
      CF_API_TOKEN: ${CF_API_TOKEN}
      CF_KV_NAMESPACE_ID: ${CF_KV_NAMESPACE_ID}
      S3_ENDPOINT: ${S3_ENDPOINT}
      S3_REGION: ${S3_REGION:-auto}
      S3_ACCESS_KEY: ${S3_ACCESS_KEY}
      S3_SECRET_KEY: ${S3_SECRET_KEY}
      MEDIA_BUCKET_PRIVATE: ${MEDIA_BUCKET_PRIVATE}
      MEDIA_BUCKET_PUBLIC: ${MEDIA_BUCKET_PUBLIC}
      MEDIA_PUBLIC_BASE_URL: ${MEDIA_PUBLIC_BASE_URL}
      BILLING_CURRENCY: IDR
      MIDTRANS_SERVER_KEY: ${MIDTRANS_SERVER_KEY}
      MIDTRANS_CLIENT_KEY: ${MIDTRANS_CLIENT_KEY}
      DUITKU_MERCHANT_CODE: ${DUITKU_MERCHANT_CODE}
      DUITKU_API_KEY: ${DUITKU_API_KEY}
    ports:
      - "127.0.0.1:8080:8080" # public API behind Nginx/Cloudflare only
      - "127.0.0.1:9090:9090" # internal listener; preferably expose only to cloudflared
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8080/healthz || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 10
    networks: [backend]

  worker:
    build:
      context: .
      target: worker
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      APP_ENV: production
      DB_URL: postgres://app_user:${APP_DB_PASSWORD}@postgres:5432/yourdb?sslmode=disable
      PLATFORM_DB_URL: postgres://platform_user:${PLATFORM_DB_PASSWORD}@postgres:5432/yourdb?sslmode=disable
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379/0
      S3_ENDPOINT: ${S3_ENDPOINT}
      S3_REGION: ${S3_REGION:-auto}
      S3_ACCESS_KEY: ${S3_ACCESS_KEY}
      S3_SECRET_KEY: ${S3_SECRET_KEY}
      MEDIA_BUCKET_PRIVATE: ${MEDIA_BUCKET_PRIVATE}
      MEDIA_BUCKET_PUBLIC: ${MEDIA_BUCKET_PUBLIC}
      MEDIA_PUBLIC_BASE_URL: ${MEDIA_PUBLIC_BASE_URL}
      CF_ACCOUNT_ID: ${CF_ACCOUNT_ID}
      CF_API_TOKEN: ${CF_API_TOKEN}
      CF_KV_NAMESPACE_ID: ${CF_KV_NAMESPACE_ID}
    networks: [backend]

networks:
  backend:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
```

> The compose file binds API ports to `127.0.0.1`, not `0.0.0.0`. Put Nginx, Caddy, or cloudflared in front. Do not publish Postgres/Redis ports publicly.

### `.env` Example

```dotenv
POSTGRES_PASSWORD=change-me-superuser-bootstrap-only
APP_DB_PASSWORD=app_password
PLATFORM_DB_PASSWORD=platform_password
MIGRATOR_DB_PASSWORD=migrator_password
REDIS_PASSWORD=change-me-redis
JWT_SECRET=change-me-long-random
INTERNAL_SSO_SECRET_CURRENT=change-me-long-random
INTERNAL_SSO_SECRET_PREVIOUS=

CF_ACCOUNT_ID=
CF_API_TOKEN=
CF_KV_NAMESPACE_ID=

S3_ENDPOINT=https://<acct>.r2.cloudflarestorage.com
S3_REGION=auto
S3_ACCESS_KEY=
S3_SECRET_KEY=
MEDIA_BUCKET_PRIVATE=yourapp-originals
MEDIA_BUCKET_PUBLIC=yourapp-media
MEDIA_PUBLIC_BASE_URL=https://cdn.yourdomain.com

MIDTRANS_SERVER_KEY=
MIDTRANS_CLIENT_KEY=
DUITKU_MERCHANT_CODE=
DUITKU_API_KEY=
```

For production, prefer Docker secrets or an external secret manager over a flat `.env` file. If `.env` is used, permissions should be `chmod 600 .env` and it must never be committed.

### Operations

```bash
# Build images
sudo docker compose build

# Start database + redis + app stack
sudo docker compose up -d postgres redis

# Run migrations explicitly, as migrator_user, before app startup/release cutover
sudo docker compose --profile migrate run --rm migrator

# Start or update API + worker
sudo docker compose up -d api worker

# Logs
sudo docker compose logs -f api worker

# Verify vips exists in the worker image
sudo docker compose run --rm --entrypoint vips worker --version
```

### Backup / Restore Notes

- Back up `postgres_data` with logical dumps or PITR, not by copying a live volume.
- Use a dedicated `backup_user` as described later in this document; never dump as `app_user`.
- Redis AOF is useful for restarts, but queues should still be treated as operational state; critical durable facts belong in Postgres.
- Test restore into a fresh Docker volume before trusting backups.

### When Not to Use This Stack

- If the database needs managed backups, HA, PITR, and point-in-time upgrades without VPS ops burden, use managed Postgres instead of the compose `postgres` service.
- If image conversion consumes too much CPU/RAM, move `worker` to a separate host or scale it independently.
- If strict isolation is required, do not publish the internal listener directly; reach it through Cloudflare Tunnel or a private network only.

---

## Option A: VPS Golang API via Systemd

### Build & Binary
```bash
# On the dev/CI machine
# 1. Generate code from SQL (if internal/db is not committed, run before build)
sqlc generate

# 2. Build both binaries — the API and the asynq worker (AVIF conversion, see media-upload.md)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/api    ./cmd/api
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/worker ./cmd/worker

# Copy to VPS
scp bin/api bin/worker user@vps:/opt/yourapp/
```

> **`CGO_ENABLED=0` stays even with AVIF.** The worker converts images by shelling out to the `vips` CLI (a subprocess), not by linking libvips into the Go binary — so both binaries remain static. The trade-off is an OS-level dependency: `vips` must be installed on the worker host (below).

```bash
# Install libvips (provides the `vips` CLI) on the VPS — needed by the worker only.
# Ensure the build includes AVIF (libaom/libheif); Ubuntu 24.04's libvips-tools does.
sudo apt-get update && sudo apt-get install -y libvips-tools
vips --version && vips --vips-config | grep -i heif   # confirm AVIF/HEIF support
```

> **sqlc generated code (`internal/db/`):** recommendation — commit the generated output to the repo. Build & CI then do not need the `sqlc` binary, and codegen diffs get reviewed too. The consequence: whenever you change `query/*.sql` or the schema, run `sqlc generate` then re-commit. Add a `sqlc diff` check in CI to ensure the generated code is not stale.

### Systemd Service
```ini
# /etc/systemd/system/yourapp-api.service
[Unit]
Description=YourApp API
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=yourapp
WorkingDirectory=/opt/yourapp
ExecStart=/opt/yourapp/api
Restart=always
RestartSec=5

# Environment variables
Environment=APP_ENV=production
Environment=PORT=8080
# Two pools to ONE database (yourdb), separated only by role (see database-schema.md):
#   app_user      → NOBYPASSRLS, used by all tenant handlers (subject to FORCE RLS)
#   platform_user → BYPASSRLS, ONLY for cross-tenant platform routes
# Note the identical /yourdb in both URLs — single DB, role is the only difference.
Environment=DB_URL=postgres://app_user:pass@localhost:5432/yourdb?sslmode=disable
Environment=PLATFORM_DB_URL=postgres://platform_user:pass@localhost:5432/yourdb?sslmode=disable
Environment=REDIS_URL=redis://localhost:6379
Environment=JWT_SECRET=your-secret-key
# Token lifetimes (see auth-sso.md). High-security profile:
#   ACCESS_TTL          = short bearer credential; renewed silently by the Worker
#   REFRESH_IDLE_TTL    = sliding window; each rotation resets it (active user stays in)
#   REFRESH_ABSOLUTE_TTL= hard cap from original login; forces periodic re-auth
Environment=ACCESS_TTL=15m
Environment=REFRESH_IDLE_TTL=12h
Environment=REFRESH_ABSOLUTE_TTL=24h
# INTERNAL_SSO_SECRET dual-key rotation (see auth-sso.md: validInternalSecret).
# Backend accepts both during a rotation window: promote new→CURRENT, keep old→PREVIOUS,
# redeploy the Worker with the new value, then clear PREVIOUS on the next release.
Environment=INTERNAL_SSO_SECRET_CURRENT=your-internal-secret
Environment=INTERNAL_SSO_SECRET_PREVIOUS=
Environment=CF_ACCOUNT_ID=xxx
Environment=CF_API_TOKEN=xxx
Environment=CF_KV_NAMESPACE_ID=xxx
# Object storage for media (see media-upload.md). S3-compatible: R2 and S3 differ only here.
#   R2:  S3_ENDPOINT=https://<acct>.r2.cloudflarestorage.com  S3_REGION=auto
#   AWS: S3_ENDPOINT=  (leave empty)                          S3_REGION=us-east-1
Environment=S3_ENDPOINT=https://<acct>.r2.cloudflarestorage.com
Environment=S3_REGION=auto
Environment=S3_ACCESS_KEY=xxx
Environment=S3_SECRET_KEY=xxx
Environment=MEDIA_BUCKET_PRIVATE=yourapp-originals          # raw uploads, never public
Environment=MEDIA_BUCKET_PUBLIC=yourapp-media               # stripped AVIF variants
Environment=MEDIA_PUBLIC_BASE_URL=https://cdn.yourdomain.com # custom domain for the public bucket
# Payment gateways (see billing.md). Server keys gate webhook signature verification —
# keep secret, never log. Webhook endpoints are PUBLIC (gateway servers call them) but
# signature-authenticated at the backend, NOT IP-gated like /api/internal/*.
Environment=BILLING_CURRENCY=IDR
Environment=MIDTRANS_SERVER_KEY=xxx
Environment=MIDTRANS_CLIENT_KEY=xxx
Environment=DUITKU_MERCHANT_CODE=xxx
Environment=DUITKU_API_KEY=xxx
# Migrations are run separately (CI/deploy) as the migrator/owner role, NOT app_user,
# so that app_user never becomes the table owner. See database-schema.md.

# Security
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable yourapp-api
sudo systemctl start yourapp-api
sudo systemctl status yourapp-api
```

### Systemd Service — Worker (asynq, AVIF conversion)

A second unit runs the `cmd/worker` process that consumes asynq jobs and shells out to `vips` (see `media-upload.md`). Same environment as the API (it needs DB, Redis, and the `S3_*`/`MEDIA_*` vars), but a different binary. Keep it separate so image conversion (CPU-heavy) never blocks API request handling, and so you can scale/restart it independently.

```ini
# /etc/systemd/system/yourapp-worker.service
[Unit]
Description=YourApp Worker (asynq)
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=yourapp
WorkingDirectory=/opt/yourapp
ExecStart=/opt/yourapp/worker
Restart=always
RestartSec=5

# Same EnvironmentFile as the API (DB_URL, REDIS_URL, S3_*, MEDIA_*, ...).
# Prefer an EnvironmentFile over duplicating Environment= lines across both units.
EnvironmentFile=/etc/yourapp/env

# Security — vips writes temp files; PrivateTmp gives the worker an isolated /tmp.
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable yourapp-worker
sudo systemctl start yourapp-worker
sudo systemctl status yourapp-worker
```

> **Bucket setup (one-time).** Create two buckets: a **private** one for originals (no public access; **keep objects permanently and enable object versioning** — they are the authoritative copy and a backup artifact, see `backup-recovery.md`) and a **public** one for variants, fronted by a custom domain (`MEDIA_PUBLIC_BASE_URL`) so served AVIFs are cacheable at the CDN edge. On R2, "public" = attach a custom domain / enable public access on that bucket only.

### Nginx Config
```nginx
# /etc/nginx/sites-available/yourapp-api
server {
    listen 80;
    server_name api.yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name api.yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;

    # Defense-in-depth only — NOT the security boundary. These ranges are shared by
    # every Cloudflare customer, so any CF Worker can reach the origin from here.
    # The real boundary is JWT signature verification in the backend (see auth-sso.md).
    # Only accept requests from Cloudflare IPs
    # https://www.cloudflare.com/ips/
    allow 103.21.244.0/22;
    allow 103.22.200.0/22;
    allow 103.31.4.0/22;
    allow 104.16.0.0/13;
    allow 104.24.0.0/14;
    allow 108.162.192.0/18;
    allow 131.0.72.0/22;
    allow 141.101.64.0/18;
    allow 162.158.0.0/15;
    allow 172.64.0.0/13;
    allow 173.245.48.0/20;
    allow 188.114.96.0/20;
    allow 190.93.240.0/20;
    allow 197.234.240.0/22;
    allow 198.41.128.0/17;
    deny all;

    # /api/internal/* is NOT reachable from the public origin. It is served only on the
    # internal listener below, reached by the Worker via Cloudflare Tunnel. Block it here
    # so a leaked X-Internal-Secret cannot be replayed against the public endpoint.
    location /api/internal/ {
        return 404;
    }

    # Access log includes the edge request id + real client IP so nginx logs line up with
    # the Worker and the Go app's structured logs (all keyed by the same X-Request-ID).
    # Define this log_format once in the http{} block:
    #   log_format obs '$remote_addr $request_method $uri $status ${request_time}s '
    #                  'rid=$http_x_request_id ip=$http_cf_connecting_ip';
    # NEVER add $http_authorization / $http_cookie / $http_x_internal_secret to the format.
    access_log /var/log/nginx/yourapp.log obs;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $http_cf_connecting_ip;  # the user's real IP from CF
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Request-ID $http_x_request_id;   # pass the edge id to the Go app
    }
}
```

### Internal Listener — `/api/internal/*` via Cloudflare Tunnel

The SSO internal endpoint (`InternalToken`, see `auth-sso.md`) must not be exposed on the public origin. Serve it on a separate port reachable only through a Cloudflare Tunnel (`cloudflared`), so the path from Worker → backend never touches the public internet.

```nginx
# Separate internal vhost — bound to localhost, fronted by cloudflared (not public DNS).
server {
    listen 127.0.0.1:8443 ssl;
    server_name internal.yourapp.local;

    ssl_certificate     /etc/yourapp/internal.crt;
    ssl_certificate_key /etc/yourapp/internal.key;

    # Only /api/internal/* lives here; everything else 404s.
    location /api/internal/ {
        proxy_pass http://127.0.0.1:8080;   # same Go process, internal route group
        proxy_set_header Host $host;
    }
    location / { return 404; }
}
```

```yaml
# ~/.cloudflared/config.yml — Tunnel maps a hostname to the internal listener.
# The Worker calls https://sso-internal.portalonline.id/api/internal/token;
# only cloudflared (authenticated to your CF account) can route to it.
tunnel: <tunnel-id>
credentials-file: /etc/cloudflared/<tunnel-id>.json
ingress:
  - hostname: sso-internal.portalonline.id
    service: https://127.0.0.1:8443
  - service: http_status:404
```

> **Layered:** the `X-Internal-Secret` (constant-time check, see `auth-sso.md`) is the authentication; the Tunnel + localhost binding is the network isolation. A leaked secret alone is not exploitable because the endpoint has no public route; an open network path alone is not exploitable because the secret still gates it.

```bash
# SSL with certbot (public origin only)
sudo certbot --nginx -d api.yourdomain.com

# Enable the site
sudo ln -s /etc/nginx/sites-available/yourapp-api /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

---

## Cloudflare Workers: Deployment

### CI/CD with GitHub Actions

Deployment workflows must run after the required test/quality gates in `testing.md`. Do not deploy from a workflow that only builds; backend, frontend, Worker, RLS isolation, and E2E gates belong in the test workflow first.

```yaml
# .github/workflows/deploy-dashboard.yml
name: Deploy Dashboard Worker

on:
  push:
    branches: [main]
    paths: ['vue-dashboard/**']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: vue-dashboard/package-lock.json

      - name: Install dependencies
        working-directory: vue-dashboard
        run: npm ci

      - name: Build
        working-directory: vue-dashboard
        run: npm run build
        env:
          VITE_APP_ENV: production

      - name: Deploy to Cloudflare Workers
        working-directory: vue-dashboard
        run: npx wrangler deploy
        env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CF_API_TOKEN }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CF_ACCOUNT_ID }}
```

### Deploy Manual
```bash
cd vue-dashboard
npm run build
npx wrangler deploy

# Set secret (not stored in wrangler.jsonc)
npx wrangler secret put INTERNAL_SSO_SECRET
```

### Environments (staging vs production)
```jsonc
// wrangler.jsonc
{
  "name": "vue-dashboard",
  "env": {
    "staging": {
      "name": "vue-dashboard-staging",
      "vars": {
        "BACKEND_URL": "https://api-staging.yourdomain.com",
        "ENVIRONMENT": "staging"
      },
      "kv_namespaces": [
        { "binding": "CACHE_KV",  "id": "staging-kv-id" },
        { "binding": "CACHE_SSO", "id": "staging-sso-id" }
      ]
    },
    "production": {
      "vars": {
        "BACKEND_URL": "https://api.yourdomain.com",
        "ENVIRONMENT": "production"
      }
    }
  }
}
```

```bash
# Deploy to staging
npx wrangler deploy --env staging

# Deploy to production
npx wrangler deploy --env production
```

---

## New Tenant Registration: Cloudflare-side Mechanics

> **The onboarding orchestration lives in `tenant-onboarding.md`** (the canonical flow: validation → transactional PG core → async provisioning → trial). This section documents only the **Cloudflare-side helpers** that the async provisioning job calls — the infra mechanics, not the flow. Do not re-implement the orchestration here.

The async provisioning step (`tenant-onboarding.md` → step 2) performs these CF API calls. Each is **idempotent** so the job can retry safely:

```go
// Called by the provisioning worker (NOT synchronously in the signup request).
// (a) Write tenant config to KV — PUT overwrites, so re-runs are harmless.
func (s *CFService) WriteTenantConfig(ctx context.Context, t *Tenant) error {
    config := TenantConfig{
        TenantID:       t.ID,
        TenantName:     t.Name,
        Plan:           t.Plan,
        HasCustomDomain: t.HasCustomDomain,
        PublicTenantAPIKey: generateAPIKey(),
        Branding:       defaultBranding(),
    }
    // Set both the public and the dashboard domain keys.
    if err := s.SetKV(ctx, "CACHE_KV", "tenant:"+t.PublicDomain, config); err != nil { return err }
    return s.SetKV(ctx, "CACHE_KV", "tenant:"+t.DashboardDomain, config)
}

// (b) Register a custom domain on the Workers — only if the tenant brought its own domain.
//     EnsureDomain = check-then-add, so adding an already-registered domain is a no-op.
func (s *CFService) EnsureDomain(ctx context.Context, worker, domain string) error {
    // GET existing domains for `worker`; if `domain` already attached → return nil; else PUT it.
}
```

The owner domain (`{slug}.portalonline.id`) needs no CF registration — it's a wildcard route already on the Worker; only a tenant's **own** custom domain requires `EnsureDomain`.

---


## Health Check & Monitoring

### Golang Health and Readiness Endpoints

Keep the public health endpoint coarse and safe; put dependency details on the internal readiness endpoint (`metrics-alerting.md`). This avoids leaking topology and prevents public uptime checks from learning whether DB or Redis failed.

```go
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
    if err := h.db.Ping(r.Context()); err != nil {
        httperr.Write(w, httperr.Unavailable("dependency unavailable"))
        return
    }
    if err := h.redis.Ping(r.Context()).Err(); err != nil {
        httperr.Write(w, httperr.Unavailable("dependency unavailable"))
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
```

- Public `/healthz`: process alive, no dependency names.
- Internal `/readyz`: dependency checks, reachable only from private listener/orchestrator.
- API errors use the standard envelope; public probes do not receive raw internals.

### Uptime Monitoring
- Use **Cloudflare Health Checks** (free, in the Cloudflare dashboard)
- Or UptimeRobot / Better Uptime for external monitoring
- Alert to Telegram/Slack via webhook

### Logs — Two Streams, Two Destinations

See `observability.md` for the structured-logging foundation. Operationally:

- **Operational logs** (JSON to stdout from the API + worker) → captured by `journald` → ship to an aggregator (Loki / CloudWatch / etc.). These are diagnostic and high-volume; set a short-ish retention (e.g. 14–30 days). Every line carries `request_id` + `tenant_id` + `user_id`, so you can filter an incident by tenant or trace one request end to end.
- **Audit log** lives in Postgres (`audit_log`, RLS + append-only) — NOT in the log aggregator. It is a security record: longer retention, captured by the PostgreSQL backups (`backup-recovery.md`), shown to tenant admins. Prune it with the migrator/maintenance role, never `app_user`.
- The two are bridged by `request_id`: an audit entry → the operational logs of that exact request.

### Metrics & Alerts

Metrics and alerting are part of the production baseline; see `metrics-alerting.md`. Serve `/metrics` on the **internal listener** (the Cloudflare Tunnel vhost above, same pattern as `/api/internal/*`), never on the public origin — it exposes internal topology and per-tenant volume. Deployment should include scraping/agent config, Grafana dashboards, and alert rules for API 5xx/latency, DB/Redis availability, worker queue backlog, billing webhook failures, stuck tenant provisioning, and backup freshness.

### Backups (operational setup)

Strategy, restore procedures, and per-tenant export are in `backup-recovery.md`. The VPS-side wiring:

```sql
-- Dedicated backup role — NEVER app_user. Read-all for logical dumps; REPLICATION for PITR.
-- The app's runtime credentials must not be able to read or delete backups.
CREATE ROLE backup_user LOGIN PASSWORD 'xxx' REPLICATION;
GRANT pg_read_all_data TO backup_user;   -- PG 14+: read every table for pg_dump
```

```ini
# postgresql.conf — WAL archiving for PITR (pgBackRest). See backup-recovery.md for the repo config.
archive_mode = on
archive_command = 'pgbackrest --stanza=yourdb archive-push %p'
wal_level = replica
```

```ini
# /etc/systemd/system/yourapp-backup.timer  — nightly encrypted logical dump → off-site
[Timer]
OnCalendar=*-*-* 03:17:00     # off-peak, off the :00 mark
Persistent=true
[Install]
WantedBy=timers.target
```
```ini
# /etc/systemd/system/yourapp-backup.service  (oneshot run by the timer)
[Service]
Type=oneshot
User=yourapp
EnvironmentFile=/etc/yourapp/backup.env   # BACKUP_DSN (backup_user), BACKUP_AGE_PUBKEY, S3_*
ExecStart=/opt/yourapp/scripts/pg-logical-dump.sh   # pg_dump | age encrypt | ship off-site
PrivateTmp=true
```

> **Off-site is non-negotiable.** Both the pgBackRest repo and the nightly dump must land on storage **separate from the VPS** (different provider/account) — a backup on the same box dies with the box. Encrypt everything (pgBackRest `aes-256-cbc`; `age`/`gpg` for dumps), and run the scheduled **restore drill** from `backup-recovery.md` — an untested backup is not a backup.

---

## Simple Restart Deploy Golang

This single-VPS systemd flow is a short restart deploy, not true zero-downtime. For zero-downtime you need at least two app instances behind a load balancer or socket activation with readiness-aware draining.

```bash
#!/bin/bash
# deploy.sh — run from the build machine
set -euo pipefail

APP_DIR=/opt/yourapp
RELEASE=$(date +%Y%m%d%H%M%S)

scp bin/api user@vps:/tmp/api.$RELEASE
ssh user@vps "
  set -euo pipefail
  sudo mkdir -p $APP_DIR/releases/$RELEASE
  sudo install -m 0755 /tmp/api.$RELEASE $APP_DIR/releases/$RELEASE/api
  sudo ln -sfn $APP_DIR/releases/$RELEASE $APP_DIR/current
  sudo systemctl restart yourapp-api
  sleep 2
  sudo systemctl is-active yourapp-api
  echo 'Deploy succeeded: $RELEASE'
"
```