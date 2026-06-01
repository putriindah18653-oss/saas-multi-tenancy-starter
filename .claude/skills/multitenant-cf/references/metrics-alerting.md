# Metrics & Alerting

Logs explain what happened. Metrics tell whether the system is healthy right now and whether users are being hurt. This stack should expose low-cardinality Prometheus metrics from the internal listener and use alerts for symptoms that require human action.

Metrics must never expose tenant names, emails, tokens, request bodies, query strings, object keys, or other high-cardinality/PII values.

---

## Architecture

```text
Go API / Worker
  ├── /healthz      → public-safe health for load balancer / uptime checks
  ├── /readyz       → internal readiness, checks dependencies
  └── /metrics      → internal only, Prometheus format

Prometheus or Grafana Agent / Alloy
  ↓ scrape internal listener via private network / Cloudflare Tunnel / localhost
Grafana
Alertmanager / notification channel
```

Rules:

- Serve `/metrics` only on the internal listener (`INTERNAL_PORT`, e.g. `9090`).
- Never expose `/metrics` on the public API origin.
- Do not label metrics with raw `tenant_id` by default. Tenant labels create high cardinality and can leak customer volume. Use plan/tier or route class instead.
- Keep logs and audit records separate from metrics.

---

## Health vs Readiness vs Metrics

| Endpoint | Audience | Purpose | Public? |
|---|---|---|---|
| `/healthz` | load balancer / uptime | process is alive; lightweight dependency check optional | yes, safe |
| `/readyz` | orchestrator/internal | DB/Redis/storage dependencies usable | internal preferred |
| `/metrics` | Prometheus/agent | numeric telemetry | never public |

Recommended:

- `/healthz`: process alive + maybe DB ping with short timeout.
- `/readyz`: DB, Redis, object storage, worker queue reachability.
- `/metrics`: Prometheus scrape.

---

## Go Implementation

Use Prometheus client:

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

Internal listener:

```go
func StartInternalServer(addr string) error {
    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/readyz", Readyz)
    mux.HandleFunc("/healthz", Healthz)
    return http.ListenAndServe(addr, mux)
}
```

Bind locally or privately:

```text
INTERNAL_PORT=9090
127.0.0.1:9090 behind cloudflared/private network
```

Never route public traffic to `/metrics`.

---

## Required Metrics

### HTTP API

```text
http_requests_total{method,route,status_class}
http_request_duration_seconds_bucket{method,route,status_class}
http_request_in_flight{route}
http_request_body_bytes_bucket{route}
http_response_body_bytes_bucket{route}
```

Label rules:

- `route` must be route pattern/class, not raw path.
  - Good: `/api/dashboard/media/{id}`
  - Bad: `/api/dashboard/media/2f3a...`
- `status_class`: `2xx`, `3xx`, `4xx`, `5xx`.
- Avoid `tenant_id`, `email`, `user_id`, request path, query string.

### PostgreSQL

From `pgxpool.Stat()`:

```text
pgx_pool_acquired_conns
pgx_pool_idle_conns
pgx_pool_total_conns
pgx_pool_max_conns
pgx_pool_acquire_count_total
pgx_pool_acquire_duration_seconds_total
pgx_pool_empty_acquire_count_total
pgx_pool_canceled_acquire_count_total
```

Also track application query outcomes:

```text
db_errors_total{operation,status}
db_tx_rollbacks_total{operation}
rls_fail_closed_total{operation}
```

### Redis

```text
redis_commands_total{operation,status}
redis_command_duration_seconds_bucket{operation,status}
redis_errors_total{operation}
rate_limit_redis_errors_total{route_class}
```

Keep `operation` low-cardinality:

```text
get, set, del, incr, eval, scan, xadd/asynq, ping
```

### Rate Limiting

From `rate-limiting.md`:

```text
rate_limit_allowed_total{rule}
rate_limit_blocked_total{rule}
rate_limit_redis_errors_total{route_class}
```

### Auth / SSO

```text
auth_login_attempts_total{result}
auth_refresh_total{result}
auth_internal_token_total{result}
auth_internal_bad_secret_total
session_revocations_total{reason}
```

Allowed `result` examples:

```text
success, invalid_credentials, rate_limited, locked, error
```

Do not label by email/user.

### Billing

```text
billing_charge_create_total{gateway,result}
billing_webhook_total{gateway,event,result}
billing_webhook_invalid_signature_total{gateway}
billing_webhook_duplicate_total{gateway}
billing_state_transition_total{from,to}
```

