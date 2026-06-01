# Local Development — Full Stack Setup

This guide gets the SaaS stack running locally with the same boundaries used in production: PostgreSQL with RLS roles, Redis for cache/locks/jobs, the Go API + worker, Vue dashboard, Vue public SSR, local object storage, and local email capture.

Local development must preserve the security model. Do not bypass tenant context, RLS, or auth just because it is local.

---

## Local Topology

```text
Developer machine
├── Docker Compose
│   ├── postgres       → PostgreSQL test/dev DB with app/platform/migrator roles
│   ├── redis          → cache, rate-limit counters, asynq broker
│   ├── minio          → S3-compatible local object storage for media originals/variants
│   └── mailpit        → local SMTP + email web UI
├── Go API             → http://api.localhost:8080 or http://localhost:8080
├── Go worker          → asynq worker, vips image conversion, email jobs
├── vue-dashboard      → http://manage.localhost:5173
└── vue-public         → http://kabarsiang.localhost:5174 / http://theme.localhost:5174
```

Recommended local hostnames:

```text
api.localhost
manage.localhost
sso.localhost
kabarsiang.localhost
theme.localhost
```

Most browsers resolve `*.localhost` automatically. If yours does not, add entries to `/etc/hosts`:

```text
127.0.0.1 api.localhost manage.localhost sso.localhost kabarsiang.localhost theme.localhost
```

---

## Prerequisites

Required:

```bash
# Go, Node, Docker
go version          # 1.23+
node --version      # 20+
npm --version
docker --version
docker compose version

# Optional but recommended local CLIs
sqlc version
migrate -version
wrangler --version
```

For media conversion without running the worker in Docker, install `vips` locally:

```bash
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y libvips-tools
vips --version
```

If the worker runs in Docker, the worker image already includes `libvips-tools` (see `deployment.md`).

---

## Environment Files

Create these files from examples. Never commit real secrets.

```bash
cp .env.example .env
cp vue-dashboard/.env.example vue-dashboard/.env.local
cp vue-public/.env.example vue-public/.env.local
```

### Root `.env.example`

```dotenv
APP_ENV=development
PORT=8080
INTERNAL_PORT=9090

POSTGRES_PASSWORD=postgres
APP_DB_PASSWORD=app_password
PLATFORM_DB_PASSWORD=platform_password
MIGRATOR_DB_PASSWORD=migrator_password
DB_NAME=yourdb_dev

DB_URL=postgres://app_user:app_password@localhost:5432/yourdb_dev?sslmode=disable
PLATFORM_DB_URL=postgres://platform_user:platform_password@localhost:5432/yourdb_dev?sslmode=disable
MIGRATOR_DSN=postgres://migrator_user:migrator_password@localhost:5432/yourdb_dev?sslmode=disable

REDIS_PASSWORD=redis_password
REDIS_URL=redis://:redis_password@localhost:6379/0

JWT_SECRET=dev-jwt-secret-change-me
ACCESS_TTL=15m
REFRESH_IDLE_TTL=12h
REFRESH_ABSOLUTE_TTL=24h
INTERNAL_SSO_SECRET_CURRENT=dev-internal-secret-change-me
INTERNAL_SSO_SECRET_PREVIOUS=

# Local S3-compatible object storage (MinIO)
S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
MEDIA_BUCKET_PRIVATE=yourapp-originals
MEDIA_BUCKET_PUBLIC=yourapp-media
MEDIA_PUBLIC_BASE_URL=http://localhost:9000/yourapp-media

# Local email capture (Mailpit)
MAIL_PROVIDER=smtp
MAIL_SMTP_HOST=localhost
MAIL_SMTP_PORT=1025
MAIL_FROM=YourApp Dev <no-reply@localhost>
MAIL_BASE_URL=http://manage.localhost:5173

# Local Cloudflare substitutes
CF_ACCOUNT_ID=local
CF_API_TOKEN=local
CF_KV_NAMESPACE_ID=local

# Payment providers: disabled or sandbox in local dev
BILLING_CURRENCY=IDR
MIDTRANS_SERVER_KEY=dev-midtrans-server-key
MIDTRANS_CLIENT_KEY=dev-midtrans-client-key
DUITKU_MERCHANT_CODE=dev-duitku-merchant
DUITKU_API_KEY=dev-duitku-api-key
```

### `vue-dashboard/.env.example`

```dotenv
VITE_APP_ENV=development
VITE_BACKEND_URL=http://api.localhost:8080
VITE_PUBLIC_SITE_BASE=http://kabarsiang.localhost:5174
```

### `vue-public/.env.example`

```dotenv
VITE_APP_ENV=development
VITE_BACKEND_URL=http://api.localhost:8080
VITE_DEFAULT_TENANT_DOMAIN=kabarsiang.localhost
```

