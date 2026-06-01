# Incident Response

Incidents are handled with speed, clarity, and evidence. The goals are: protect tenant data, restore service, communicate honestly, preserve auditability, and prevent recurrence.

---

## Severity Levels

| Severity | Definition | Examples | Response |
|---|---|---|---|
| SEV1 | Active data breach, cross-tenant leak, total outage, payment corruption | RLS bypass, public cache leaks Tenant A to B, DB down | page immediately, incident commander, status updates |
| SEV2 | Major feature outage or severe degradation | login down, billing webhooks failing, worker backlog blocking uploads | urgent response, frequent updates |
| SEV3 | Partial degradation with workaround | email delayed, some public pages stale, non-critical dashboard module down | business-hours or on-call response |
| SEV4 | Minor issue/no user impact | alert noise, isolated retryable job failure | ticket and normal triage |

When unsure, escalate severity. You can downgrade later.

---

## Incident Roles

```text
Incident Commander (IC)  → coordinates, decides, keeps timeline
Tech Lead                → leads diagnosis/fix
Comms Lead               → user/internal updates
Scribe                   → records timeline, commands, decisions
Subject Matter Expert    → DB, Worker, billing, storage, etc.
```

Small teams can combine roles, but IC and Tech Lead should be distinct for SEV1/SEV2 when possible.

---

## First 10 Minutes Checklist

```text
[ ] Acknowledge alert/report
[ ] Assign severity
[ ] Open incident channel/document
[ ] Assign IC and Tech Lead
[ ] Start timeline
[ ] Identify affected tenants/users/surfaces
[ ] Check dashboards: API, DB, Redis, Workers, billing, queue, public cache
[ ] Decide immediate containment
[ ] Communicate initial internal status
[ ] For SEV1/SEV2, prepare external status update
```

Do not start destructive actions without recording what was done and why.

---

## Core Diagnostics

### API outage / high 5xx

Check:

```bash
curl -i https://api.yourdomain.com/healthz
curl -i http://127.0.0.1:9090/readyz
journalctl -u yourapp-api -n 300 --no-pager
# or Docker
docker compose logs --tail=300 api
```

Look at:

```text
http_requests_total 5xx
http_request_duration p95/p99
DB pool saturation
Redis errors
recent deploy/version
```

### Database issue

Check:

```sql
SELECT now();
SELECT count(*) FROM pg_stat_activity;
SELECT state, wait_event_type, wait_event, count(*) FROM pg_stat_activity GROUP BY 1,2,3;
SELECT * FROM pg_stat_database WHERE datname = 'yourdb';
```

Do not disable RLS to “quickly fix” a tenant issue.

### Redis issue

Check:

```bash
redis-cli -a "$REDIS_PASSWORD" PING
redis-cli -a "$REDIS_PASSWORD" INFO memory
redis-cli -a "$REDIS_PASSWORD" INFO stats
```

Impact:

```text
sessions/edge state may fail
rate limits may fail closed
asynq jobs may pause
one-time codes may fail
cache/idempotency degraded
```

### Worker queue backlog

Check:

```text
asynq_queue_pending
asynq_queue_retry
asynq_queue_dead
media_conversion_jobs_total{result="error"}
tenant_provisioning_jobs_total{result="error"}
```

Actions:

- scale/restart worker
- pause problematic job type if poison messages
- move dead jobs only after root cause understood

### Public cache leak/stale content

Check:

```text
tenant domain resolution
cache key includes tenant identity
cache version key cachever:{tenant_id}
Authorization/cookie not cached
recent Worker deploy
```

Containment:

```text
bump affected tenant cache version
purge KV cachebody keys
optional Cloudflare purge-by-URL
rollback Worker if cache key regression
```

### Billing incident

Check:

```text
billing_webhook_total{result="error"}
billing_webhook_invalid_signature_total
payments unique constraints
recent gateway config changes
idempotency Redis keys
```

Containment:

- stop automatic state transitions if corrupting data
- keep accepting signed webhooks if safe
- replay provider webhooks after fix
- never mark paid from client-side evidence alone

---

## Security Incident: Cross-Tenant Leak

This is SEV1.

Immediate actions:

```text
[ ] Stop the leak: rollback deploy, disable affected route, or bump/purge cache
[ ] Preserve evidence: logs, request IDs, cache keys, DB snapshots if needed
[ ] Identify affected tenants and time window
[ ] Determine data classes exposed
[ ] Check whether RLS failed or app/cache leaked
[ ] Notify leadership/legal according to policy
[ ] Prepare tenant communication
```

Common causes:

```text
missing FORCE RLS
missing WITH CHECK
handler used platform pool accidentally
tenant_id accepted from request input
public cache key missing tenant identity
authenticated response cached publicly
frontend displayed wrong tenant due to host resolution bug
```

Do not hide or overwrite audit evidence. Do not run broad manual SQL updates without review.

---

## Containment Playbooks

### Roll back application

Systemd:

```bash
sudo ln -sfn /opt/yourapp/releases/<previous> /opt/yourapp/current
sudo systemctl restart yourapp-api yourapp-worker
```

Docker:

```bash
export RELEASE_SHA=<previous-sha>
docker compose up -d api worker
```

