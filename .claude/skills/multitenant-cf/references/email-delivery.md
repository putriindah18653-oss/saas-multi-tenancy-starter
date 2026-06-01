# Email Delivery

Email is part of the product surface: signup verification, password reset, onboarding, billing notices, security alerts, and operational notifications. It must be asynchronous, idempotent, observable, and safe from abuse.

---

## Provider Strategy

Recommended default:

```text
Production: Postmark or Resend for transactional email
Fallback: SMTP only if the provider supports SPF/DKIM/DMARC and webhooks
Local dev: Mailpit (see local-development.md)
```

Why transactional provider first:

- better deliverability
- webhook events for delivered/bounced/complained
- template support
- rate controls
- suppression lists

Do not send production transactional email from a raw VPS SMTP server unless you are prepared to operate deliverability, IP reputation, DKIM rotation, bounces, and blocklists.

---

## Email Types

| Type | Trigger | Blocking? | Recipient | Notes |
|---|---|---:|---|---|
| `verify_email` | signup/onboarding | no | tenant owner | soft verification; trial can start |
| `welcome` | tenant provisioning complete | no | tenant owner | contains dashboard/public links |
| `password_reset` | reset request | no | tenant/platform user | generic response, one-time code |
| `billing_notice` | invoice/payment/subscription state | no | tenant billing contacts | idempotent per invoice/event |
| `security_alert` | password changed, suspicious login | no | affected user/admin | no secrets in body |
| `invitation` | admin invites tenant user | no | invitee | one-time invite code |
| `domain_status` | custom domain verified/failed | no | tenant owner/admin | operational status |

All email sends are jobs. Request handlers enqueue jobs and return.

---

## Configuration

```dotenv
MAIL_PROVIDER=postmark # postmark | resend | smtp | noop
MAIL_API_KEY=change-me
MAIL_FROM=YourApp <no-reply@yourdomain.com>
MAIL_REPLY_TO=support@yourdomain.com
MAIL_BASE_URL=https://manage.portalonline.id
MAIL_PUBLIC_BASE_URL=https://portalonline.id
MAIL_WEBHOOK_SECRET=change-me

# SMTP fallback/local
MAIL_SMTP_HOST=localhost
MAIL_SMTP_PORT=1025
MAIL_SMTP_USERNAME=
MAIL_SMTP_PASSWORD=
MAIL_SMTP_TLS=false
```

Rules:

- Secrets come from env/secret manager, never code.
- `MAIL_FROM` domain must be authenticated with SPF/DKIM/DMARC.
- Local dev uses Mailpit and test domains only.
- `noop` provider is allowed only in tests, never production.

---

## DNS Deliverability Setup

Before production launch:

```text
[ ] SPF includes provider
[ ] DKIM records verified
[ ] DMARC record exists
[ ] Return-path/bounce domain configured if provider supports it
[ ] From domain matches product domain policy
[ ] Provider webhook endpoint configured
[ ] Suppression/bounce handling enabled
```

Example DMARC starting policy:

```text
_dmarc.yourdomain.com TXT "v=DMARC1; p=none; rua=mailto:dmarc@yourdomain.com; adkim=s; aspf=s"
```

After monitoring alignment, move toward stricter policy:

```text
p=quarantine → p=reject
```

---

## Data Model

Track email sends for idempotency, troubleshooting, and compliance.

```sql
CREATE TABLE email_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NULL REFERENCES tenants(id),
    type            VARCHAR(50) NOT NULL,
    recipient_hash  VARCHAR(128) NOT NULL,
    provider        VARCHAR(50) NOT NULL,
    provider_msg_id VARCHAR(255),
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    status          VARCHAR(30) NOT NULL DEFAULT 'queued', -- queued|sent|delivered|bounced|complained|failed|suppressed
    error_code      VARCHAR(100),
    error_message   TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_email_events_tenant ON email_events(tenant_id, created_at DESC);
CREATE INDEX idx_email_events_status ON email_events(status, created_at DESC);
```

RLS choice:

- Tenant-visible email history: include `tenant_id`, ENABLE+FORCE RLS, tenant policy.
- Platform/system-only operational table: no tenant UI access, platform pool only.

If tenant admins can view email delivery history, apply the standard tenant RLS checklist.

---

## Go Interface

```go
type Mailer interface {
    Send(ctx context.Context, msg Message) (ProviderResult, error)
}

type Message struct {
    Type           string
    TenantID       *uuid.UUID
    To             string
    Subject        string
    Template       string
    Data           map[string]any
    IdempotencyKey string
    ReplyTo        string
}

type ProviderResult struct {
    ProviderMessageID string
    Status            string // sent|queued
}
```

