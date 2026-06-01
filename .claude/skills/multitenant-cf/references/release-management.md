# Release Management

Release management keeps changes safe, reversible, and observable. A release is not just a deploy; it includes tests, migrations, Worker deploys, backend deploys, cache behavior, monitoring, rollback, and communication.

---

## Release Principles

- Every release has a version, changelog, migration plan, and rollback plan.
- Tests and quality gates must pass before deployment.
- Migrations run separately as `migrator_user`, never inside API startup.
- Prefer backward-compatible database changes.
- Deploy in stages: staging → production canary/small blast radius → full production.
- Observe metrics/logs/audit after deploy.
- Rollback must be practiced and documented.

---

## Environments

```text
local       → developer machine, Docker Compose dependencies
staging     → production-like, isolated data/secrets/domains
production  → real users/data
```

Rules:

- Separate databases, Redis, KV namespaces, buckets, payment keys, mail provider config.
- Never reuse production secrets in staging/local.
- Staging should use sandbox payment and test email recipients.
- Staging should run the same migrations and startup path as production.

---

## Versioning

Use semantic-ish release identifiers:

```text
v2026.06.01-1
v1.8.0
git SHA for container image tag
```

Recommended image tags:

```text
yourapp-api:<git-sha>
yourapp-worker:<git-sha>
yourapp-api:stable
yourapp-worker:stable
```

Never deploy untagged `latest` without an immutable SHA recorded.

---

## Required Pre-Release Gates

From `testing.md`:

```text
[ ] gofmt/go vet/staticcheck/golangci-lint
[ ] go test ./...
[ ] integration tests with PostgreSQL + Redis
[ ] RLS assertions + behavioral isolation tests
[ ] Vue typecheck/lint/unit/build
[ ] Worker tests
[ ] Playwright smoke on staging
[ ] security regression tests
[ ] vulnerability scans
[ ] Docker image build succeeds
```

From `security-hardening.md`:

```text
[ ] route auth/permission/rate-limit/audit reviewed
[ ] no public /metrics exposure
[ ] no secret/logging regression
```

---

## Database Migration Strategy

### Safe migration pattern

Use expand/contract:

```text
1. Expand: add nullable column/table/index, keep old code working
2. Deploy app that writes both old/new or reads fallback
3. Backfill asynchronously if needed
4. Switch reads to new field
5. Contract: remove old field only after one or more releases
```

Avoid in one release:

- rename column used by old code
- drop column used by old code
- add NOT NULL without default/backfill plan
- long blocking migrations during peak traffic
- migration that assumes all tenants fit in one transaction

### Migration execution

```bash
migrate -path ./migrations -database "$MIGRATOR_DSN" up
```

Rules:

- Run as `migrator_user`.
- `app_user` must not own tables.
- Every new tenant table must include RLS and tests.
- Large backfills run in batches via maintenance job, not one giant transaction.
- Record migration version in release notes.

### Rollback migrations

Rollback SQL is useful but not always safe after data changes. For production, prefer forward fixes unless rollback is explicitly validated.

Every migration must document:

```text
[ ] backward-compatible with previous app version?
[ ] expected duration
[ ] lock risk
[ ] rollback/forward-fix path
[ ] backfill required?
[ ] RLS policy added/changed?
```

---

## Deployment Order

Recommended order:

```text
1. Announce maintenance window if needed
2. Freeze release branch/tag
3. Run pre-release gates
4. Deploy to staging
5. Run migrations on staging
6. Run staging smoke/E2E
7. Build immutable production artifacts/images
8. Run production DB backup checkpoint
9. Run production migrations
10. Deploy backend API
11. Deploy worker
12. Deploy Cloudflare Workers
13. Purge/bump public cache where needed
14. Run production smoke checks
15. Monitor metrics/alerts/logs
16. Mark release complete
```

If the frontend depends on new backend behavior, maintain compatibility or deploy backend first. If backend depends on new frontend behavior, deploy frontend first only if old backend remains compatible.

---

## Systemd Release Flow

```bash
# Build on CI
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/api ./cmd/api
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/worker ./cmd/worker

# On VPS
sudo install -m 0755 api /opt/yourapp/releases/<version>/api
sudo install -m 0755 worker /opt/yourapp/releases/<version>/worker
sudo ln -sfn /opt/yourapp/releases/<version> /opt/yourapp/current

migrate -path /opt/yourapp/current/migrations -database "$MIGRATOR_DSN" up

sudo systemctl restart yourapp-api
sudo systemctl restart yourapp-worker
sudo systemctl status yourapp-api yourapp-worker
```

Keep previous release directory for rollback.

---

## Docker Compose Release Flow