Cloudflare Worker:

```bash
# Prefer provider-supported rollback for the installed Wrangler version, for example:
npx wrangler rollback --env production

# If rollback is unavailable, redeploy the previous known-good git SHA/artifact:
git checkout <previous-sha>
npm ci
npm run build
npx wrangler deploy --env production
```

Confirm actual Wrangler rollback command/version support before relying on it during an incident.

### Disable risky route temporarily

Options:

```text
Nginx temporary 503/404 for route
Cloudflare WAF rule
feature flag off
backend config denylist
```

Use only as containment, then fix root cause.

### Revoke sessions

```text
delete CACHE_SSO session entries for affected users/tenants
rotate refresh token family if replay suspected
force re-authentication
```

### Rotate secret

Use dual-key where supported:

```text
1. Add new as CURRENT, keep old as PREVIOUS
2. Deploy verifiers accepting both
3. Deploy signers/callers using new
4. Wait rotation window
5. Remove old PREVIOUS
```

For compromised secrets, shorten overlap or revoke immediately.

---

## Communication

Internal update template:

```text
SEV: <level>
Status: investigating / mitigated / monitoring / resolved
Impact: <who/what>
Start time: <UTC>
Current hypothesis: <short>
Actions taken: <short>
Next update: <time>
IC: <name>
```

External update template:

```text
We are investigating an issue affecting <surface>. Users may experience <symptom>. We will provide the next update by <time>.
```

For security incidents, coordinate with legal/compliance before detailed external statements, but do not delay containment.

---

## Evidence Collection

Collect:

```text
request_id / CF-Ray
release version / git SHA
relevant logs with timestamps
metrics screenshots/ranges
DB migration version
cache keys/version values
Worker deployment version
affected tenant IDs
commands run during incident
```

Do not export raw tenant data into ad hoc files unless necessary. If exported, encrypt and access-control it.

---

## Recovery Validation

Before declaring resolved:

```text
[ ] symptom stopped
[ ] root cause mitigated or safely worked around
[ ] smoke tests pass
[ ] RLS isolation tests pass if data isolation involved
[ ] metrics stable for agreed window
[ ] no new related alerts
[ ] affected tenants identified
[ ] support/comms updated
```

Monitoring window:

```text
SEV1: at least 60 minutes stable
SEV2: at least 30 minutes stable
SEV3: reasonable confidence / business decision
```

---

## Post-Incident Review

Complete within 2–5 business days for SEV1/SEV2.

Template:

```markdown
# Incident Review: <title>

## Summary

## Impact
- Affected tenants/users:
- Data affected:
- Duration:

## Timeline
- UTC timestamp — event/action/decision

## Root cause

## What went well

## What went poorly

## Where we got lucky

## Action items
| Action | Owner | Due date | Priority |
|---|---|---|---|

## Tests/alerts to add

## Communication follow-up
```

Action items must be concrete and tracked. At least one action should improve prevention, detection, or response.

---

## Incident-Specific Runbooks

### Login outage

```text
check auth metrics
check Redis/CACHE_SSO
check JWT signing key config
check recent auth deploy
check rate-limit false positives
```

### Email outage

```text
check email_send_total{result="error"}
check provider status
check MAIL_API_KEY / SMTP config
check worker queue
check Mail provider suppression/bounces
```

### Media conversion outage

```text
check worker logs
check vips availability
check R2/S3 credentials
check media_conversion_duration/errors
check disk/tmp space
```

### Tenant stuck provisioning

```text
check tenants_stuck_provisioning
check provisioning job retries/dead queue
check CF KV/domain API errors
check email send errors
re-enqueue idempotent provisioning job
```

### Backup failure

```text
check backup_last_success_timestamp
check storage credentials
check disk space
run manual backup
verify encrypted artifact offsite
schedule restore drill if confidence lost
```

---

## Do Not Do

```text
[ ] Do not disable RLS to fix an access issue
[ ] Do not use platform_user in tenant handlers
[ ] Do not mark invoices paid without signed gateway/manual-admin evidence
[ ] Do not delete logs/audit rows to hide noise
[ ] Do not purge all tenant caches blindly unless leak scope is unknown and urgent
[ ] Do not expose /metrics or internal endpoints publicly for debugging
[ ] Do not paste secrets/tokens/customer data into chat/tickets
```

---

## Preparedness Checklist

```text
[ ] On-call/contact list exists
[ ] Access to VPS/Cloudflare/DB/provider dashboards tested
[ ] Rollback procedure tested
[ ] Restore drill completed
[ ] Dashboards and alerts exist
[ ] Runbooks linked from alert descriptions
[ ] Security/legal contact path defined
[ ] Status page or customer comms path defined
[ ] Test tenant accounts available for smoke checks
```

---

## Definition of Done

Incident response is ready when:

```text
[ ] Severity definitions documented
[ ] Roles assigned for SEV1/SEV2
[ ] Core diagnostics documented
[ ] Security leak playbook exists
[ ] Rollback/disable/revoke/rotate playbooks exist
[ ] Communication templates exist
[ ] Post-incident review template exists
[ ] Alerts link to relevant runbooks
[ ] Team has practiced at least one tabletop exercise
```