No labels for external payment ID or invoice ID.

### Media Upload / Worker

```text
media_upload_total{result,input_type}
media_upload_bytes_bucket{input_type}
media_conversion_jobs_total{result}
media_conversion_duration_seconds_bucket{result}
media_conversion_output_bytes_bucket{variant}
media_vips_errors_total{reason}
```

Allowed `variant` examples:

```text
thumb, small, medium, large
```

### Tenant Onboarding

```text
tenant_onboarding_total{trigger,result}
tenant_provisioning_jobs_total{step,result}
tenant_provisioning_duration_seconds_bucket{step,result}
tenants_stuck_provisioning
```

Allowed `trigger`:

```text
self_service, platform_admin
```

### Public Worker / Cache

If Worker metrics are not exported through Prometheus, capture them through logs/analytics and mirror key counters in the backend where possible.

```text
public_cache_requests_total{layer,result}
public_cache_purge_total{result}
public_ssr_render_total{result}
public_tenant_resolution_total{result}
```

Allowed `layer`:

```text
cache_api, kv, backend
```

### Email

```text
email_send_total{type,provider,result}
email_delivery_webhook_total{provider,event}
email_bounce_total{provider,type}
email_verification_total{result}
```

Allowed `type`:

```text
welcome, verify_email, password_reset, billing_notice
```

---

## HTTP Middleware Example

```go
var (
    httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total HTTP requests.",
    }, []string{"method", "route", "status_class"})

    httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "HTTP request duration.",
        Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
    }, []string{"method", "route", "status_class"})
)

func MetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
        next.ServeHTTP(rw, r)

        route := routePattern(r) // framework-specific pattern, not raw path
        statusClass := fmt.Sprintf("%dxx", rw.status/100)
        httpRequests.WithLabelValues(r.Method, route, statusClass).Inc()
        httpDuration.WithLabelValues(r.Method, route, statusClass).Observe(time.Since(start).Seconds())
    })
}
```

`routePattern(r)` must return a bounded set. If the framework cannot expose route patterns, map paths to route classes manually.

---

## Alerting Strategy

Alert on symptoms, not every individual error. Prefer burn-rate style alerts for user-facing SLOs.

### Core SLOs

| Area | SLO target |
|---|---:|
| API availability | 99.9% monthly for non-5xx responses |
| API latency | 95% of dashboard API requests < 500ms |
| Public SSR/cache availability | 99.9% successful public responses |
| Login success path | 99% non-rate-limited valid logins complete < 1s |
| Worker job freshness | 99% media/provisioning jobs processed within 5 min |

### Critical Alerts

```text
APIHigh5xxRate
APILatencyHigh
DatabaseUnavailable
RedisUnavailable
RLSAssertionFailure
WorkerQueueBacklogHigh
MediaConversionFailuresHigh
BillingWebhookInvalidSignatureSpike
BillingWebhookProcessingFailures
TenantProvisioningStuck
PublicCacheBackendFallbackHigh
DiskSpaceLow
BackupMissing
RestoreDrillFailed
```

### Example Prometheus Alert Rules

```yaml
groups:
  - name: yourapp-api
    rules:
      - alert: APIHigh5xxRate
        expr: |
          sum(rate(http_requests_total{status_class="5xx"}[5m]))
          /
          sum(rate(http_requests_total[5m])) > 0.02
        for: 10m
        labels:
          severity: page
        annotations:
          summary: "API 5xx rate above 2%"

      - alert: APILatencyHigh
        expr: |
          histogram_quantile(0.95,
            sum(rate(http_request_duration_seconds_bucket[5m])) by (le, route)
          ) > 0.5
        for: 15m
        labels:
          severity: ticket
        annotations:
          summary: "p95 API latency above 500ms"

      - alert: RedisUnavailable
        expr: up{job="redis"} == 0
        for: 2m
        labels:
          severity: page
        annotations:
          summary: "Redis is unavailable"

      - alert: WorkerQueueBacklogHigh
        expr: asynq_queue_pending > 1000
        for: 10m
        labels:
          severity: page
        annotations:
          summary: "Worker queue backlog high"

      - alert: TenantProvisioningStuck
        expr: tenants_stuck_provisioning > 0
        for: 30m
        labels:
          severity: ticket
        annotations:
          summary: "Tenants stuck in provisioning"

      - alert: BillingWebhookProcessingFailures
        expr: rate(billing_webhook_total{result="error"}[10m]) > 0
        for: 10m
        labels:
          severity: page
        annotations:
          summary: "Billing webhook processing errors"
```