```bash
# CI builds and pushes immutable images
docker build --target api -t registry/yourapp-api:<sha> .
docker build --target worker -t registry/yourapp-worker:<sha> .
docker push registry/yourapp-api:<sha>
docker push registry/yourapp-worker:<sha>

# VPS pulls exact version
export RELEASE_SHA=<sha>
docker compose pull api worker

# Run migrations first
docker compose --profile migrate run --rm migrator

# Deploy app processes
docker compose up -d api worker

docker compose ps
docker compose logs -f api worker
```

Do not run migrations automatically as part of `api` container startup.

---

## Cloudflare Worker Release Flow

For dashboard/public Workers:

```bash
npm ci
npm run typecheck
npm run lint
npm run test:unit
npm run build
npx wrangler deploy --env staging
# smoke staging
npx wrangler deploy --env production
```

Rules:

- Secrets via `wrangler secret`, not `vars`.
- KV namespace IDs are environment-specific.
- Public Worker cache keys must remain backward-compatible or versioned.
- If cache format changes, bump cache version or purge affected tenant paths.

---

## Feature Flags

Use feature flags for risky changes:

```text
feature:{tenant_id}:{feature_name}
```

Rules:

- Backend enforces feature behavior.
- Frontend only hides/shows UI.
- Flags default off for new risky features.
- Support tenant-level rollout.
- Remove stale flags after rollout.

Good candidates:

```text
new billing gateway
new public SSR renderer
new upload pipeline
new dashboard module
plan feature changes
```

---

## Smoke Tests

Production smoke after deploy:

```text
[ ] /healthz returns ok
[ ] /readyz returns ok internally
[ ] login works for test tenant
[ ] dashboard loads tenant branding
[ ] public tenant page SSR returns title/canonical
[ ] API tenant isolation check passes with test tenants
[ ] worker processes a test no-op job
[ ] Redis reachable
[ ] DB pool healthy
[ ] metrics scrape works internally
[ ] no 5xx spike after 10–15 minutes
```

Automate where possible.

---

## Rollback Strategy

Rollback types:

```text
application rollback       → previous API/worker image/binary
Worker rollback            → previous wrangler deployment
configuration rollback     → restore previous env/secret value
migration rollback         → only if explicitly safe; otherwise forward fix
cache rollback             → bump/purge cache version
```

Rollback runbook must include:

```text
[ ] previous version identifier
[ ] command to switch back
[ ] whether DB migration is compatible
[ ] whether cache purge needed
[ ] smoke checks after rollback
```

Systemd rollback:

```bash
sudo ln -sfn /opt/yourapp/releases/<previous> /opt/yourapp/current
sudo systemctl restart yourapp-api yourapp-worker
```

Docker rollback:

```bash
export RELEASE_SHA=<previous-sha>
docker compose up -d api worker
```

Worker rollback:

```bash
# Prefer provider-supported rollback for the installed Wrangler version, for example:
npx wrangler rollback --env production

# If rollback is unavailable, redeploy the previous known-good git SHA/artifact:
git checkout <previous-sha>
npm ci
npm run build
npx wrangler deploy --env production
```

Confirm actual Wrangler rollback command/version support during release planning; keep the previous known-good artifact/SHA available.

---

## Release Notes Template

```markdown
# Release <version> - <date>

## Summary
- 

## User-visible changes
- 

## Backend changes
- 

## Frontend/Worker changes
- 

## Database migrations
- Migration versions:
- Backward compatible: yes/no
- Backfill: yes/no

## Security/RLS changes
- 

## Operational notes
- New env vars:
- New secrets:
- Cache purge required:
- Metrics/alerts added:

## Rollback plan
- App rollback:
- DB rollback/forward fix:
- Worker rollback:

## Validation
- CI run:
- Staging smoke:
- Production smoke:
```

---

## Change Freeze / High-Risk Releases

High-risk changes require extra review:

```text
RLS policy changes
Auth/session changes
Billing/payment changes
Migration dropping/rewriting data
Public cache key changes
File storage/object key changes
Secret rotation
Infrastructure exposure changes
```

Require:

```text
[ ] second reviewer
[ ] staging test with production-like data shape
[ ] rollback plan approved
[ ] monitoring dashboard open during deploy
[ ] incident owner assigned
```

---

## Definition of Done

```text
[ ] Version/tag recorded
[ ] Release notes written
[ ] Tests and quality gates pass
[ ] Migrations reviewed and run as migrator_user
[ ] Staging deploy and smoke passed
[ ] Production backup/checkpoint completed
[ ] Production deploy completed
[ ] Production smoke passed
[ ] Metrics/logs checked for regressions
[ ] Rollback path verified
[ ] Stakeholders notified if needed
```
