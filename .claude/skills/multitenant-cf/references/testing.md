# Testing & Code Quality — Required Gates

Testing is not optional in this stack. Multi-tenancy fails silently when one layer is missed: a forgotten `WITH CHECK`, a UI permission mismatch, an unsigned webhook accepted, or a public cache key that leaks tenant content. Every release must pass the gates below before deployment.

## Quality Bar

Required on every pull request:

```text
1. Format / lint / static analysis
2. Unit tests
3. Integration tests against PostgreSQL + Redis
4. RLS tenant-isolation tests
5. API contract/error-shape tests
6. Frontend unit/component tests
7. Worker tests (Cloudflare/Miniflare)
8. E2E smoke tests on staging
9. Security-sensitive regression tests
```

A feature is not done until it has tests for the failure mode it introduces.

---

## Test Matrix

| Layer | Tooling | Required checks |
|---|---|---|
| Go code quality | `gofmt`, `go vet`, `staticcheck`, `golangci-lint` | formatting, vet, lint, no unchecked errors in critical paths |
| Go unit | `go test ./...` | pure business logic, validators, auth helpers, billing math |
| Go integration | `go test -tags=integration ./...` | handlers, store, Redis/asynq, DB transactions |
| PostgreSQL RLS | `psql` assertions + Go behavioral tests | `ENABLE` + `FORCE`, `USING` + `WITH CHECK`, fail-closed tenant access |
| API contract | Go handler tests / golden JSON | envelope shape, status codes, field errors, no raw internal detail |
| Vue dashboard | `vue-tsc`, ESLint, Vitest, Vue Test Utils | components, composables, RBAC display, theme light/dark |
| Vue public SSR | Vitest + server-render tests | SEO head, canonical, JSON-LD escaping, no browser API during SSR |
| Cloudflare Workers | Vitest + Miniflare / Wrangler test | routing, auth proxy, KV cache, purge versioning, internal headers |
| E2E | Playwright | login, tenant isolation, billing, upload, dashboard/public paths |
| Accessibility | axe / Playwright accessibility checks | keyboard, focus, ARIA, contrast, form errors |
| Backup/DR | scheduled restore drill | restore DB, run RLS assertions, row-count spot checks |

---

## Code Standards Gate

### Go

Required commands:

```bash
gofmt -w ./
go vet ./...
staticcheck ./...
golangci-lint run ./...
go test ./...
```

Standards:

- Keep handlers thin; business logic lives in services.
- Use `context.Context` on every request/DB/Redis call.
- Never use `context.Background()` inside request flow except process startup/shutdown.
- Always return envelope errors via `httperr.Write`; never raw `http.Error` for API responses.
- Never log secrets: `Authorization`, cookies, JWT contents, `X-Internal-Secret`, API keys, payment keys.
- No dynamic SQL string concatenation with user input; use sqlc or parameterized pgx.
- Tenant queries must run through `store.InTenant` / `store.InTenantRaw` so `set_config('app.current_tenant_id', ...)` and the query share one transaction.
- Platform routes must use the platform pool intentionally; never bypass RLS from normal tenant handlers.
- Money uses integers in minor units, never `float32`/`float64`.
- External side effects do not run inside DB transactions; enqueue idempotent jobs instead.

### SQL / Migrations

Every new tenant-scoped table must include:

```sql
tenant_id UUID NOT NULL
ALTER TABLE ... ENABLE ROW LEVEL SECURITY;
ALTER TABLE ... FORCE ROW LEVEL SECURITY;
CREATE POLICY ... USING (tenant_id = current_tenant_id())
                  WITH CHECK (tenant_id = current_tenant_id());
CREATE INDEX ... ON table_name (tenant_id, ...);
```

Migration rules:

- Run migrations as `migrator_user`, never `app_user`.
- `app_user` must be `NOSUPERUSER NOBYPASSRLS`.
- `platform_user` is the only `BYPASSRLS` role and is used only by platform routes.
- Add every new tenant table to the isolation tests.
- Revoke dangerous privileges for append-only tables like `audit_log`.

### Vue / TypeScript

Required commands:

```bash
npm run typecheck
npm run lint
npm run test:unit
npm run build
```

Standards:

- Use TypeScript strict mode.
- Keep logic in composables; templates stay declarative.
- Use base components from `ui-dashboard.md` instead of raw repeated controls.
- UI permission checks are cosmetic only; backend route checks are authoritative.
- Use semantic Tailwind tokens (`bg-surface`, `text-text`, `border-surface-border`) instead of hard-coded colors when light/dark theme matters.
- No React/Next.js patterns in Vue code.
- Public SSR code must not read `window`, `document`, `localStorage`, or `matchMedia` during server render.
- Escape tenant-authored content in SSR HTML and script payloads.

### Cloudflare Workers

Standards:

- Public Worker cache key must include tenant identity and content version.
- Never cache authenticated/personalized responses under public shared keys.
- Dashboard Worker must forward auth intentionally and never expose internal secrets to the browser.
- Internal endpoints require `X-Internal-Secret` and should be reachable only via tunnel/private path.
- KV writes/purges must be idempotent.

