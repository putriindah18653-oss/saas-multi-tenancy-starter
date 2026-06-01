# SaaS Multitenancy Skill — Golang, PostgreSQL RLS, Redis, Vue, Cloudflare Workers

A Claude Code Skill for designing and implementing a production-grade multi-tenant SaaS blueprint with:

- Golang API and background workers
- PostgreSQL Row-Level Security (RLS)
- Redis / asynq
- Vue + Tailwind CSS dashboard
- Cloudflare Workers for dashboard/public/API routing
- Tenant custom domains
- Hybrid SSO, JWT, RBAC
- Billing webhooks
- Media upload and async processing
- Rate limiting
- Email delivery
- Metrics, alerts, and monitoring stack
- Backup, deployment, release, incident response
- Data retention and privacy
- Local development and testing strategy

This repository is a **Skill/documentation package**, not a runnable application starter kit. It provides architecture decisions, implementation patterns, safety rules, code examples, and production checklists for an agent or developer building the actual project.

---

## Skill Files

```text
multitenant-cf/
├── SKILL.md                         # Skill entrypoint used by skill loaders
├── multi-tenant-cloudflare.md       # Legacy/original entrypoint kept as alias/reference
├── README.md                        # This file
├── evals/
│   └── evals.json                   # Evaluation prompts and grading expectations
└── references/
    ├── api-contracts.md
    ├── auth-sso.md
    ├── backend-golang.md
    ├── backup-recovery.md
    ├── billing.md
    ├── database-schema.md
    ├── data-retention-privacy.md
    ├── deployment.md
    ├── email-delivery.md
    ├── frontend-production.md
    ├── frontend-vue-cloudflare.md
    ├── incident-response.md
    ├── local-development.md
    ├── media-upload.md
    ├── metrics-alerting.md
    ├── monitoring-stack.md
    ├── observability.md
    ├── rate-limiting.md
    ├── rbac.md
    ├── release-management.md
    ├── security-hardening.md
    ├── tenant-onboarding.md
    ├── testing.md
    ├── ui-dashboard.md
    └── ui-public.md
```

Packaged skill artifact:

```text
../multitenant-cf.skill
```

If you generated the package from this directory, it should be located at:

```text
/home/indatech/Documents/AI/SKILL/multitenant-cf.skill
```

---

## Installation

### Option 1 — Install from the packaged `.skill` file

Use this option when your skill runner supports installing `.skill` packages.

Package path:

```text
/home/indatech/Documents/AI/SKILL/multitenant-cf.skill
```

Typical flow:

1. Open your Claude/agent skill manager.
2. Choose **Install Skill** or **Import Skill**.
3. Select:

   ```text
   /home/indatech/Documents/AI/SKILL/multitenant-cf.skill
   ```

4. Restart the agent/session if required by your client.
5. Ask a prompt that should trigger the skill, such as:

   ```text
   I want to build a multi-tenant SaaS with Go, PostgreSQL RLS, Vue, and Cloudflare Workers. Create the architecture blueprint.
   ```

### Option 2 — Install from the unpacked skill directory

Use this option when your skill runner supports loading skills from a folder.

Source directory:

```text
/home/indatech/Documents/AI/SKILL/multitenant-cf
```

Requirements:

- The directory must contain `SKILL.md` at the top level.
- Reference docs should remain in `references/`.
- Eval prompts should remain in `evals/`.

Typical flow:

1. Copy or symlink the folder into your client’s skills directory.
2. Restart the agent/session if required.
3. Confirm the skill list includes:

   ```text
   saas-multitenancy
   ```

### Option 3 — Rebuild the `.skill` package

From this machine:

```bash
cd /home/indatech/Documents/AI/SKILL
python3 /home/indatech/Documents/AI/.gemini/skills/skills/skill-creator/scripts/quick_validate.py \
  /home/indatech/Documents/AI/SKILL/multitenant-cf

python3 /home/indatech/Documents/AI/.gemini/skills/skills/skill-creator/scripts/package_skill.py \
  /home/indatech/Documents/AI/SKILL/multitenant-cf
```

