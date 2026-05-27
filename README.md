# SaaS Multi-Tenancy Starter

Starter kit monorepo untuk membangun aplikasi SaaS multi-tenant seperti POS, CRM, HRM, inventory, billing, dan produk SaaS lain.

## Stack

- Backend: Go
- Database: PostgreSQL
- Cache/session/rate limit: Redis
- Frontend: Vue 3 + Vite + TypeScript + Tailwind CSS
- Deployment: Docker Compose

## Repository layout

```text
.
├── backend/
│   └── .env.example
├── frontend/
│   └── .env.example
├── docker-compose.yml
├── docker-compose.prod.yml
├── .env.example
├── IDEA.md
├── AGENT_TASKS.md
├── PROJECT.md
└── README.md
```

Backend and frontend implementation files are owned by later tasks in `AGENT_TASKS.md`. Task 01 only creates shell, env contracts, Docker contracts, and base docs.

## Development quick start

1. Copy env file:

```bash
cp .env.example .env
```

2. Update secrets before shared use:

```bash
JWT_ACCESS_SECRET=change-me-access-secret
JWT_REFRESH_SECRET=change-me-refresh-secret
```

3. Validate compose:

```bash
docker compose config
```

4. Start services after backend/frontend implementation exists:

```bash
docker compose up --build
```

Development ports:

- Backend: `http://localhost:8080`
- Frontend: `http://localhost:5173`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

## Production-oriented compose

`docker-compose.prod.yml` expects built images or image tags:

```bash
cp .env.example .env
# edit .env: strong secrets, production DATABASE_URL, CORS_ALLOWED_ORIGINS, image tags
docker compose -f docker-compose.prod.yml config
docker compose -f docker-compose.prod.yml up -d
```

Required production values:

- `POSTGRES_PASSWORD`
- `DATABASE_URL`
- `JWT_ACCESS_SECRET`
- `JWT_REFRESH_SECRET`
- `CORS_ALLOWED_ORIGINS`

## Environment contract

Backend requires:

```text
APP_ENV
APP_PORT
DATABASE_URL
REDIS_ADDR
JWT_ACCESS_SECRET
JWT_REFRESH_SECRET
JWT_ACCESS_TTL_MINUTES
JWT_REFRESH_TTL_HOURS
CORS_ALLOWED_ORIGINS
```

Frontend requires:

```text
VITE_API_BASE_URL
VITE_TENANT_HEADER
```

## Multi-tenancy contract

This project uses shared database multi-tenancy with `tenant_id` on tenant-scoped tables.

Rules:

- Every tenant-owned row must include `tenant_id`.
- Every tenant-scoped query must filter by `tenant_id`.
- Development tenant context uses `X-Tenant-ID`.
- Code should remain ready for future subdomain tenant resolution.
- Never log passwords, access tokens, refresh tokens, or secrets.

## Planned API base path

```text
/api/v1
```

Health endpoints from backend foundation task:

```text
GET /health
GET /ready
```

## Task execution

See `AGENT_TASKS.md` for non-overlap agent tasks and file ownership. Suggested first waves:

1. Task 01 — Monorepo, Docker, env, base documentation
2. Task 02 — Backend foundation
3. Task 03 — Database migrations/seed

Do not let agents edit files outside assigned ownership.

## Current status

Task 01 defines shell and contracts only. Backend, database migrations, auth, RBAC, tenant management, and frontend app are implemented in later tasks.