---

## Backend Testing

### Unit Tests

Target pure logic first:

```bash
go test ./internal/... -run 'Test' -count=1
```

Required areas:

- `httperr` envelope formatting.
- Tenant extraction and validation.
- JWT claims parsing and expiry.
- Permission wildcard matching: `*`, `resource:*`, `resource:action`.
- Billing price calculation: monthly/annual, currency, integer minor units.
- Payment signature verification: Midtrans SHA512, Duitku MD5.
- Media validation: MIME sniffing, size limits, extension mismatch.
- Cache-key construction: tenant + path + version.

Example billing test:

```go
func TestAnnualPriceIsTenMonths(t *testing.T) {
    got := AnnualPrice(100_000)
    require.Equal(t, int64(1_000_000), got)
}
```

### Handler / API Contract Tests

Every handler must prove:

- Correct status code.
- Correct JSON envelope.
- Validation errors map to `error.fields`.
- 401/403/404 semantics are correct.
- Cross-tenant resource misses return 404, not 403.
- 500 responses do not leak raw internal details.

Golden shape:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Invalid input",
    "fields": {
      "email": "Invalid email"
    }
  }
}
```

### Integration Tests: PostgreSQL + Redis

Use Docker Compose for test dependencies:

```bash
docker compose up -d postgres redis
docker compose --profile migrate run --rm migrator
go test -tags=integration ./... -count=1
```

Required integration coverage:

- `store.InTenant` sets tenant and commits/rolls back correctly.
- `store.InTenantRaw` works for dynamic queries without losing RLS scope.
- Redis idempotency claims for webhook/job handling.
- asynq enqueue/dequeue path for provisioning and media jobs.
- Audit log insert on sensitive actions.
- Health endpoint returns unhealthy when DB/Redis is down.

---

## RLS Tenant Isolation Testing

Canonical details live in `database-schema.md` → `Tenant Isolation Testing`. This gate is mandatory.

Run both:

```bash
# Config assertions: fail if any tenant table lacks FORCE RLS / policy coverage
psql "$MIGRATOR_DSN" -v ON_ERROR_STOP=1 -f ./scripts/assert_rls.sql

# Behavioral isolation tests, using app_user connection
go test ./internal/isolation/... -run TestTenantIsolation -v -count=1
```

Required scenarios:

- Tenant A sees only Tenant A rows.
- Tenant B sees only Tenant B rows.
- No active tenant sees zero rows.
- Cross-tenant insert is rejected by `WITH CHECK`.
- Updating `tenant_id` to another tenant fails or affects zero rows.
- Deleting another tenant's row affects zero rows.

New tenant table checklist:

```text
[ ] tenant_id UUID NOT NULL
[ ] tenant_id-first index
[ ] ENABLE ROW LEVEL SECURITY
[ ] FORCE ROW LEVEL SECURITY
[ ] USING policy
[ ] WITH CHECK policy
[ ] grants to app_user/platform_user reviewed
[ ] isolation test fixture added
[ ] assert_rls.sql catches misconfig
```

---

## Frontend Testing — Vue Dashboard

Recommended scripts:

```json
{
  "scripts": {
    "typecheck": "vue-tsc --noEmit",
    "lint": "eslint .",
    "test:unit": "vitest run",
    "test:e2e": "playwright test",
    "build": "vite build"
  }
}
```

Required tests:

- `useTenant`: loads tenant, applies branding CSS variables.
- `useTheme`: toggles `light`, `dark`, `auto`; applies `html.dark` through VueUse.
- `hasPermission`: matches backend wildcard semantics.
- `BaseInput`: renders validation errors and ARIA attributes.
- `BaseModal`: focus trap, Escape close, ARIA labels.
- `BaseTable`: empty/loading/error states.
- Billing UI: annual saving shown; client price is display-only.
- Upload UI: rejects invalid type/size before upload but still expects backend validation.

Permission UI tests must assert only visibility. They do not prove security. Backend permission tests are required too.

---

## Frontend Testing — Public SSR

Public UI is SSR and SEO-critical. Required tests:

- `renderShell()` includes title, description, canonical URL, Open Graph tags.
- Canonical uses tenant's own domain.
- JSON-LD is valid and escaped.
- Tenant-authored title/body cannot break out of HTML or `<script>`.
- Branding CSS variables are in first HTML response.
- Theme boot script is present before app hydration script.
- SSR render path does not access browser-only APIs.
- Public cache never stores authenticated/personalized response.

Example checks:

```ts
it('escapes tenant-authored title in SSR head', () => {
  const html = renderShell({ tenant, page: maliciousPage, appHtml: '<main />' })
  expect(html).not.toContain('<script>alert(1)</script>')
  expect(html).toContain('&lt;script&gt;alert(1)&lt;/script&gt;')
})
```

---

## Cloudflare Worker Testing

Use Miniflare/Wrangler-compatible tests for Worker behavior.

Required public Worker tests:

- Resolves tenant by hostname.
- Unknown domain returns 404/tenant-not-found.
- Public cache key includes tenant ID/domain + path + cache version.
- KV shield fallback works when Cache API miss occurs.
- Purge bumps `cachever:{tenant_id}` so old cache keys become unreachable.
- Does not cache responses with `Authorization` or visitor-specific data.

Required dashboard Worker tests:

- Proxies dashboard API with auth headers intact.
- Does not expose `INTERNAL_SSO_SECRET` to client responses.
- Handles 401 refresh/session flow as designed in `auth-sso.md`.
- Adds/propagates `X-Request-ID` for observability.

Required SSO/internal tests:

- `X-Internal-Secret` accepted for current and previous secret.
- Wrong secret rejected.
- Rotation window works.
- Secret is never logged.

---

## E2E Tests — Playwright

Run against staging, not production:

```bash
npx playwright test --project=chromium
```

Required smoke flows:

1. Platform admin logs in.
2. Platform admin creates tenant.
3. Tenant owner logs in via dashboard domain.
4. Tenant dashboard loads correct branding.
5. Tenant A cannot access Tenant B resource URL.
6. RBAC hides forbidden action and backend rejects forced request.
7. Public tenant domain renders SSR HTML with SEO tags.
8. Billing plan page shows monthly/annual prices and annual saving.
9. Payment webhook duplicate does not double-apply payment.
10. Media upload creates AVIF variants and public URL.
11. Audit log records sensitive actions.
12. Dark/light theme toggle works on dashboard and public hydrated island.

Accessibility minimum:

- Keyboard can navigate modals/forms.
- Focus visible on interactive controls.
- Form errors are announced/associated.
- No critical axe violations on dashboard core pages.

---

## Security Regression Tests

Always add regression tests for these classes:

- Cross-tenant read/write/delete.
- Missing `WITH CHECK` policy.
- Platform route accidentally using tenant auth, or tenant route using platform pool.
- JWT with wrong issuer/audience/expiry.
- Refresh token replay.
- Webhook invalid signature.
- Webhook duplicate delivery.
- Client-supplied amount/plan tampering.
- Stored XSS in tenant-authored public content.
- Public cache leak across tenants.
- Internal endpoint reachable without internal secret.
- Secret accidentally present in logs.

---

## CI Pipeline

Recommended GitHub Actions layout:

```yaml
name: Test