---

## Docker Compose for Local Dependencies

Use Compose for stateful dependencies even if the Go API is run directly on the host.

```yaml
# docker-compose.local.yml
services:
  postgres:
    image: postgres:16-bookworm
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
      POSTGRES_DB: postgres
      APP_DB_PASSWORD: ${APP_DB_PASSWORD:-app_password}
      PLATFORM_DB_PASSWORD: ${PLATFORM_DB_PASSWORD:-platform_password}
      MIGRATOR_DB_PASSWORD: ${MIGRATOR_DB_PASSWORD:-migrator_password}
      DB_NAME: ${DB_NAME:-yourdb_dev}
    ports:
      - "127.0.0.1:5432:5432"
    volumes:
      - pg_dev:/var/lib/postgresql/data
      - ./docker/postgres-local:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d $${DB_NAME:-yourdb_dev}"]
      interval: 5s
      timeout: 3s
      retries: 20

  redis:
    image: redis:7-bookworm
    restart: unless-stopped
    command: ["redis-server", "--appendonly", "yes", "--requirepass", "${REDIS_PASSWORD:-redis_password}"]
    environment:
      REDIS_PASSWORD: ${REDIS_PASSWORD:-redis_password}
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - redis_dev:/data
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a $$REDIS_PASSWORD ping | grep PONG"]
      interval: 5s
      timeout: 3s
      retries: 20

  minio:
    image: minio/minio:RELEASE.2025-01-20T14-49-07Z
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${S3_ACCESS_KEY:-minioadmin}
      MINIO_ROOT_PASSWORD: ${S3_SECRET_KEY:-minioadmin}
    ports:
      - "127.0.0.1:9000:9000" # S3 API
      - "127.0.0.1:9001:9001" # console
    volumes:
      - minio_dev:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 5s
      timeout: 3s
      retries: 20

  create-buckets:
    image: minio/mc:RELEASE.2025-01-17T23-25-50Z
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set local http://minio:9000 ${S3_ACCESS_KEY:-minioadmin} ${S3_SECRET_KEY:-minioadmin};
      mc mb -p local/${MEDIA_BUCKET_PRIVATE:-yourapp-originals};
      mc mb -p local/${MEDIA_BUCKET_PUBLIC:-yourapp-media};
      mc anonymous set download local/${MEDIA_BUCKET_PUBLIC:-yourapp-media};
      exit 0;
      "

  mailpit:
    image: axllent/mailpit:v1.21
    restart: unless-stopped
    ports:
      - "127.0.0.1:1025:1025" # SMTP
      - "127.0.0.1:8025:8025" # Web UI

volumes:
  pg_dev:
  redis_dev:
  minio_dev:
```

### Local Postgres bootstrap

```bash
mkdir -p docker/postgres-local
```

```bash
#!/usr/bin/env bash
# docker/postgres-local/001-init-local.sh
set -euo pipefail

: "${DB_NAME:=yourdb_dev}"
: "${APP_DB_PASSWORD:=app_password}"
: "${PLATFORM_DB_PASSWORD:=platform_password}"
: "${MIGRATOR_DB_PASSWORD:=migrator_password}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
  -v db_name="$DB_NAME" <<'SQL'
SELECT 'CREATE DATABASE ' || quote_ident(:'db_name')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'db_name')\gexec
SQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$DB_NAME" \
  -v db_name="$DB_NAME" \
  -v app_password="$APP_DB_PASSWORD" \
  -v platform_password="$PLATFORM_DB_PASSWORD" \
  -v migrator_password="$MIGRATOR_DB_PASSWORD" <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_user') THEN
    CREATE ROLE app_user LOGIN PASSWORD :'app_password' NOSUPERUSER NOBYPASSRLS;
  ELSE
    ALTER ROLE app_user WITH LOGIN PASSWORD :'app_password' NOSUPERUSER NOBYPASSRLS;
  END IF;

  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'platform_user') THEN
    CREATE ROLE platform_user LOGIN PASSWORD :'platform_password' NOSUPERUSER BYPASSRLS;
  ELSE
    ALTER ROLE platform_user WITH LOGIN PASSWORD :'platform_password' NOSUPERUSER BYPASSRLS;
  END IF;

  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'migrator_user') THEN
    CREATE ROLE migrator_user LOGIN PASSWORD :'migrator_password' NOSUPERUSER NOBYPASSRLS;
  ELSE
    ALTER ROLE migrator_user WITH LOGIN PASSWORD :'migrator_password' NOSUPERUSER NOBYPASSRLS;
  END IF;
END $$;

SELECT format('GRANT CONNECT ON DATABASE %I TO app_user, platform_user, migrator_user', :'db_name')\gexec
GRANT CREATE, USAGE ON SCHEMA public TO migrator_user;
GRANT USAGE ON SCHEMA public TO app_user, platform_user;
SQL
```