Expected result:

```text
Skill is valid!
Successfully packaged skill to: /home/indatech/Documents/AI/SKILL/multitenant-cf.skill
```

---

## When This Skill Should Trigger

Use this skill when the user asks about any of these topics:

- Multi-tenant SaaS architecture
- Tenant isolation
- PostgreSQL Row-Level Security
- Golang backend for multi-tenancy
- Redis/asynq background jobs
- Cloudflare Workers for tenant routing
- Cloudflare KV tenant config
- Tenant custom domains
- Vue dashboard for SaaS tenants
- Public tenant site with SSR/pre-rendering
- SSO, JWT, refresh tokens, session security
- RBAC and tenant-scoped permissions
- Billing subscriptions and payment webhooks
- Media upload and async image processing
- API contracts and typed frontend clients
- Rate limiting and abuse protection
- Email delivery and provider webhooks
- Metrics, Prometheus, Grafana, Alertmanager
- Backup and disaster recovery
- Deployment to VPS/Docker/systemd
- Release management and rollback
- Incident response
- Data retention/privacy
- Local development environment
- CI quality gates and testing strategy

---

## Core Safety Rules

When using this skill, preserve these invariants:

1. **`X-Tenant-ID` is only a hint.**
   - Authenticated routes must verify JWT signature and cross-check `tenant_id` claims.
   - Never trust a raw tenant header as authority.

2. **PostgreSQL RLS is mandatory for tenant data.**
   - Use `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY`.
   - Policies must include both `USING` and `WITH CHECK`.
   - Tenant access must run inside the same transaction that sets tenant context.

3. **Separate DB roles.**
   - `app_user`: tenant app queries, subject to RLS.
   - `platform_user`: cross-tenant platform queries only, `BYPASSRLS` allowed only here.
   - `migrator_user`: migrations only.

4. **Vue-first frontend.**
   - Dashboard is Vue SPA.
   - Public tenant site is SSR/pre-render oriented.
   - Do not introduce React/Next.js as the primary UI pattern.

5. **Canonical API error envelope.**

   ```json
   {
     "error": {
       "code": "machine_readable_code",
       "message": "Display-safe message"
     }
   }
   ```

6. **Billing webhooks require durable idempotency.**
   - Verify signature before state change.
   - Use PostgreSQL unique constraints/idempotency state as source of truth.
   - Redis may reduce duplicate work but must not be the only processed marker.

7. **Metrics are production baseline.**
   - `/metrics` must be internal-only.
   - Public health should not leak DB/Redis details.

---

## Reference Guide

Use the decision tree in `SKILL.md` first. Common doc clusters:

### Architecture and backend

```text
SKILL.md
references/database-schema.md
references/backend-golang.md
references/api-contracts.md
references/security-hardening.md
references/testing.md
```

### Frontend and Cloudflare Workers

```text
references/frontend-vue-cloudflare.md
references/ui-dashboard.md
references/ui-public.md
references/frontend-production.md
references/api-contracts.md
```

### Auth, SSO, and RBAC

```text
references/auth-sso.md
references/rbac.md
references/backend-golang.md
references/rate-limiting.md
references/security-hardening.md
```

### Billing and email

```text
references/billing.md
references/email-delivery.md
references/rate-limiting.md
references/metrics-alerting.md
references/testing.md
```

### Production operations

```text
references/deployment.md
references/release-management.md
references/metrics-alerting.md
references/monitoring-stack.md
references/backup-recovery.md
references/incident-response.md
references/security-hardening.md
```

### Local development and CI

```text
references/local-development.md
references/testing.md
references/deployment.md
```

### Privacy and lifecycle

```text
references/tenant-onboarding.md
references/billing.md
references/data-retention-privacy.md
references/database-schema.md
references/testing.md
```

---

## Example Prompts

### 1. Full architecture blueprint

