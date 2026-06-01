# Security Hardening

Security hardening is the set of non-negotiable controls applied across backend, database, Redis, Workers, frontend, storage, deployment, and operations. Multi-tenancy increases blast radius: a single mistake can leak data across customers.

---

## Security Baseline

```text
[ ] Tenant isolation enforced by app filter + PostgreSQL RLS
[ ] app_user is NOSUPERUSER NOBYPASSRLS
[ ] platform_user is BYPASSRLS and used only by platform routes
[ ] migrations run as migrator/owner, never app_user
[ ] secrets are env/secret-manager only, never committed
[ ] all public writes rate-limited
[ ] all auth/session paths tested
[ ] all webhook paths signature-verified
[ ] all logs scrub secrets/PII
[ ] backups encrypted and offsite
[ ] /metrics and /api/internal/* are internal-only
```

---

## Threat Model

Primary threats:

| Threat | Control |
|---|---|
| Cross-tenant data access | RLS ENABLE+FORCE, USING+WITH CHECK, `store.InTenant`, isolation tests |
| Platform privilege abuse | separate platform pool, audit log, scoped impersonation |
| Credential stuffing | rate limits, generic errors, optional Turnstile/MFA |
| Session theft/replay | short access TTL, refresh rotation, edge session revocation |
| Stored XSS in tenant content | escaping/sanitization in public SSR, CSP |
| Public cache leak | tenant + version in cache keys, no auth response cached |
| Webhook forgery | signature verification, constant-time compare, idempotency |
| Secret leakage | log hygiene, secret manager, no query-string tokens |
| Backup breach | encryption, dedicated backup role, offsite access controls |
| Internal endpoint exposure | tunnel/private listener + internal secret + rate limiting |

---

## Secrets Management

Rules:

- Secrets only in env, Docker secrets, systemd `EnvironmentFile`, or external secret manager.
- `.env` is local/dev only and `chmod 600`.
- Never commit `.env`, private keys, JWT secrets, gateway keys, R2/S3 keys, SMTP keys.
- Rotate secrets with dual-key windows where supported.
- Never log secrets or full request headers.

Secrets requiring rotation plan:

```text
JWT_SECRET / signing keys
INTERNAL_SSO_SECRET_CURRENT/PREVIOUS
Payment gateway server keys
S3/R2 access keys
MAIL_API_KEY / webhook secret
Database passwords
Redis password
Backup encryption keys
```

Prefer asymmetric JWT signing (`RS256`/`EdDSA`) once multiple services verify tokens. If using HMAC, keep `JWT_SECRET` high entropy and rotate carefully.

---

## Authentication & Session Hardening

Controls:

- Access token short TTL (`15m` default).
- Refresh token rotation.
- Refresh idle TTL and absolute TTL.
- Store refresh/session state server-side or edge-side (`CACHE_SSO`) for revocation.
- Cookies: `HttpOnly`, `Secure`, `SameSite=Lax` or stricter where possible.
- Password hashing: Argon2id or bcrypt with current cost.
- Generic login/reset errors to prevent account enumeration.
- Rate-limit login/reset/signup (`rate-limiting.md`).
- Audit sensitive auth events.

Recommended security events:

```text
login_success
login_failed
password_changed
password_reset_requested
password_reset_completed
refresh_reused_or_invalid
session_revoked
internal_secret_failed
```

---

## Authorization Hardening

- Backend is authoritative; Vue checks are cosmetic.
- Platform routes use `RequirePlatformRole` only.
- Tenant routes use `RequirePermission` only.
- Superuser bypass must be explicit and audit-logged.
- Tenant-scoped resource misses return `404`, not `403`.
- Route-level authorization failures return `403`.
- Do not accept `tenant_id` from body/query/header except trusted Worker-internal flows that are verified.

Every new route must declare:

```text
[ ] auth required? public/internal/authenticated
[ ] tenant route or platform route?
[ ] required permission/role
[ ] audit event if sensitive
[ ] rate-limit rule if public/write
[ ] tests for forbidden and cross-tenant access
```

---

## Database Hardening

- RLS on every tenant table with `ENABLE` + `FORCE`.
- Policy has both `USING` and `WITH CHECK`.
- `tenant_id UUID NOT NULL`.
- `tenant_id` first in composite indexes.
- `app_user` is never owner/superuser/BYPASSRLS.
- `platform_user` is isolated to platform code path.
- `audit_log` append-only for app role (`REVOKE UPDATE, DELETE`).
- Backups use `backup_user`, never app runtime credentials.
- Avoid pgbouncer statement pooling; use transaction/session pooling.

CI must run RLS assertions from `database-schema.md`.

---

## Redis Hardening

Redis is operational state, not source of truth.

Controls:

- Require password.
- Bind privately; never expose to public internet.
- Use TLS/private network in production if crossing hosts.
- Separate DB index/prefixes by environment.
- Key names must not contain raw email/IP/token.
- Use TTLs for sessions, one-time codes, rate-limit counters, locks.
- Use atomic Lua for locks/rate limits where races matter.

Redis data classes:

```text
sessions / refresh state
rate-limit counters
one-time codes
idempotency claims
asynq queues
cache entries
```