Make it executable:

```bash
chmod +x docker/postgres-local/001-init-local.sh
```

---

## Recommended Makefile

```makefile
include .env
export

.PHONY: dev-up dev-down dev-reset migrate seed api worker dashboard public test test-integration check

dev-up:
	docker compose -f docker-compose.local.yml up -d postgres redis minio create-buckets mailpit

dev-down:
	docker compose -f docker-compose.local.yml down

dev-reset:
	docker compose -f docker-compose.local.yml down -v
	docker compose -f docker-compose.local.yml up -d postgres redis minio create-buckets mailpit

migrate:
	migrate -path ./migrations -database "$(MIGRATOR_DSN)" up

seed:
	go run ./cmd/seed

api:
	go run ./cmd/api

worker:
	go run ./cmd/worker

dashboard:
	cd vue-dashboard && npm run dev -- --host manage.localhost --port 5173

public:
	cd vue-public && npm run dev -- --host 0.0.0.0 --port 5174

check:
	gofmt -w ./
	go vet ./...
	go test ./...
	cd vue-dashboard && npm run typecheck && npm run lint && npm run test:unit && npm run build
	cd vue-public && npm run typecheck && npm run lint && npm run test:unit && npm run build

test:
	go test ./...

test-integration:
	go test -tags=integration ./... -count=1
```

---

## First Run

```bash
# 1. Start dependencies
make dev-up

# 2. Generate sqlc code if internal/db is not committed
sqlc generate

# 3. Run migrations as migrator_user, never app_user
make migrate

# 4. Seed local data
make seed

# 5. Start backend processes in separate terminals
make api
make worker

# 6. Start frontend apps in separate terminals
make dashboard
make public
```

Open:

```text
Dashboard: http://manage.localhost:5173
Public:    http://kabarsiang.localhost:5174
Mailpit:   http://localhost:8025
MinIO:     http://localhost:9001
API:       http://api.localhost:8080/healthz or http://localhost:8080/healthz
```

---

## Seed Data

The seed command should be idempotent. Running it twice must not duplicate tenants, users, plans, or roles.

Recommended seed data:

```text
Platform admin
  email: admin@example.test
  password: password12345
  roles: superuser

Tenant A
  slug: kabarsiang
  domain: kabarsiang.localhost
  owner: owner@kabarsiang.test
  password: password12345
  plan: starter
  status: trial

Tenant B
  slug: theme
  domain: theme.localhost
  owner: owner@theme.test
  password: password12345
  plan: free
  status: trial

Plans
  free
  starter
  pro
  enterprise
```

Seed rules:

- Create platform-level rows with the platform/migrator connection.
- Create tenant-scoped rows via `store.InTenant`, not by disabling RLS.
- Hash passwords with the same code path used in production.
- Seed permissions and roles exactly as production expects.
- Write audit records only if useful for local debugging; avoid noisy seed audit spam if it confuses tests.

Example command shape:

```bash
go run ./cmd/seed --reset=false
```

---

## Local Cloudflare Worker Strategy

For local work, choose one mode per task.

### Mode A — Vite dev server talks directly to Go API

Use this for dashboard UI development. It is fastest.

```text
vue-dashboard dev server → http://localhost:8080
```

The Go API should allow local dev origins:

```text
http://manage.localhost:5173
http://kabarsiang.localhost:5174
```

Only in development. Production CORS remains strict.

### Mode B — Wrangler/Miniflare Worker dev

Use this when testing Worker behavior:

- tenant hostname resolution
- KV cache behavior
- dashboard auth proxy
- SSO edge session flow
- public SSR cache/purge

Example:

```bash
cd vue-public
npx wrangler dev --local --port 8787

cd ../vue-dashboard
npx wrangler dev --local --port 8788
```

Local KV is not production KV. Tests must still verify cache key format and purge behavior with Miniflare/Wrangler test utilities (see `testing.md`).

---

## Local Email Flow

Mailpit captures all outbound email.

```text
SMTP: http://localhost:1025
Web UI: http://localhost:8025
```

Local onboarding flow:

```text
1. Create tenant / signup
2. API enqueues provisioning job
3. worker sends welcome + verification email to Mailpit
4. Open Mailpit UI
5. Click verification link
6. tenant.email_verified becomes true
```

Rules:

- Email sending is always async through the worker.
- Email verification is soft/non-blocking for trial access.
- Verification links use one-time codes from Redis as documented in `backend-golang.md`.
- In local dev, links should point to `MAIL_BASE_URL`.

---

## Local Media Flow

Use MinIO as S3-compatible storage.

```text
Originals bucket: yourapp-originals  private
Public bucket:    yourapp-media      anonymous download in local dev
```