```text
I want to build a production-grade multi-tenant SaaS using Golang, PostgreSQL RLS, Redis/asynq, Vue, Tailwind CSS, and Cloudflare Workers. Each tenant has a custom domain, dashboard, public site, SSO, RBAC, billing, and media upload. Create the architecture blueprint and implementation order. Treat this as a documentation-driven blueprint, not a runnable starter app.
```

### 2. Tenant isolation and database schema

```text
Design the PostgreSQL schema and RLS policies for tenants, users, roles, projects, billing subscriptions, and media assets. Include app_user/platform_user/migrator_user separation, FORCE RLS, USING/WITH CHECK policies, and CI tests that prove tenant isolation.
```

### 3. Golang backend structure

```text
Design the Golang backend layout for this multi-tenant SaaS. Include tenant resolution, JWT tenant claim verification, store.InTenant transaction handling, sqlc usage, Redis/asynq jobs, error envelope helpers, audit logging, and testing strategy.
```

### 4. Cloudflare Worker routing

```text
Create the Cloudflare Worker routing strategy for dashboard, public tenant site, SSO callbacks, and API proxying. Explain domain-to-tenant resolution using KV, when X-Tenant-ID may be sent as a hint, and why the backend must still verify JWT tenant claims.
```

### 5. Vue dashboard UI

```text
Create a Vue + Tailwind dashboard design system for this SaaS without a third-party UI library. Include layout, sidebar, cards, tables, forms, empty/loading/error states, tenant branding, and dark/light theme support.
```

### 6. Public tenant website

```text
Design the public tenant website architecture on Cloudflare Workers. It must be SEO-friendly, SSR/pre-render oriented, support tenant branding, dark/light theme without FOUC, and use Cache API + KV + backend safely. Do not make it SPA-only.
```

### 7. Billing webhook safety

```text
Implement the billing webhook design for Midtrans/Duitku subscriptions. Include signature verification, durable PostgreSQL idempotency, invoice amount cross-checking, tenant resolution from stored payment records, RLS tenant context, metrics, tests, and failure behavior for gateway retries.
```

### 8. Auth, SSO, and sessions

```text
Design hybrid SSO for tenant dashboards. Include login flow, SSO callback, internal Worker-to-backend endpoint protection, JWT claims, refresh token rotation, tenant claim cross-checking, rate limits, generic auth errors, and security tests.
```

### 9. RBAC design

```text
Design tenant-scoped RBAC for owner, admin, member, and custom roles. Include database tables, permission checks in Go handlers, frontend permission gating in Vue, audit logs, and tests for privilege escalation.
```

### 10. Rate limiting and abuse protection

```text
Design rate limiting for login, signup, password reset, email resend, media upload, billing webhooks, public API reads, and internal endpoints. Use Redis as the backend authority, Cloudflare WAF/Turnstile as an outer layer, and include metrics and tests.
```

### 11. Email delivery

```text
Design email delivery for verification, password reset, invites, billing receipts, and security notifications. Include provider setup, SPF/DKIM/DMARC, async job flow, idempotency, templates, suppression handling, provider webhooks, metrics, and tests.
```

### 12. Media upload

```text
Design tenant-scoped media upload. Include API contract, RBAC checks, tenant-scoped object keys, file size and MIME validation, async image processing via worker/libvips, metadata storage under RLS, public/private access rules, metrics, and tests.
```

### 13. API contracts

```text
Create API contracts for tenant dashboard endpoints. Include versioning, canonical success and error envelopes, pagination format, idempotency headers, OpenAPI generation, typed Vue client generation, and contract tests.
```

### 14. Local development

```text
Create a local development setup for this stack. Backend, PostgreSQL, Redis, MinIO, and Mailpit should run with Docker Compose. Include .env examples, local tenant domains, seed data, Worker local strategy, webhook simulation, reset commands, and developer troubleshooting.
```

### 15. CI and testing gates

```text
Create a CI quality gate for this project. Include Go tests, sqlc validation, PostgreSQL RLS isolation tests, Vue unit/component tests, Worker tests, Playwright E2E, API contract tests, security regression tests, linting, type checking, and deployment gating.
```