---

## HTTP Security Headers

Set at Worker or Nginx depending on route.

Recommended baseline:

```text
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
X-Content-Type-Options: nosniff
X-Frame-Options: DENY or frame-ancestors via CSP
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
Content-Security-Policy: route-specific
```

Dashboard CSP starting point:

```text
default-src 'self';
script-src 'self';
style-src 'self' 'unsafe-inline';
img-src 'self' data: https:;
connect-src 'self' https://api.yourdomain.com;
frame-ancestors 'none';
base-uri 'self';
form-action 'self';
```

Public SSR CSP needs to account for tenant assets and inline boot scripts. Prefer nonce-based inline scripts:

```text
script-src 'self' 'nonce-{nonce}'
```

If using inline theme boot script, add a nonce rather than broad `unsafe-inline` in production.

---

## Input Validation & Output Encoding

- Validate at handler boundary with struct tags and explicit domain checks.
- Reject unknown/oversized payloads.
- Use `http.MaxBytesReader` for uploads and webhooks.
- Use parameterized SQL only.
- Escape all SSR HTML values.
- JSON-safe encode script payloads; escape `</script`.
- Sanitize tenant-authored rich text with an allowlist sanitizer.
- Never trust MIME or file extension; sniff bytes.

Stored-XSS boundary: public tenant pages render tenant-authored content. Tests must prove malicious titles/body cannot execute scripts.

---

## CORS / CSRF

CORS:

- Production allowlist only known dashboard/public origins.
- Do not use `Access-Control-Allow-Origin: *` with credentials.
- Do not reflect arbitrary Origin.

CSRF:

- If auth uses cookies, protect state-changing routes with SameSite cookies plus CSRF token or double-submit pattern.
- If auth uses bearer tokens in headers, XSS becomes the main risk; keep tokens out of localStorage where possible.
- Webhook routes are CSRF-exempt but signature-required.

---

## Cloudflare Worker Hardening

- Worker secrets stored via `wrangler secret`, never `vars`.
- Dashboard Worker never sends `INTERNAL_SSO_SECRET` to browser.
- Public Worker cache never stores responses with `Authorization` or cookies.
- Cache key includes tenant identity and content version.
- Unknown domain fails closed.
- Internal Worker→backend calls include request ID.
- Error responses to API clients are JSON, not login HTML for fetch requests.

---

## Storage Hardening

R2/S3:

- Originals bucket private, versioning enabled, retained permanently.
- Public bucket only for derived variants.
- Object keys tenant-scoped and random/UUID, never client filename.
- Strip EXIF/metadata before public variants.
- Use least-privilege access keys.
- Do not log signed URLs with secrets.
- Validate file before storing original.

---

## Dependency & Supply Chain

Required:

```bash
go list -m -u all
govulncheck ./...
npm audit --audit-level=high
npm outdated
```

Recommended CI:

- Dependabot/Renovate.
- Pin Docker image versions.
- Generate SBOM for release artifacts.
- Scan container images.
- Verify lockfiles are committed.
- Avoid unreviewed postinstall scripts where possible.

---

## Container / Host Hardening

Docker:

- Run as non-root user.
- No privileged containers.
- No Docker socket mounted into app containers.
- Bind app ports to localhost unless intentionally public.
- Postgres/Redis not publicly exposed.
- Read-only root filesystem where practical.
- Resource limits for worker/media conversion.

Systemd:

```text
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict where possible
ProtectHome=true
Restart=always
```

Nginx/edge:

- TLS only.
- Cloudflare IP allowlist is defense-in-depth, not auth.
- Do not log auth headers/cookies/query strings.

---

## Security Testing

Required tests/gates:

```text
[ ] RLS isolation tests
[ ] auth invalid/expired/replayed token tests
[ ] route permission tests
[ ] cross-tenant URL forcing tests
[ ] webhook invalid signature tests
[ ] stored XSS SSR tests
[ ] public cache leak tests
[ ] rate-limit tests
[ ] /metrics public exposure test
[ ] secret-in-log regression tests
[ ] dependency vulnerability scan
```

---

## Production Security Checklist

```text
[ ] All production secrets rotated from dev defaults
[ ] SPF/DKIM/DMARC configured for email
[ ] TLS/HSTS enabled
[ ] CORS allowlist configured
[ ] CSP configured for dashboard and public site
[ ] RLS CI assertions pass
[ ] app_user/platform_user/migrator_user verified
[ ] Internal listener not publicly reachable
[ ] Redis/Postgres not publicly reachable
[ ] Buckets have correct public/private policy
[ ] Backups encrypted and offsite
[ ] Restore drill completed
[ ] Rate limits active on public writes
[ ] Metrics/alerts active
[ ] Incident contacts/on-call defined
```

---

## Definition of Done

```text
[ ] Threat model reviewed for new feature
[ ] Route auth/permission/rate-limit/audit declared
[ ] Inputs validated and outputs encoded
[ ] Secrets not logged or committed
[ ] RLS and permission tests added
[ ] Security regression test added for any security fix
[ ] Dependency scans clean or exceptions documented
[ ] Production checklist updated if infrastructure changes
```
