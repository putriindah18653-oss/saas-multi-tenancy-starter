# Observability — Structured Logging, Request Tracing & Audit Trail

Two distinct streams, often confused:

- **Operational logs** — diagnostic, high-volume, ephemeral. JSON to stdout → journald → an aggregator. For debugging "what happened in this request."
- **Audit log** — a persisted, tenant-scoped, **append-only** record of *who did what*. Lives in Postgres, shown in the dashboard, retained/backed up. For "who deleted this / who changed that role."

They are bridged by one **`request_id`**: every audit row stores the `request_id` of the action, so an admin reading the audit trail can pivot to the operational logs of that exact request. That single id is minted once (at the edge) and flows through everything — context, every log line, the response header, the audit row, and the correlation id shown in a 500.

This reuses the skill's primitives: `store.InTenant` (RLS, `backend-golang.md`), `httperr` (`backend-golang.md`), RBAC (`rbac.md`), `BaseTable` (`ui-dashboard.md`), the internal-listener pattern (`deployment.md`).

---

## 1. Structured Logging Foundation (`slog`, JSON)

One logger, JSON output, set up once. No new dependency — `slog` is stdlib (already used ad-hoc by `Recover` and the media worker; this makes it the standard).

```go
// pkg/logging/logging.go
package logging

import (
    "context"
    "log/slog"
    "os"
)

// New — JSON logger to stdout. Level from env (info in prod, debug in dev).
func New(level slog.Level) *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

// ─── Context-aware logging ──────────────────────────────────────────────────
// Every log line in a request should carry request_id + tenant_id + user_id without
// the caller passing them each time. We stash a request-scoped logger in context.

type ctxKey struct{}

func Into(ctx context.Context, l *slog.Logger) context.Context {
    return context.WithValue(ctx, ctxKey{}, l)
}

// From — the request-scoped logger (already tagged), or the default if none.
func From(ctx context.Context) *slog.Logger {
    if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
        return l
    }
    return slog.Default()
}
```

> Call `logging.From(ctx).Info("…")` everywhere. Because the middleware below tagged that logger with `request_id`/`tenant_id`/`user_id`, those fields appear on every line automatically — which is what makes per-tenant and per-request log filtering possible during an incident.

---

## 2. Request-ID Middleware — the one id, end to end

```go
// internal/obs/requestid.go
const HeaderRequestID = "X-Request-ID"

// RequestID — establish the single correlation id for this request.
// Source order: the edge Worker's X-Request-ID (it forwards CF-Ray or a UUID,
// see auth-sso.md / frontend-vue-cloudflare.md) → else mint one. The Worker is
// trusted (only it reaches the origin over the tunnel/allowlist), so we accept its id.
func RequestID(base *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            id := r.Header.Get(HeaderRequestID)
            if id == "" {
                id = uuid.NewString()
            }
            // Echo it back so clients/curl can quote it when reporting an issue.
            w.Header().Set(HeaderRequestID, id)

            // Build the request-scoped logger. tenant_id/user_id are added later by a
            // small enricher once auth + tenant middleware have populated the context.
            l := base.With("request_id", id)
            ctx := logging.Into(WithRequestID(r.Context(), id), l)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Enrich — run AFTER AuthMiddleware + ResolveTenant: fold tenant_id/user_id into the
// request logger so all subsequent lines carry them.
func Enrich(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        l := logging.From(r.Context())
        if t, ok := tenant.FromContext(r.Context()); ok {
            l = l.With("tenant_id", t.ID)
        }
        if c := auth.ClaimsFromContext(r.Context()); c != nil {
            l = l.With("user_id", c.UserID)
        }
        next.ServeHTTP(w, r.WithContext(logging.Into(r.Context(), l)))
    })
}

func RequestIDFromContext(ctx context.Context) string { /* read WithRequestID value */ }
```

> **This is the fix for the orphaned correlation id.** Previously `Recover` minted a *fresh* UUID per panic, so the id a user saw in a 500 matched nothing in the logs. Now `Recover` reads `RequestIDFromContext` — the id in the 500 body is the same id on every log line and in the response header (`backend-golang.md`).

---

## 3. Access Log Middleware

```go
// internal/obs/accesslog.go — one structured line per request.
func AccessLog(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        sw := &statusWriter{ResponseWriter: w, status: 200} // capture status code
        next.ServeHTTP(sw, r)

        logging.From(r.Context()).Info("http_request",
            "method", r.Method,
            "path", r.URL.Path,            // path only — NEVER the query string (may carry tokens)
            "status", sw.status,
            "duration_ms", time.Since(start).Milliseconds(),
            "ip", r.Header.Get("CF-Connecting-IP"),
            // request_id/tenant_id/user_id are already on the logger from RequestID+Enrich.
        )
    })
}
```

---

## 4. Logging Hygiene — what must NEVER be logged

> A log aggregator is a lower-trust store than your DB. Treat logs as potentially exfiltrated. Never log:
> - Credentials/secrets: passwords, `Authorization`/`Bearer` tokens, `__session`/`__refresh` cookies, `X-Internal-Secret`, JWT contents, API keys.
> - **Full URLs with query strings** (tokens ride in query params on some flows) — log `r.URL.Path`, not `RequestURI`.
> - PII beyond what's needed: prefer `user_id` (a UUID) over email; never log request bodies wholesale.
>
> Log **identifiers**, not **contents**. `user_id` + `request_id` are enough to reconstruct context from the DB without storing sensitive data in the log stream.

---

## 5. Audit Log — persisted, tenant-scoped, append-only