Tune thresholds after baseline traffic exists.

---

## Dashboards

Minimum Grafana dashboard panels:

### API

- Request rate by route/status class.
- p50/p95/p99 latency by route.
- 5xx rate.
- 4xx rate including `429`.
- In-flight requests.

### Database

- pgx pool acquired/idle/total/max.
- acquire duration.
- DB errors by operation.
- transaction rollbacks.

### Redis

- command latency.
- Redis errors.
- rate-limit allowed/blocked.
- cache hit/miss if tracked.

### Workers / Jobs

- asynq pending/active/retry/dead counts.
- job success/failure rate.
- media conversion duration.
- provisioning job duration.

### Billing

- charges created by gateway/result.
- webhook events by gateway/result.
- invalid signature spikes.
- duplicate webhook count.

### Public Site

- SSR render count/error.
- cache hit ratio by layer.
- backend fallback rate.
- tenant resolution failures.

### Email

- sends by type/result.
- bounce/complaint count.
- verification completion count.

---

## Asynq Metrics

Use asynq's inspector or metrics integration. Track at least:

```text
asynq_queue_pending{queue}
asynq_queue_active{queue}
asynq_queue_retry{queue}
asynq_queue_dead{queue}
asynq_tasks_processed_total{queue,type,result}
asynq_task_duration_seconds_bucket{queue,type,result}
```

Queues should be low-cardinality:

```text
default, media, provisioning, billing, email
```

Alerts:

- `dead > 0` for critical queues.
- pending backlog grows for 10+ minutes.
- media/provisioning job failures spike.

---

## Backup and DR Metrics

Backups are production-critical. Expose or push:

```text
backup_last_success_timestamp{type}
backup_duration_seconds{type,result}
backup_bytes{type}
restore_drill_last_success_timestamp
```

Alert if:

```text
backup_last_success_timestamp older than 24h
restore drill missing for the quarter
backup size drops unexpectedly
```

Backup details remain in `backup-recovery.md`; metrics make failures visible.

---

## Security Metrics

Track counts, not secrets:

```text
auth_internal_bad_secret_total
rate_limit_blocked_total{rule}
billing_webhook_invalid_signature_total{gateway}
public_tenant_resolution_total{result="not_found"}
rbac_denied_total{permission_class}
```

Avoid labeling by raw permission if custom tenant permissions can create high cardinality. Use permission class/resource when possible.

Security spikes should create tickets or pages depending on severity.

---

## Local Development

For local testing, Prometheus is optional. A quick local Prometheus can be added:

```yaml
# docker-compose.metrics.yml
services:
  prometheus:
    image: prom/prometheus:v2.55.1
    ports:
      - "127.0.0.1:9091:9090"
    volumes:
      - ./docker/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro

  grafana:
    image: grafana/grafana:11.4.0
    ports:
      - "127.0.0.1:3000:3000"
```

Prometheus config:

```yaml
scrape_configs:
  - job_name: yourapp-api
    static_configs:
      - targets: ['host.docker.internal:9090']
```

On Linux, `host.docker.internal` may require:

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

---

## Testing

Required tests:

- `/metrics` is not reachable from public listener.
- `/metrics` is reachable from internal listener.
- HTTP middleware records route pattern, not raw path.
- Metrics labels do not include `tenant_id`, email, token, object key, or query string.
- 5xx responses increment `http_requests_total{status_class="5xx"}`.
- Rate-limit blocks increment `rate_limit_blocked_total`.
- Billing invalid signature increments `billing_webhook_invalid_signature_total`.
- Worker failure increments job failure metrics.

Example label safety test:

```go
func TestRouteMetricUsesPatternNotRawID(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/dashboard/media/550e8400-e29b-41d4-a716-446655440000", nil)
    route := routePattern(req)
    require.Equal(t, "/api/dashboard/media/{id}", route)
    require.NotContains(t, route, "550e8400")
}
```

---

## Definition of Done

```text
[ ] /metrics served only on internal listener
[ ] HTTP request count/duration metrics added
[ ] pgx pool metrics exported
[ ] Redis operation/error metrics exported
[ ] rate-limit metrics exported
[ ] auth/billing/media/provisioning/email metrics exported for critical paths
[ ] asynq queue depth and failure metrics visible
[ ] core Grafana dashboard exists
[ ] critical Prometheus alert rules exist
[ ] backup freshness alert exists
[ ] tests prevent high-cardinality/PII labels
```
