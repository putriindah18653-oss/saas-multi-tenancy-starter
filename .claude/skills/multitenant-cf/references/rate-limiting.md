# Rate Limiting & Abuse Protection

Rate limiting is a security boundary for public SaaS surfaces: login, signup, password reset, email verification, media upload, public API reads, and payment webhooks. It must be enforced server-side; frontend throttling is only UX.

This stack uses Redis for distributed counters because API instances, workers, and Cloudflare Worker requests may run concurrently. Cloudflare WAF/Turnstile can add an outer layer, but the backend remains authoritative for user/tenant/account limits.

---

## Principles

- **Fail closed for abuse-sensitive writes**: if Redis is unavailable on login/signup/password reset, return a safe `503` or a conservative `429` rather than allowing unlimited attempts.
- **Fail open only for non-sensitive reads**: public content reads may continue with degraded protection if Redis is down, but log the degradation.
- **Key by multiple dimensions**: IP-only limits punish NATs; account-only limits allow IP rotation. Use layered keys.
- **Do not reveal account existence**: login/password reset limits must return generic responses.
- **Use tenant/domain in keys**: one tenant's traffic must not exhaust another tenant's quota.
- **Never trust client IP headers directly**: accept `CF-Connecting-IP` only from Cloudflare/Nginx trusted path; otherwise use remote address.
- **Every `429` uses the same error envelope**: `{ error: { code: "too_many_requests", message: "Too many requests" } }`.

---

## Limit Matrix

| Surface | Key dimensions | Example limit | Behavior |
|---|---|---:|---|
| Login | tenant/domain + IP + email hash | 5/min, 20/hour | `429`, generic message |
| Signup / tenant creation | IP + domain + email hash | 3/hour, 10/day | `429`; consider Turnstile after first failure |
| Password reset request | tenant/domain + IP + email hash | 3/hour | Always return generic success body |
| Email verification resend | tenant + user/email hash | 3/hour | `429`; show retry time |
| Refresh token rotation | session/user + IP | 30/min | suspicious activity log if exceeded |
| Media upload | tenant + user + IP | plan-based, e.g. 30/hour | `429`; also enforce size/quota |
| Billing create charge | tenant + user | 10/10min | idempotency key still required |
| Payment webhook | provider + IP + payment external id | high burst + signature required | invalid signatures still counted |
| Public API reads | tenant/domain + IP | 300/min | cache should absorb normal traffic |
| Internal endpoints | caller identity + IP | strict; e.g. 60/min | wrong secret counted separately |
| Admin/platform actions | platform user + IP | conservative | audit suspicious bursts |

Use these as starting defaults; tune using metrics in `metrics-alerting.md`.

---

## Redis Key Design

Use explicit prefixes and TTLs:

```text
rl:login:ip:{tenant_id}:{ip_hash}:{window}
rl:login:acct:{tenant_id}:{email_hash}:{window}
rl:signup:ip:{ip_hash}:{window}
rl:reset:acct:{tenant_id}:{email_hash}:{window}
rl:upload:user:{tenant_id}:{user_id}:{window}
rl:public:ip:{tenant_id}:{ip_hash}:{path_class}:{window}
rl:webhook:{provider}:{ip_hash}:{window}
```

Hash PII before putting it in Redis keys:

```go
func hashKey(s string) string {
    sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(s))))
    return hex.EncodeToString(sum[:16]) // enough for keying; do not log raw email/IP
}
```

Do not store raw email addresses in Redis keys, logs, or metrics labels.

---

## Algorithm Choice

### Fixed window for simple limits

Good for low-risk limits and simple counters.

```text
INCR key
EXPIRE key window on first hit
```

Pros: simple, fast. Cons: burst at boundary.

### Sliding window / token bucket for sensitive surfaces

Use token bucket or sliding window for login/signup/upload where boundary bursts matter.

Recommended default:

```text
login/signup/password reset → sliding window or token bucket
public reads/media upload    → token bucket
low-risk admin buttons       → fixed window acceptable
```