on:
  pull_request:
  push:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-bookworm
        env:
          POSTGRES_PASSWORD: postgres
        ports: ['5432:5432']
        options: >-
          --health-cmd "pg_isready -U postgres"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 10
      redis:
        image: redis:7-bookworm
        ports: ['6379:6379']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Format check
        run: test -z "$(gofmt -l .)"
      - name: Vet
        run: go vet ./...
      - name: Unit tests
        run: go test ./... -count=1
      - name: Migrate test DB
        run: migrate -path ./migrations -database "$MIGRATOR_DSN" up
        env:
          MIGRATOR_DSN: postgres://postgres:postgres@localhost:5432/yourdb_test?sslmode=disable
      - name: RLS assertions
        run: psql "$MIGRATOR_DSN" -v ON_ERROR_STOP=1 -f ./scripts/assert_rls.sql
        env:
          MIGRATOR_DSN: postgres://postgres:postgres@localhost:5432/yourdb_test?sslmode=disable
      - name: Integration tests
        run: go test -tags=integration ./... -count=1
        env:
          APP_USER_DSN: postgres://app_user:test@localhost:5432/yourdb_test?sslmode=disable
          REDIS_URL: redis://localhost:6379/0

  vue-dashboard:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: vue-dashboard
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: npm
          cache-dependency-path: vue-dashboard/package-lock.json
      - run: npm ci
      - run: npm run typecheck
      - run: npm run lint
      - run: npm run test:unit
      - run: npm run build

  vue-public:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: vue-public
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: npm
          cache-dependency-path: vue-public/package-lock.json
      - run: npm ci
      - run: npm run typecheck
      - run: npm run lint
      - run: npm run test:unit
      - run: npm run build

  e2e-staging:
    runs-on: ubuntu-latest
    needs: [backend, vue-dashboard, vue-public]
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: npm ci
      - run: npx playwright install --with-deps chromium
      - run: npx playwright test
        env:
          BASE_URL: https://staging.yourdomain.com
```

Adjust DB bootstrap to match the migration tool used by the project. The invariant stays the same: migrations run as migrator/owner, application tests run as `app_user`.

---

## Definition of Done

A task is complete only when:

```text
[ ] Code follows Go/Vue/SQL standards above
[ ] Unit tests added/updated
[ ] Integration tests added/updated for DB/Redis behavior
[ ] RLS isolation tests updated for new tenant tables
[ ] API error contract tested
[ ] Frontend component/composable tests added when UI changes
[ ] Worker tests added when routing/cache/auth changes
[ ] E2E smoke updated for critical user flow changes
[ ] Security regression test added for any bug/security fix
[ ] CI passes cleanly
[ ] No secrets or raw internal errors in logs/responses
```