Provider implementations:

```text
internal/email/postmark.go
internal/email/resend.go
internal/email/smtp.go
internal/email/noop.go  # tests only
```

Do not let handlers call providers directly. They enqueue a job.

---

## Asynchronous Job Flow

```text
handler/service
  ↓ enqueue email job (asynq)
worker
  ↓ create/find email_events by idempotency_key
  ↓ render template
  ↓ send via provider
  ↓ update email_events status/provider_msg_id
  ↓ emit metrics/logs
provider webhook
  ↓ verify signature
  ↓ update delivered/bounced/complained
```

Idempotency key examples:

```text
verify_email:{tenant_id}:{user_id}:{code_id}
welcome:{tenant_id}
password_reset:{user_id}:{code_id}
billing_notice:{tenant_id}:{invoice_id}:{notice_type}
invitation:{tenant_id}:{invite_id}
```

The worker must skip send if `email_events.idempotency_key` already exists with `sent|delivered`.

---

## Templates

Structure:

```text
internal/email/templates/
  layouts/base.html
  layouts/base.txt
  verify_email.html
  verify_email.txt
  welcome.html
  welcome.txt
  password_reset.html
  password_reset.txt
  billing_notice.html
  billing_notice.txt
```

Rules:

- Always send multipart HTML + plain text.
- Escape all tenant/user-supplied values.
- Do not put passwords, JWTs, refresh tokens, or internal secrets in email.
- One-time links contain opaque codes, not raw JWTs.
- Links expire.
- Include support/contact footer.
- Billing emails must show server-computed amount/currency only.

---

## One-Time Codes

Use the Redis one-time code pattern from `backend-golang.md` for:

```text
verify_email
password_reset
invitation_accept
```

Rules:

- Store only hashed code values if persisted outside Redis.
- Short TTL:
  - email verification: 24h
  - password reset: 15–30m
  - invitation: 7d or configured
- Consume once.
- Rate-limit resend and reset requests (`rate-limiting.md`).
- Response must not reveal whether an email exists.

---

## Provider Webhooks

Webhook endpoint:

```text
POST /api/webhooks/email/{provider}
```

Security:

- Verify provider signature using `MAIL_WEBHOOK_SECRET` or provider-specific key.
- Apply request size cap.
- Rate-limit invalid signatures.
- Use idempotency by provider event/message id.
- Never trust recipient/provider fields without matching known `provider_msg_id`.

Events:

```text
delivered
bounced
complained
opened/clicked (optional; avoid privacy-invasive tracking unless needed)
```

Bounces/complaints should update suppression status and stop future non-critical emails to that address.

---

## Suppression and Preferences

Minimum:

```text
hard bounce      → suppress recipient
spam complaint   → suppress recipient
billing/security → still allowed if legally/contractually required, but use caution
marketing        → separate consent and unsubscribe, out of scope for transactional baseline
```

If marketing emails are added later, keep them separate from transactional mail.

---

## Observability

Metrics in `metrics-alerting.md`:

```text
email_send_total{type,provider,result}
email_delivery_webhook_total{provider,event}
email_bounce_total{provider,type}
email_verification_total{result}
```

Logs:

```text
request_id, tenant_id, email_type, idempotency_key, provider, provider_msg_id, status
```

Never log raw recipient email. Use `recipient_hash`.

---

## Testing

Required tests:

- Template renders HTML and text.
- Template escapes tenant/user input.
- Idempotency prevents duplicate sends.
- Provider error marks event failed and retries according to job policy.
- Webhook signature verification accepts valid and rejects invalid.
- Bounce webhook suppresses recipient.
- Password reset response is enumeration-safe.
- Resend verification is rate-limited.
- Local Mailpit flow works in `local-development.md`.

---

## Definition of Done

```text
[ ] Provider selected and DNS authenticated
[ ] MAIL_* env variables documented
[ ] Mailer interface implemented
[ ] Email jobs async via asynq
[ ] email_events table or equivalent tracking exists
[ ] Idempotency key used for every email type
[ ] Templates have HTML + text and escape user data
[ ] One-time links expire and consume once
[ ] Provider webhooks verify signatures
[ ] Bounce/complaint suppression implemented
[ ] Rate limits on reset/resend/signup emails
[ ] Metrics and logs emitted without PII
[ ] Tests cover template escaping, idempotency, webhook signatures, suppression
```
