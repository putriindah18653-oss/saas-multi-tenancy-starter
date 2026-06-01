# Backend API

Go REST API for the SaaS Multi-Tenancy Starter. Uses [chi](https://github.com/go-chi/chi) for routing, [pgx v5](https://github.com/jackc/pgx) for PostgreSQL, [go-redis](https://github.com/redis/go-redis) for caching and rate limiting, and [Prometheus client_golang](https://github.com/prometheus/client_golang) for metrics.

## Architecture

```
cmd/api/main.go          → Entry point, wires dependencies
internal/
  config/                → Configuration via environment variables
  database/              → PostgreSQL pool + RLS transaction wrapper
  redis/                 → Redis client
  auth/                  → JWT authentication, user profiles, sessions
  rbac/                  → Role-based access control + tenant membership
  tenant/                → Tenant CRUD, settings, app settings
  user/                  → Tenant-scoped user management
  audit/                 → Audit logging
  media/                 → AVIF image processing + upload serving
  metrics/               → Prometheus middleware
  middleware/             → Auth, rate limiting, tenant context, logging, etc.
  http/
    handler/             → HTTP handlers (one per domain)
    router/              → Route registration (one file per domain)
    response/            → JSON envelope helpers
  migrations/            → SQL migrations (numbered, applied alphabetically)
```

## Quick Start

```bash
cp backend-api/.env.example backend-api/.env
# Edit .env with your DATABASE_URL, JWT secrets, etc.

# Run migrations (idempotent — safe to re-run)
go run ./cmd/migrate up

# Start the API server
go run ./cmd/api
```

## Row-Level Security (RLS)

Multi-tenancy isolation uses PostgreSQL Row-Level Security. Every tenant-scoped database query **must** go through `database.WithRLS()` which opens a transaction, sets session-level GUC variables (`app.current_tenant_id`, `app.current_user_id`, `app.is_platform_admin`), runs the callback, and commits.

```go
// Example: tenant-scoped read
err := database.WithRLS(ctx, pool, database.RLSContext{
    TenantID: tenantID,
    UserID:   userID,
}, func(q database.Querier) error {
    return q.QueryRow(ctx, "SELECT ... FROM user_tenants WHERE ...").Scan(&result)
})
```

**Rules:**
- **Tenant-scoped queries** → `RLSContext{TenantID: ..., UserID: ...}`
- **Platform-admin queries** → `RLSContext{PlatformAdmin: true}`
- **User-scoped queries** → `RLSContext{UserID: ...}` (e.g., listing own memberships)
- Never call `pool.Query/Exec` directly on tables with RLS policies.

**Current RLS coverage** (migration 000006):
- `user_tenants` — select, insert, update, delete
- `tenant_settings` — all operations
- `audit_logs` — select, insert (update/delete: platform-admin only)

Tables not yet under RLS: `users`, `tenants`, `refresh_tokens`, `permissions`, `roles`, `role_permissions`. `FORCE ROW LEVEL SECURITY` is deferred until these tables are covered.

## Rate Limiting

Rate limits use Redis-backed counters with per-request EXPIRE refresh. Use `RateLimitRules` for new endpoints:

```go
tr.Use(middleware.RateLimitRules(redisClient, trustProxy,
    middleware.RateLimitRule{Name: "upload:avatar", Scope: middleware.RateLimitByIP,   Limit: 10, Window: time.Minute},
    middleware.RateLimitRule{Name: "upload:avatar", Scope: middleware.RateLimitByUser, Limit: 10, Window: time.Minute},
))
```

Available scopes: `RateLimitByIP`, `RateLimitByUser`, `RateLimitByTenant`. Rules with `Limit <= 0` or `Window <= 0` are skipped. When no active rules exist, requests pass through (fail-open). Redis errors return 503.

## Metrics

The `/metrics` endpoint exports Prometheus metrics:
- `saas_starter_http_requests_total{method, route, status_code}` — request counter
- `saas_starter_http_request_duration_seconds{method, route}` — latency histogram

**Protect `/metrics` at the reverse-proxy/firewall layer** — it is unauthenticated by design so Prometheus can scrape it.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | `development` / `production` — controls secure cookies & JWT strength |
| `APP_PORT` | `8080` | HTTP listen port |
| `TRUST_PROXY` | `false` | Trust `CF-Connecting-IP` (with `CF-Ray` verification) and `X-Real-IP` |
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection string |
| `DATABASE_MAX_CONNS` | `10` | pgx pool max connections |
| `DATABASE_MIN_CONNS` | `1` | pgx pool min connections |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `JWT_ACCESS_SECRET` | — | HS256 signing key (min 32 chars in production) |
| `JWT_REFRESH_SECRET` | — | HS256 signing key for refresh tokens |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Access token lifetime |
| `JWT_REFRESH_TTL_HOURS` | `168` | Refresh token lifetime (7 days) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated allowed origins |
| `INTERNAL_PROXY_SECRET` | — | Shared secret with Cloudflare Dashboard Worker |

## Error Codes

| Code | HTTP | Meaning |
|------|------|---------|
| `unauthorized` | 401 | Missing or invalid Bearer token |
| `invalid_credentials` | 401 | Wrong email/password |
| `invalid_refresh_token` | 401 | Refresh token expired or revoked |
| `refresh_token_required` | 400 | No refresh token in request or cookie |
| `password_change_required` | 403 | Must change temporary password first |
| `forbidden` | 403 | Insufficient app/tenant role or permission |
| `tenant_required` | 400 | Missing `X-Tenant-ID` header |
| `tenant_access_denied` | 403 | User is not a member of the requested tenant |
| `tenant_check_error` | 500 | Could not verify tenant membership (DB error) |
| `rate_limited` | 429 | Too many requests — see `Retry-After` header |
| `service_unavailable` | 503 | Rate limiter (Redis) unavailable |
| `invalid_upload` | 400 | File too large or invalid upload |
| `image_convert_failed` | 400 | AVIF conversion error |
| `upload_access_denied` | 403 | Attempting to read another user's upload |

## Migrations

Migrations live in `migrations/` and are applied alphabetically by the idempotent runner (`cmd/migrate/main.go`). There is no version tracking table — every migration must be safe to re-run:

- Use `CREATE OR REPLACE` for functions
- Use `DROP ... IF EXISTS` before `CREATE`
- Use `ON CONFLICT DO NOTHING` for inserts
- Use `IF NOT EXISTS` / `IF EXISTS` for DDL

**Important:** Migration 000006 enables RLS on three tables. New tables holding tenant data must receive RLS policies before queries go through `WithRLS`.

## Production Deployment

- **HTTPS is required** — refresh-token cookies are `Secure` in non-development environments.
- **Protect `/metrics`** — use firewall rules or a reverse-proxy auth layer.
- **Strip `X-Internal-Tenant-ID`** at the reverse proxy — it is trusted only from the Dashboard Worker.
- **Set strong JWT secrets** (≥32 characters) — the config loader rejects short secrets in production.
- **Add `Strict-Transport-Security`** at the reverse proxy — the API does not set it directly.
- **Monitor pool metrics** — pgx and Redis pool stats should be exported to your monitoring stack.
