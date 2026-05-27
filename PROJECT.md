# Project Architecture — SaaS Multi-Tenancy Starter

## Purpose

This repository is a production-minded starter kit for future SaaS products. It is intentionally generic so it can become POS, CRM, HRM, inventory, billing, or another tenant-based application.

## Architecture summary

```text
Browser / Vue 3 frontend
        |
        | HTTP JSON API
        v
Go backend API
        |
        +--> PostgreSQL shared database with tenant_id isolation
        |
        +--> Redis for cache/session/rate limit support
```

Primary API base path:

```text
/api/v1
```

Development ports:

- Backend: `8080`
- Frontend: `5173`
- PostgreSQL: `5432`
- Redis: `6379`

## Repository layout

```text
backend-api/      Go API service (auth, RBAC, tenant/app routes, migrations)
frontend-owner/   Vue app for owner-app platform surface
frontend-tenant/  Vue app for tenant workspace surface
```

Both frontend apps share foundational Phase 1 patterns (auth store, API client, RBAC-aware navigation) while preserving separate root folders and independent app boundaries.

Database migrations live at:

```text
backend-api/migrations/
```

## Stack decisions

- Backend router: `chi`
- Database access: `pgxpool`
- Migrations: plain SQL compatible with `golang-migrate`
- Password hashing: `bcrypt`
- JWT library: `github.com/golang-jwt/jwt/v5`
- IDs: PostgreSQL UUIDs using `gen_random_uuid()` where possible
- Frontend: Vue 3 + Vite + TypeScript + Tailwind + Pinia + Vue Router + Axios
- Deployment: Docker Compose for development and production-oriented setups

## Multi-tenancy model

The project uses shared database multi-tenancy.

Tenant-owned tables must include:

```text
tenant_id UUID NOT NULL
```

Rules:

1. Every tenant-owned table must include `tenant_id`.
2. Every tenant-scoped query must filter by `tenant_id`.
3. API handlers must never trust a tenant id alone; they must check authenticated membership and permissions.
4. Development tenant context uses `X-Tenant-ID`.
5. The design must remain compatible with future subdomain resolution such as `tenant-a.app.com`.

Tenant isolation is a security boundary. Missing `tenant_id` filters are treated as data-leak bugs, not normal defects.

## RBAC model

RBAC has two levels.

### App-level roles

- `owner-app`
- `admin`
- reserved for later: `finance`, `support`, `marketing`

### Tenant-level roles

- `owner-tenant`
- `admin`
- `finance`
- `support`

### Permission examples

App permissions:

- `app.tenants.read`
- `app.tenants.create`
- `app.tenants.update`
- `app.tenants.delete`
- `app.users.read`
- `app.users.manage`
- `app.audit.read`

Tenant permissions:

- `tenant.dashboard.read`
- `tenant.users.read`
- `tenant.users.invite`
- `tenant.users.update`
- `tenant.users.remove`
- `tenant.settings.read`
- `tenant.settings.update`
- `tenant.billing.read`
- `tenant.billing.manage`
- `tenant.support.read`
- `tenant.support.manage`

## Security decisions

- Passwords are never stored in plaintext.
- JWT secrets come only from environment variables.
- Passwords, access tokens, refresh tokens, and secrets must not be logged.
- CORS origins are configurable.
- SQL access must use parameterized queries or safe query builders.
- Tenant-scoped data access must require tenant context and permission checks.
- Audit logs should record important security and administrative actions.
- Error responses should be safe and avoid leaking internals.

## Docker contracts

Development compose services:

- `backend`
- `frontend`
- `postgres`
- `redis`

Service names are part of the contract for later tasks:

- PostgreSQL hostname: `postgres`
- Redis hostname: `redis`
- Backend listens on `:8080`
- Frontend listens on `:5173`

## Environment contract

Backend:

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

Frontend:

```text
VITE_API_BASE_URL
VITE_TENANT_HEADER
```

## Agent workflow notes

Agents must follow `AGENT_TASKS.md` file ownership. If an agent needs an interface from another task, it should use the dependency contract instead of editing another task's files.

Recommended flow:

1. Finish and commit Task 01.
2. Run Task 02 and Task 03 after Task 01 contract is stable.
3. Continue wave-by-wave, verifying each task before merge.