---

## Go Middleware Pattern

```go
package ratelimit

import (
    "context"
    "net/http"
    "time"
)

type Limiter interface {
    Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error)
}

type Decision struct {
    Allowed    bool
    Limit      int
    Remaining  int
    RetryAfter time.Duration
    ResetAt    time.Time
}

type Rule struct {
    Name   string
    Limit  int
    Window time.Duration
    KeyFn  func(*http.Request) string
}
```

Middleware shape:

```go
func Middleware(l Limiter, rule Rule) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            decision, err := l.Allow(r.Context(), rule.KeyFn(r), rule.Limit, rule.Window)
            if err != nil {
                // Sensitive routes should fail closed; public reads may fail open.
                httperr.Write(w, httperr.Unavailable("rate limiter unavailable"))
                return
            }

            w.Header().Set("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
            w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))
            w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))

            if !decision.Allowed {
                w.Header().Set("Retry-After", strconv.Itoa(int(decision.RetryAfter.Seconds())))
                httperr.Write(w, &httperr.AppError{
                    Status:  http.StatusTooManyRequests,
                    Code:    "too_many_requests",
                    Message: "Too many requests. Please try again later.",
                })
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

`httperr` must keep the standard envelope from `backend-golang.md`.

---

## Redis Fixed Window Implementation

Use Lua for atomic `INCR + EXPIRE`:

```lua
-- KEYS[1] = counter key
-- ARGV[1] = limit
-- ARGV[2] = window seconds
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
local ttl = redis.call('TTL', KEYS[1])
local allowed = current <= tonumber(ARGV[1])
return { allowed and 1 or 0, current, ttl }
```

Go wrapper returns remaining/retry metadata. Keep script SHA cached but fall back to `EVAL` on `NOSCRIPT`.

---

## Login Rate Limit

Apply layered checks before password verification:

```text
1. tenant/domain + IP
2. tenant/domain + email hash
3. tenant/domain + IP + email hash
```

Return the same error for invalid credentials and throttling where account enumeration is a concern. The UI can show a generic wait message after a `429` without confirming whether the email exists.

```go
func LoginRateRules(tenantID, ip, email string) []RuleCheck {
    emailHash := hashKey(email)
    ipHash := hashKey(ip)
    return []RuleCheck{
        {Key: "rl:login:ip:" + tenantID + ":" + ipHash, Limit: 20, Window: time.Hour},
        {Key: "rl:login:acct:" + tenantID + ":" + emailHash, Limit: 10, Window: time.Hour},
        {Key: "rl:login:combo:" + tenantID + ":" + ipHash + ":" + emailHash, Limit: 5, Window: time.Minute},
    }
}
```

On successful login, do **not** blindly delete all counters. Keeping some history slows credential stuffing. You may reset only the short combo counter for UX.

---

## Signup / Tenant Creation Abuse

Self-service signup is expensive: it creates DB rows, queues provisioning, writes KV, and sends email. Protect it with multiple layers:

```text
Cloudflare Turnstile after suspicious behavior
Redis limits by IP / email / domain slug
DB unique constraints for slug/email
asynq idempotency for provisioning
```

Recommended behavior:

- First few attempts: normal form.
- Suspicious burst: require Turnstile token.
- Exceeded: `429 too_many_requests`.
- Slug uniqueness failure remains `422 validation_failed`, not `500`.

---

## Password Reset and Verification Email

Password reset request must always respond generically:

```json
{ "ok": true, "message": "If the account exists, an email has been sent." }
```

Rate-limit by:

```text
tenant/domain + IP
tenant/domain + email hash
```

Verification resend limit:

```text
rl:verify-resend:{tenant_id}:{email_hash}
```

Do not send unlimited email from local or production systems. Email provider reputation is part of production reliability.

---

## Media Upload Limits

Rate limiting is not a substitute for media validation in `media-upload.md`. Use both:

```text
rate limit → request size cap → magic-byte sniff → dimensions/pixel cap → store original → enqueue conversion
```

Use plan-aware limits:

```text
free:      10 uploads/hour, 100 MB/day
starter:   50 uploads/hour, 1 GB/day
pro:      200 uploads/hour, 10 GB/day
enterprise: configured per tenant
```

Store durable quota facts in PostgreSQL if they affect billing/entitlements. Redis counters are operational, not authoritative for invoices.

---

## Public API Limits

Public tenant pages should be protected mainly by the two-layer cache (`Cache API → KV → backend`). Rate limiting is still needed for cache-miss amplification and search endpoints.

Key by:

```text
rl:public:ip:{tenant_id}:{ip_hash}:{path_class}
rl:public:tenant:{tenant_id}:{path_class}
```

Do not include raw path segments that may contain PII or unbounded cardinality. Normalize into path classes:

```text
article_view
article_list
search
asset_metadata
```

---

## Webhook Rate Limits

Payment webhooks are public but signature-authenticated. Count invalid signatures because they may be probes.

Layering:

```text
1. IP/provider rate limit
2. payload size cap
3. signature verification
4. Redis idempotency claim
5. DB unique constraint
```

Do not block legitimate provider retries too aggressively. Prefer generous burst limits plus signature/idempotency enforcement.

---

## Internal Endpoint Limits

`/api/internal/*` is protected by network isolation and `X-Internal-Secret`, but still rate-limit wrong-secret attempts:

```text
rl:internal:bad-secret:{ip_hash}
rl:internal:ok:{caller}:{ip_hash}
```

Wrong-secret attempts should produce security logs and metrics without logging the secret value.

---

## Cloudflare Layer

Use Cloudflare as an outer shield:

- WAF managed rules.
- Bot Fight / Super Bot Fight if available.
- Turnstile on signup/password reset escalation.
- Rate limiting rules for obvious floods.
- Cache rules for public static/SSR content.

But do not rely on Cloudflare alone. Backend Redis limits are still required because traffic can arrive through trusted paths and because tenant/account-aware limits require app context.

---

## Observability

Emit metrics described in `metrics-alerting.md`:

```text
rate_limit_allowed_total{rule}
rate_limit_blocked_total{rule}
rate_limit_redis_errors_total{route_class}
```

Log blocked events with low-cardinality fields:

```text
request_id, tenant_id, user_id(optional), rule, route_class, ip_hash
```

Never log raw IP, email, password, token, or secret.

---

## Testing

Required tests:

- Allows requests below limit.
- Blocks request above limit.
- Sets `Retry-After` and rate-limit headers.
- Uses standard error envelope for `429`.
- Login limits do not reveal whether email exists.
- Password reset response remains generic.
- Redis failures fail closed for sensitive routes.
- Public read route can fail open only if explicitly configured.
- Keys include tenant/domain where required.
- PII is hashed in keys/logs.
- Concurrent requests do not exceed limit due to race.

Concurrency test example:

```go
func TestLimiterConcurrent(t *testing.T) {
    const n = 50
    var allowed atomic.Int64
    g, ctx := errgroup.WithContext(context.Background())
    for i := 0; i < n; i++ {
        g.Go(func() error {
            d, err := limiter.Allow(ctx, "rl:test", 10, time.Minute)
            if err != nil { return err }
            if d.Allowed { allowed.Add(1) }
            return nil
        })
    }
    require.NoError(t, g.Wait())
    require.Equal(t, int64(10), allowed.Load())
}
```

---

## Definition of Done

```text
[ ] Every public write route has a documented rate-limit rule
[ ] Login/signup/password reset have layered IP + account limits
[ ] 429 uses the standard error envelope
[ ] Retry headers are set
[ ] Redis keys include tenant/domain where needed
[ ] PII is hashed before keying/logging
[ ] Sensitive routes fail closed if Redis is unavailable
[ ] Metrics emitted for allowed/blocked/errors
[ ] Tests cover concurrency and enumeration-safe responses
```
