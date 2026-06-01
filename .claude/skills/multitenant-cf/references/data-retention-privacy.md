# Data Retention, Privacy & Tenant Lifecycle

This document defines how tenant data is retained, exported, deleted, and protected. It complements `backup-recovery.md`, `observability.md`, `security-hardening.md`, and `incident-response.md`.

---

## Data Classification

Classify data before deciding retention.

| Class | Examples | Storage | Notes |
|---|---|---|---|
| Tenant business data | articles, pages, media metadata, settings | PostgreSQL + R2/S3 originals | tenant-owned |
| Account identity | name, email, password hash, roles | PostgreSQL | PII, security-sensitive |
| Billing data | invoices, subscription state, gateway refs | PostgreSQL | legal/tax retention may apply |
| Audit logs | actor, action, resource, request_id, IP hash | PostgreSQL | append-only, compliance/security |
| Operational logs | request logs, errors, metrics | stdout/log backend | minimize PII |
| Derived media | AVIF variants, thumbnails | public bucket/cache | recomputable |
| Ephemeral state | Redis sessions, rate limits, one-time codes | Redis/KV | TTL-based |
| Backups | PG dumps/PITR, originals | offsite encrypted | cross-tenant breach surface |

---

## Retention Defaults

Recommended defaults for production MVP:

```text
Tenant active data:          retained while tenant is active
Soft-deleted tenant data:    30 days before hard-delete, unless legal hold
Audit logs:                  1 year minimum, configurable
Operational logs:            30–90 days
Metrics:                     30–180 days depending storage cost
Redis sessions/codes:        TTL only
Derived media variants:      can be deleted/rebuilt anytime
Original media:              retained with tenant source data; versioned
Billing records:             retain per legal/tax requirements
Backups:                     PITR window 7–14 days, monthly archive as policy requires
```

These are defaults, not legal advice. Adjust by jurisdiction and customer contract.

---

## Tenant Offboarding States

Use explicit tenant lifecycle states:

```text
active
trial_expired
suspended
pending_deletion
deleted
```

Behavior:

| State | Login | Public site | Billing | Data mutation |
|---|---|---|---|---|
| active | allowed | served | active | allowed |
| trial_expired | limited | optional banner/disabled | upgrade required | limited |
| suspended | owner/admin limited | disabled or maintenance page | retry/resolve | blocked |
| pending_deletion | owner export only | disabled | cancel scheduled | blocked |
| deleted | blocked | 404/410 | inactive | blocked |

Do not physically delete immediately when a user clicks delete. Use a grace period unless policy requires immediate erasure.

---

## Tenant Export

Per-tenant export is required for portability, offboarding, and selective restore.

Export should include:

```text
tenant metadata
users and roles
content/pages/articles
media metadata
billing summary/invoices if allowed
audit log subset if policy allows
original media object manifest
```

Export rules:

- Run export through `store.InTenant` so RLS scopes data.
- Do not use `platform_user` unless the export is a platform-admin operation with audit record.
- Include schema/version metadata.
- Sign or checksum export archive.
- Encrypt export at rest.
- Expire download links.
- Audit export request and completion.

Example manifest:

```json
{
  "version": "2026-06-01",
  "tenant_id": "uuid",
  "exported_at": "2026-06-01T00:00:00Z",
  "tables": ["users", "articles", "media"],
  "objects": [{ "bucket": "originals", "key": "tenant/..." }]
}
```

---

## Tenant Deletion

Deletion workflow:

```text
1. Tenant owner requests deletion
2. Require re-authentication / confirmation phrase
3. Mark tenant pending_deletion
4. Disable public site and new mutations
5. Queue export if requested
6. Wait grace period
7. Hard-delete tenant-scoped PostgreSQL rows
8. Delete original media objects and derived variants
9. Delete KV tenant config/cache/session entries
10. Keep required billing/audit/legal records if policy requires
11. Mark tenant deleted with tombstone
```

Hard-delete must be idempotent and resumable.

Tombstone fields:

```text
tenant_id
deleted_at
slug/domain hash or reserved slug
reason
request_id
```

Keep tombstone minimal to avoid retaining unnecessary PII.

---

