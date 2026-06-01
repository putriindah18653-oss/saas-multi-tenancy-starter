---
name: saas-multitenancy
description: >
  Guide to building a production-grade Multi-Tenancy SaaS with Golang + PostgreSQL RLS + Redis/asynq,
  Vue + Tailwind CSS + Cloudflare Workers, tenant custom domains, SSO, RBAC, billing, media upload,
  testing, Docker/VPS deployment, observability, monitoring, security, release, incident, and data
  retention practices. Use this skill whenever the user mentions multi-tenant SaaS, tenant isolation,
  custom tenant domains, PostgreSQL Row-Level Security, multi-tenant JWT, Cloudflare Workers KV,
  Vue dashboard/public SSR, tenant onboarding, RBAC, billing/webhooks, rate limiting, metrics/alerts,
  Prometheus/Grafana, email delivery, API contracts, release management, incident response,
  production readiness, local development, or wants to build an application that serves many
  clients/organizations each with their own domain.
---

# SaaS Multi-Tenancy Skill

## Stack Overview

```
Browser
  ↓
CF Worker PUBLIC  (1 worker, N custom domains)  → server-rendered (SSR) tenant sites
CF Worker DASHBOARD (1 worker, N manage.domain) → dashboard Vue SPA per tenant
  ↓ (fetch API, add Authorization; X-Tenant-ID is only a tenant-resolution hint cross-checked by backend)
VPS — Golang API (Gin/Echo + pgx + sqlc + go-redis) + asynq worker (image → AVIF)
  ↓
PostgreSQL (RLS per tenant) + Redis (cache + lock + jobs) + R2/S3 (media objects)
```

## Tenant & Domain Architecture

| Type | Example Domain | Worker |
|------|--------------|--------|
| Tenant with its own domain (public) | `kabarsiang.id` | PUBLIC |
| Tenant without a domain (uses owner domain) | `theme.portalonline.id` | PUBLIC |
| Dashboard for a tenant with a domain | `manage.kabarsiang.id` | DASHBOARD |
| Dashboard for a tenant on the owner domain | `manage.portalonline.id` | DASHBOARD |
| Owner SSO (fallback & tenants without a domain) | `sso.portalonline.id` | SSO/AUTH |

## Cloudflare KV Strategy

### CACHE_KV — Tenant Config & Public-Content Cache
```
tenant:{domain}                    → { PUBLIC_SITE_URL, PUBLIC_TENANT_API_KEY, tenant_id, features, branding }
tenant:{manage.domain}             → { tenant_id, DASHBOARD_API_URL, features }
cachever:{tenant_id}               → content version counter (bumped on purge; busts Cache API keys)
cachebody:{tenant_id}:{path}{qs}   → cached public-content body (global origin shield, TTL ~5 min)
```
Public content is cached in two layers (Cache API per-colo → KV global → backend), tenant-scoped,
**only** for unauthenticated `/api/public/*` GETs. See `references/frontend-vue-cloudflare.md`.

### CACHE_SSO — Edge Access-Session Store
```
session:{token_hash}         → { tenant_id, role, permissions }   (TTL 15m = access lifetime)
ratelimit:{ip}:{domain}      → login attempt counter
```
Note: only the short-lived **access** session lives here. Refresh-token rotation state
(`refresh_family:{id}` → current_jti) lives in **Redis** — it needs atomic compare-and-set
for reuse detection, which KV cannot do safely. See `references/auth-sso.md`.

## Auth Flow: Hybrid SSO

### Tenant with a custom domain (proxy — user never leaves the tenant domain)
```
1. User → manage.kabarsiang.id/login
2. CF Worker DASHBOARD:
   a. Read CACHE_KV: tenant:manage.kabarsiang.id → get tenant_id
   b. Server-to-server POST to sso.portalonline.id/api/token (invisible to the user)
   c. Store session in CACHE_SSO: session:{hash} → user data
   d. Set httpOnly cookie __session on manage.kabarsiang.id
3. All subsequent requests: Worker reads the cookie → validates in CACHE_SSO → injects headers to the backend
```

### Tenant without a custom domain (redirect to the owner SSO)
```
1. User → manage.portalonline.id/login
2. Redirect to sso.portalonline.id?redirect=manage.portalonline.id
3. Log in at sso.portalonline.id
4. Redirect back + set session cookie
```

## Role Hierarchy

```
PLATFORM LEVEL (platform owner):
  superuser  → full access to all tenants, platform configuration
  admin      → tenant management, support, cannot change billing

TENANT LEVEL (per tenant):
  owner-tenant → full access within their tenant, manage members
  admin        → operational access, cannot delete tenant/billing
  [other role] → custom per tenant, defined in the DB
```