### 16. Production deployment

```text
Prepare production deployment for a single VPS plus Cloudflare Workers. Include Go API and worker deployment, Postgres, Redis, Docker Compose option, systemd option, migrations as migrator_user, Cloudflare Worker deploy, secrets, health/readiness, metrics, and smoke tests.
```

### 17. Monitoring stack

```text
Deploy Prometheus, Grafana, Alertmanager, postgres-exporter, redis-exporter, and node-exporter for this SaaS. Keep metrics internal-only. Include scrape config, alert rules, dashboard guidance, secret handling, and production verification.
```

### 18. Release and rollback

```text
Create a release management workflow for this SaaS. Include pre-release gates, expand/contract migrations, deploy order, feature flags, Cloudflare Worker rollout, smoke tests, rollback strategy, release notes, and high-risk release handling.
```

### 19. Incident response

```text
Create incident response runbooks for this multi-tenant SaaS. Include severity levels, first 10 minutes, cross-tenant data leak response, billing webhook failure, Redis outage, DB outage, stuck tenant provisioning, evidence collection, communication, recovery validation, and postmortem.
```

### 20. Data retention and privacy

```text
Design tenant lifecycle and privacy handling for trial, active, suspended, trial_expired, deletion_requested, and deleted tenants. Include data export, deletion/anonymization, backup caveat, cache/KV/Redis/media purge, legal hold, admin access audit, and tests.
```

### 21. Security audit prompt

```text
Audit this proposed endpoint for security: POST /api/dashboard/media. Check tenant isolation, JWT tenant claim verification, X-Tenant-ID usage, RBAC, RLS, rate limiting, file validation, storage key design, async processing, API contract, audit logs, metrics, and tests.
```

### 22. Documentation consistency audit

```text
Review the multitenant-cf skill documentation for contradictions. Check especially X-Tenant-ID trust boundary, API error envelope, Vue vs React guidance, public SSR vs dashboard SPA, metrics exposure, billing webhook idempotency, RLS role separation, and deployment/release claims.
```

---

## Evaluation Suite

Eval file:

```text
evals/evals.json
```

Current eval cases:

```text
architecture-implementation-plan
billing-webhook-safety
public-ssr-cache-frontend
production-deployment-ops
media-endpoint-security-review
tenant-lifecycle-privacy
local-dev-to-ci-quality-gates
```

The evals are designed to check whether a model using this skill:

- chooses the right reference-doc cluster,
- preserves tenant-isolation boundaries,
- avoids React/Next.js drift,
- uses canonical API contracts,
- includes production-readiness steps,
- avoids dangerous shortcuts.

Validate the skill package with:

```bash
python3 /home/indatech/Documents/AI/.gemini/skills/skills/skill-creator/scripts/quick_validate.py \
  /home/indatech/Documents/AI/SKILL/multitenant-cf
```

Package it with:

```bash
python3 /home/indatech/Documents/AI/.gemini/skills/skills/skill-creator/scripts/package_skill.py \
  /home/indatech/Documents/AI/SKILL/multitenant-cf
```

---

## Maintenance Checklist

Before publishing or re-packaging this skill:

- [ ] `SKILL.md` exists and has valid frontmatter.
- [ ] `multi-tenant-cloudflare.md` is kept in sync or intentionally marked as legacy alias.
- [ ] New references are linked from `SKILL.md` decision tree.
- [ ] Long references include headings/table-of-contents style navigation where useful.
- [ ] No React/Next.js examples are introduced as primary frontend guidance.
- [ ] `X-Tenant-ID` remains documented only as a hint.
- [ ] API errors use the canonical envelope.
- [ ] Metrics remain internal-only.
- [ ] Billing/email/webhook examples use durable idempotency.
- [ ] `evals/evals.json` is updated when major behavior changes.
- [ ] `quick_validate.py` passes.
- [ ] `package_skill.py` succeeds.

---

## License

No standalone license file is currently included in this skill directory. Add one before public distribution if needed.