## User Deletion / Anonymization

For user-level deletion inside a tenant:

```text
[ ] revoke sessions
[ ] remove role memberships
[ ] anonymize author display where content remains
[ ] delete profile fields not required
[ ] retain audit actor as stable pseudonymous id if required
[ ] preserve billing/legal records if applicable
```

Do not break referential integrity for content. Prefer anonymization for historical content and audit trails when deletion would corrupt business records.

---

## Backups and Deletion Caveat

Backups are immutable-ish safety artifacts. Immediate deletion from all backups is often impractical.

Policy wording should state:

```text
Deleted data is removed from active systems within the deletion SLA.
Encrypted backups expire according to retention policy. If restored, deletion tombstones/jobs must be replayed before the system returns to service.
```

Restore process must include:

```text
[ ] apply latest migrations
[ ] replay deletion tombstones after restore
[ ] purge KV/cache for deleted tenants
[ ] delete restored media for deleted tenants
```

---

## Privacy by Design

Rules:

- Collect only required fields.
- Do not log raw email/IP/token/cookie/query string.
- Hash or truncate IP where possible.
- Do not put PII in Redis keys, metric labels, object keys, or cache keys.
- Avoid email open/click tracking unless explicitly needed and disclosed.
- Store provider external IDs but not full provider payloads unless necessary.
- Protect exports and backups as high-risk data.

---

## Admin Access and Impersonation

Platform admin access to tenant data is sensitive.

Rules:

- Prefer support tooling that shows metadata, not raw content.
- Require elevated role for tenant access.
- Require reason/comment for impersonation or data export.
- Audit every platform access:
  - actor
  - tenant
  - reason
  - action
  - request_id
- Show tenant-visible audit where appropriate.

If impersonation exists, make it explicit and time-limited. Never silently become a tenant user without an audit trail.

---

## Legal Hold

Legal hold prevents deletion even if tenant requests removal.

Minimum fields:

```text
hold_id
tenant_id
scope
reason
created_by
created_at
expires_at nullable
```

Deletion jobs must check legal hold before hard delete.

---

## Cache and Derived Data Purge

When tenant data is deleted or made private:

```text
[ ] bump cachever:{tenant_id}
[ ] delete cachebody:{tenant_id}:* KV keys
[ ] purge public Worker cached pages by version bump
[ ] delete derived public media variants
[ ] remove tenant domain mapping from KV
[ ] revoke SSO/session KV entries if applicable
```

Derived data is not authoritative, but stale derived data can still leak private content.

---

## Billing Data

Billing records may need longer retention than tenant content.

Rules:

- Store gateway references and invoice facts needed for accounting.
- Do not store card data; use provider tokens/hosted checkout.
- Keep billing records in platform-controlled tables with clear access policy.
- If tenant content is deleted, billing invoices may remain with minimized identity fields as required.

---

## Data Processing Records

Maintain a simple register:

```text
Data category
Purpose
Storage location
Retention period
Access roles
Exportable? yes/no
Deletable? yes/no/after retention
Subprocessor/provider
```

Subprocessors may include:

```text
Cloudflare
VPS/cloud provider
Postmark/Resend
Payment gateway
Object storage provider
Monitoring/logging provider
```

---

## Tests

Required tests:

- Tenant export only includes one tenant's data.
- Tenant deletion job is idempotent.
- Deleted tenant public domain returns disabled/404/410.
- KV/cache entries are purged or version-bumped.
- Media originals and variants are deleted according to policy.
- Legal hold blocks hard delete.
- User deletion revokes sessions.
- Logs/metrics do not include raw PII.
- Restore drill replays deletion tombstones.

---

## Definition of Done

```text
[ ] Data classes and retention periods documented
[ ] Tenant lifecycle states implemented
[ ] Tenant export is RLS-scoped, encrypted, audited, expiring
[ ] Tenant deletion is staged, idempotent, and resumable
[ ] Backup deletion caveat and replay procedure documented
[ ] Cache/KV/media purge included in deletion
[ ] Legal hold supported if needed
[ ] Admin access/impersonation audited
[ ] Tests cover export, deletion, cache purge, legal hold, restore replay
```