## Decision Tree

### Choosing a Database Isolation Strategy?
→ **Row-Level Security (RLS)** — recommended for a single-instance VPS
→ Read: `references/database-schema.md`

### Building the Golang API?
→ Middleware chain, sqlc data access (+ pgx escape hatch), RLS store helper, Redis namespacing
→ Read: `references/backend-golang.md`

### Setting up Vue + Cloudflare Worker?
→ Deployment, KV binding, proxy to the backend, tenant resolution
→ Read: `references/frontend-vue-cloudflare.md`

### Implementing Auth & SSO?
→ JWT structure, CF Worker auth proxy, session in KV
→ Read: `references/auth-sso.md`

### Implementing Roles & Permissions?
→ RBAC structure, Go middleware, Vue route guard
→ Read: `references/rbac.md`

### Creating or onboarding a new tenant?
→ Self-service signup + admin create, transactional PG core + async provisioning, trial, soft email verification
→ Read: `references/tenant-onboarding.md`

### Email delivery, verification, or password reset emails?
→ Transactional provider, async jobs, templates, idempotency, webhooks, bounces, Mailpit local flow
→ Read: `references/email-delivery.md`

### Billing, subscriptions, or payments?
→ Platform plan catalog vs tenant subscriptions, Midtrans/Duitku/manual, signed idempotent webhooks, annual = 2 months free
→ Read: `references/billing.md`

### API contracts, versioning, or typed clients?
→ Versioned routes, success/error envelopes, error code registry, idempotency, OpenAPI, contract tests
→ Read: `references/api-contracts.md`

### Building the dashboard UI (authenticated, interactive)?
→ Tailwind design system, base components (form/table/modal/toast), RBAC-aware, SPA
→ Read: `references/ui-dashboard.md`

### Building the public site UI (anonymous, SEO-critical)?
→ SSR/pre-render (not SPA), SEO head + JSON-LD, branding at render, content components
→ Read: `references/ui-public.md`

### Frontend production hardening?
→ Vue build gates, bundle budgets, accessibility, CSP compatibility, hydration safety, Lighthouse, asset caching
→ Read: `references/frontend-production.md`

### Handling image / file upload?
→ Strict validation, convert to AVIF (multi-size), R2/S3 storage, async worker, tenant-scoped keys
→ Read: `references/media-upload.md`

### Logging, request tracing, or audit trail?
→ Structured slog, one request_id edge→backend, append-only tenant audit log shown in dashboard
→ Read: `references/observability.md`

### Rate limiting or abuse protection?
→ Redis-backed layered limits for login/signup/reset/upload/public API/webhooks, 429 envelope, Turnstile/WAF outer layer
→ Read: `references/rate-limiting.md`

### Metrics, dashboards, or alerts?
→ Prometheus metrics on internal listener, low-cardinality labels, SLOs, Grafana panels, alert rules
→ Read: `references/metrics-alerting.md`

### Deploying Prometheus, Grafana, Alertmanager, or exporters?
→ Monitoring Compose/profile, Prometheus scrape config, Grafana provisioning, Alertmanager routing, private access
→ Read: `references/monitoring-stack.md`

### Security hardening?
→ Secrets, auth/session, RLS, Redis, CSP/CORS/CSRF, Workers, storage, dependency/container hardening
→ Read: `references/security-hardening.md`

### Data retention, privacy, tenant export, or deletion?
→ Data classes, retention defaults, tenant lifecycle, export, deletion, legal hold, backup deletion caveat
→ Read: `references/data-retention-privacy.md`

### Release management?
→ Versioning, migrations, staging, deploy order, feature flags, smoke tests, rollback, release notes
→ Read: `references/release-management.md`

### Incident response?
→ Severity levels, roles, diagnostics, containment, security leak playbook, comms, postmortem
→ Read: `references/incident-response.md`

### Backup, restore, or disaster recovery?
→ PostgreSQL PITR + nightly dump, per-tenant export (GDPR/selective restore), R2 originals kept + versioned, encrypted off-site
→ Read: `references/backup-recovery.md`

### Setting up local development?
→ Docker Compose dependencies, env files, seed data, local domains, Mailpit, MinIO, dev commands
→ Read: `references/local-development.md`

### Setting up deployment & infrastructure?
→ VPS systemd or Docker Compose, Nginx, Wrangler deploy, CI/CD
→ Read: `references/deployment.md`

### Defining testing and code-quality gates?
→ Go/Vue/Worker tests, RLS isolation, Docker integration, Playwright E2E, CI quality gates
→ Read: `references/testing.md`