The audit trail is a security artifact, not a debug aid. It answers "who did this" months later, survives log rotation, and is shown to tenant admins. Several places in the skill already say "must be audit-logged" (`rbac.md` superuser bypass, `ImpersonateTenant`) — this is the one mechanism they all use.

### Table (follows the RLS checklist + append-only)

```sql
-- migrations/00NN_create_audit_log.up.sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id    UUID,                     -- tenant_user (or platform user) who acted; null = system
    actor_kind  VARCHAR(20) NOT NULL,     -- tenant_user | platform_user | system
    action      VARCHAR(100) NOT NULL,    -- e.g. user.invite, role.update, media.delete, auth.login
    target_type VARCHAR(50),              -- e.g. user, role, media
    target_id   TEXT,
    metadata    JSONB NOT NULL DEFAULT '{}', -- extra context (NO secrets/PII)
    request_id  TEXT,                     -- ← bridge to the operational logs of this action
    ip          INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON audit_log
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());

CREATE INDEX idx_audit_tenant_created ON audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_audit_tenant_action  ON audit_log(tenant_id, action);

-- APPEND-ONLY: app_user may INSERT and SELECT, but NOT UPDATE/DELETE. An audit trail
-- that can be edited or erased by the app role is worthless after a breach. The general
-- grant in database-schema.md gave app_user all four verbs — revoke the mutating two here.
REVOKE UPDATE, DELETE ON audit_log FROM app_user;
-- (Retention/pruning is done by a separate maintenance role, not the app.)
```

> **Append-only is the whole point.** RLS scopes *which* rows a tenant sees; the `REVOKE UPDATE, DELETE` makes the trail tamper-evident — even a compromised `app_user` connection cannot rewrite history. Pruning old rows (retention) runs as the migrator/maintenance role, never `app_user`.

### Recording (`internal/audit/audit.go`)

```go
type Entry struct {
    Action     string         // "media.delete"
    TargetType string         // "media"
    TargetID   string
    Metadata   map[string]any // no secrets/PII
}

// Record writes one audit row inside an RLS tx. actor + request_id + ip come from context,
// so callers can't forget them and can't spoof a different tenant (current_tenant_id()).
func (a *Audit) Record(ctx context.Context, e Entry) {
    actorID, actorKind := actorFromContext(ctx) // claims: tenant_user | platform_user | system
    err := a.store.InTenant(ctx, func(q *db.Queries) error {
        return q.InsertAudit(ctx, db.InsertAuditParams{
            ActorID:    actorID,
            ActorKind:  actorKind,
            Action:     e.Action,
            TargetType: e.TargetType,
            TargetID:   e.TargetID,
            Metadata:   toJSON(e.Metadata),
            RequestID:  obs.RequestIDFromContext(ctx), // same id as the logs + 500
            IP:         ipFromContext(ctx),
        })
    })
    if err != nil {
        // Audit failure must be loud but must NOT break the user action — log and move on.
        logging.From(ctx).Error("audit record failed", "action", e.Action, "err", err)
    }
}
```

```go
// Usage — at the point the action succeeds (e.g. in the media delete handler):
audit.Record(ctx, audit.Entry{Action: "media.delete", TargetType: "media", TargetID: id})
```

### Platform / cross-tenant actions

> A superuser bypass or impersonation (`rbac.md`) acts *on a tenant*. Record it against **that tenant's** `tenant_id` (so it shows in the affected tenant's trail) with `actor_kind = platform_user`. Because the action runs through `store.InTenant` with the target tenant in context, this is automatic. Platform staff can additionally review across tenants via the **platform pool** (`platform_user`/`BYPASSRLS`, `backend-golang.md`) — that read bypasses RLS to span all tenants, the same separation used elsewhere.

---

## 6. Dashboard Display

Tenant admins view their own trail; RLS guarantees they see only their tenant's rows.

```go
// GET /api/dashboard/audit?cursor=...&action=... — paginated, RLS-scoped.
r.With(RequirePermission("audit:read")).Get("/api/dashboard/audit", auditHandler.List)
```

```go
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
    var rows []db.AuditLog
    err := h.store.InTenant(r.Context(), func(q *db.Queries) error {
        var e error
        rows, e = q.ListAudit(r.Context(), db.ListAuditParams{Limit: 50, Cursor: cursorFrom(r)})
        return e
    })
    if err != nil {
        httperr.Write(w, httperr.Internal("Could not load audit log"))
        return
    }
    json.NewEncoder(w).Encode(rows)
}
```

Rendered with `BaseTable` (`ui-dashboard.md`) — columns: time, actor, action, target, and the `request_id` (so support can quote it to correlate with operational logs).

```vue
<!-- dashboard audit view -->
<BaseTable :columns="cols" :rows="entries">
  <template #cell-actor="{ row }">{{ row.actor_kind }}:{{ row.actor_id }}</template>
  <template #cell-request_id="{ value }"><code class="text-xs">{{ value }}</code></template>
</BaseTable>
```

`audit:read` joins the permission list in `rbac.md` (typically owner-tenant/admin only).

---

## 7. Metrics & Alerts

Detailed metrics, dashboards, SLOs, and alert rules live in `metrics-alerting.md`.

- Prefer Prometheus (`/metrics`, pull-based) or OpenTelemetry (OTLP).
- **Serve `/metrics` on the internal listener** (Cloudflare Tunnel, `deployment.md`), never on the public origin — it leaks internal topology and per-tenant volume.
- Watch **label cardinality**: a `tenant_id` label on every metric explodes series count at scale; aggregate, or sample, or keep per-tenant detail in logs/DB instead.
- Track RED metrics (rate/errors/duration), DB pool, Redis, rate-limit blocks, billing webhooks, worker queues, media conversion, onboarding, backup freshness, and email delivery.