Flow:

```text
1. Upload image from dashboard
2. API validates bytes and stores original in private bucket
3. API inserts media row via RLS-bound transaction
4. API enqueues conversion job
5. worker shells out to vips and writes AVIF variants to public bucket
6. public UI uses variant URLs in srcset
```

Verify:

```bash
vips --version
curl http://localhost:9000/yourapp-media/<tenant>/<path-to-variant>.avif
```

Do not use client filenames as object keys. Object keys remain tenant-scoped and UUID/randomized even locally.

---

## Local Payment/Webhook Simulation

Local development should not hit real payment gateways unless explicitly using sandbox credentials.

Recommended modes:

```text
manual transfer mode → easiest for UI/state testing
sandbox gateway mode → tests signature code with provider sandbox
fixture webhook mode → replay signed test payloads from ./testdata/webhooks
```

Webhook testing rules:

- Always verify signatures, even in local dev.
- Test duplicate delivery.
- Test wrong amount/plan tampering.
- Use Redis idempotency claim + DB unique constraint as in `billing.md`.

Example fixture command:

```bash
go run ./cmd/dev-webhook --provider=midtrans --fixture=./testdata/webhooks/midtrans-settlement.json
```

---

## Running Tests Locally

```bash
# Fast backend unit tests
go test ./...

# Integration tests after dependencies + migrations
docker compose -f docker-compose.local.yml up -d postgres redis
migrate -path ./migrations -database "$MIGRATOR_DSN" up
go test -tags=integration ./... -count=1

# RLS assertions
psql "$MIGRATOR_DSN" -v ON_ERROR_STOP=1 -f ./scripts/assert_rls.sql
go test ./internal/isolation/... -run TestTenantIsolation -v -count=1

# Frontend
cd vue-dashboard && npm run typecheck && npm run lint && npm run test:unit && npm run build
cd ../vue-public && npm run typecheck && npm run lint && npm run test:unit && npm run build

# E2E
npx playwright test
```

---

## Resetting Local State

Full reset:

```bash
make dev-reset
make migrate
make seed
```

Reset only Redis:

```bash
redis-cli -a "$REDIS_PASSWORD" FLUSHDB
```

Reset only MinIO buckets:

```bash
docker compose -f docker-compose.local.yml exec minio sh -lc 'rm -rf /data/yourapp-originals /data/yourapp-media'
docker compose -f docker-compose.local.yml up create-buckets
```

Reset database with caution:

```bash
docker compose -f docker-compose.local.yml down -v postgres
```

Prefer full `make dev-reset` to avoid half-reset state.

---

## Troubleshooting

### `RLS returns zero rows`

Check:

- Is the request authenticated?
- Does JWT contain the expected `tenant_id`?
- Did the handler use `store.InTenant`?
- Is `set_config('app.current_tenant_id', ...)` inside the same transaction as the query?
- Are you accidentally using `app_user` for platform route or `platform_user` for tenant route?

### `relation does not exist` after startup

Migrations did not run or ran against the wrong database.

```bash
echo "$MIGRATOR_DSN"
migrate -path ./migrations -database "$MIGRATOR_DSN" version
```

### `permission denied for table`

Migration did not grant privileges to `app_user` / `platform_user`, or `ALTER DEFAULT PRIVILEGES` is missing.

### `Redis NOAUTH Authentication required`

Your `REDIS_URL` is missing the password:

```text
redis://:redis_password@localhost:6379/0
```

### `Email not received`

Check:

```text
MAIL_PROVIDER=smtp
MAIL_SMTP_HOST=localhost
MAIL_SMTP_PORT=1025
Mailpit UI: http://localhost:8025
```

Also confirm the worker is running; emails are sent async.

### `vips: command not found`

Install `libvips-tools` locally or run the worker in Docker.

### `CORS blocked in browser`

Allow local origins only in development:

```text
http://manage.localhost:5173
http://kabarsiang.localhost:5174
```

Do not loosen production CORS.

### Public site shows wrong tenant

Check hostname resolution and tenant domain seed rows:

```text
kabarsiang.localhost → tenant kabarsiang
theme.localhost      → tenant theme
```

Tenant resolution must use the request hostname, not a client-supplied tenant id.

---

## Local Development Definition of Done

Before implementing a feature:

```text
[ ] Dependencies start with `make dev-up`
[ ] Migrations run with migrator role
[ ] Seed creates platform admin + at least two tenants
[ ] API health check passes
[ ] Worker can process a test job
[ ] Dashboard opens on manage.localhost
[ ] Public tenant site opens on tenant.localhost
[ ] Mailpit receives verification email
[ ] MinIO stores an uploaded original and public AVIF variant
[ ] RLS isolation test passes locally
```
