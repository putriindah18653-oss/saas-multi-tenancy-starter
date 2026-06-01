# Billing & Subscriptions — Plans, Payment Gateways, Webhooks

Subscription billing for a multi-tenant SaaS: a platform-managed plan catalog, tenant subscriptions, and payments through **Midtrans**, **Duitku**, or **manual bank/cash transfer**. This is a money path, so signature verification, idempotency, and never trusting the client are not optional.

It builds on existing hooks: `tenants.plan`/`tenants.status` (`database-schema.md`), `billing:read`/`billing:manage` permissions (`rbac.md`), `features[]` + `hasFeature()` (`frontend-vue-cloudflare.md`), Redis idempotency (`backend-golang.md`), and `audit_log` (`observability.md`).

---

## The Platform vs Tenant Split

> **Who sets prices and features is the platform owner — not the tenant.** This determines where each table lives, and it's the single most important design decision here:
> - **`plans` catalog** (starter/pro/enterprise, their prices + features) is **platform-level**: NO `tenant_id`, NO RLS — same class as `tenants`/`platform_users` (`database-schema.md`). Edited only by superuser/admin via the **platform pool** (`platform_user`, `backend-golang.md`) and the platform dashboard.
> - **`subscriptions` / `invoices` / `payments`** are **tenant-scoped**: full RLS, accessed via `store.InTenant`. A tenant *chooses* a plan and pays; it never edits the catalog.

Mixing these up is the classic multi-tenant billing bug — letting a tenant write plan prices, or putting the shared catalog behind RLS so the platform can't manage it.

---

## Schema

### Platform catalog — `plans` (NO RLS)

```sql
-- Platform-level, like tenants/platform_users — NO tenant_id, NO RLS.
-- Managed via the platform pool (platform_user) + platform dashboard.
CREATE TABLE plans (
    code           VARCHAR(30) PRIMARY KEY,         -- starter | pro | enterprise
    name           VARCHAR(100) NOT NULL,
    price_monthly  BIGINT NOT NULL,                 -- minor units (IDR has none → rupiah as integer)
    currency       CHAR(3) NOT NULL DEFAULT 'IDR',
    features       JSONB NOT NULL DEFAULT '[]',     -- ["comments","analytics",...] → drives hasFeature()
    limits         JSONB NOT NULL DEFAULT '{}',     -- {"max_users":10,"max_storage_mb":5000}
    sort_order     INT NOT NULL DEFAULT 0,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,    -- inactive = hidden from the picker, existing subs keep working
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);
-- No ENABLE ROW LEVEL SECURITY here — this is a shared platform table.
-- Seed the three plans; annual price is DERIVED (see Pricing), not stored.
```

### Tenant subscription — source of truth + history (RLS)

```sql
CREATE TABLE subscriptions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_code        VARCHAR(30) NOT NULL REFERENCES plans(code),
    billing_cycle    VARCHAR(10) NOT NULL,           -- monthly | annual
    status           VARCHAR(20) NOT NULL,           -- trial | active | past_due | suspended | canceled
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,        -- renewal / expiry boundary
    trial_end        TIMESTAMPTZ,
    canceled_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions FORCE  ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON subscriptions
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());
CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id, status);

CREATE TABLE invoices (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    amount        BIGINT NOT NULL,                   -- minor units, computed server-side
    currency      CHAR(3) NOT NULL DEFAULT 'IDR',
    status        VARCHAR(20) NOT NULL,              -- pending | paid | failed | expired | refunded
    due_at        TIMESTAMPTZ NOT NULL,
    paid_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoices FORCE  ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON invoices
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());
CREATE INDEX idx_invoices_tenant_status ON invoices(tenant_id, status);

CREATE TABLE payments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id    UUID NOT NULL REFERENCES invoices(id),
    gateway       VARCHAR(20) NOT NULL,              -- midtrans | duitku | manual
    gateway_ref   TEXT,                              -- order_id / merchantOrderId / transaction id
    amount        BIGINT NOT NULL,
    status        VARCHAR(20) NOT NULL,              -- pending | settled | failed | refunded
    raw_event     JSONB,                             -- last verified webhook payload (no secrets)
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (gateway, gateway_ref)                    -- dedupe gateway retries (+ Redis idempotency)
);
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE payments FORCE  ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON payments
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());
```

### Denormalized cache on `tenants`

```sql
-- subscriptions is the source of truth; these columns are a fast-read CACHE for entitlement
-- checks on every request, and what gets synced to Cloudflare KV (tenant config).
ALTER TABLE tenants ADD COLUMN billing_cycle       VARCHAR(10);
ALTER TABLE tenants ADD COLUMN current_period_end  TIMESTAMPTZ;
-- tenants.plan + tenants.status already exist (database-schema.md); status now also carries
-- past_due | suspended | canceled in addition to active | trial.
```

> Keep the cache in sync: whenever a subscription changes (purchase, renewal, webhook, suspension), update `tenants.{plan,status,billing_cycle,current_period_end}` in the **same transaction**, then push the tenant config to KV (`deployment.md` onboarding pattern). Entitlement checks read the cache, not a join.

---

## Pricing — money as integers, annual = 2 months free

> **Two rules, both load-bearing:**
> 1. **Money is an integer in minor units + explicit currency.** Never float (rounding corrupts money). IDR has no minor unit, so the integer is rupiah; the pattern still holds for other currencies.
> 2. **Annual = monthly × 10** — pay for 10 months, get 12. That *is* the "2 months free" discount. Derive it; never store an annual price that can drift out of sync with the monthly one.

```go
// AnnualPrice — the 2-months-free rule, in one place. monthly is minor units.
// 12 months billed as 10 → the customer saves 2 months by paying annually.
const annualMonthsCharged = 10

func AnnualPrice(monthly int64) int64 { return monthly * annualMonthsCharged }

// PriceFor returns what to charge for a plan + cycle (minor units).
func PriceFor(p Plan, cycle string) (int64, error) {
    switch cycle {
    case "monthly":
        return p.PriceMonthly, nil
    case "annual":
        return AnnualPrice(p.PriceMonthly), nil // = monthly × 10
    default:
        return 0, httperr.BadRequest("invalid billing cycle")
    }
}
```

The dashboard plan picker shows both, with the annual saving made explicit — e.g. "Annual: pay 10 months, get 12 — **save 2 months**." The amount charged is always computed server-side via `PriceFor` from the catalog; the client never sends a price.

---

## Payment Gateways — one interface, three implementations

```go
// PaymentGateway abstracts the provider. Midtrans, Duitku, and manual transfer all satisfy it,
// so adding/swapping a provider doesn't touch the billing service.
type PaymentGateway interface {
    // CreateCharge starts a payment for an invoice; returns a redirect/snap URL or VA details.
    CreateCharge(ctx context.Context, inv Invoice) (*ChargeResult, error)
    // VerifyWebhook validates the signature and parses an inbound callback into a normalized event.
    VerifyWebhook(r *http.Request, body []byte) (*WebhookEvent, error)
}

type WebhookEvent struct {
    GatewayRef string // order_id / merchantOrderId
    Status     string // normalized: settled | pending | failed | expired
    Amount     int64
    Raw        map[string]any
}
```

> **Manual bank/cash transfer is a different shape.** Midtrans/Duitku auto-confirm via webhook; manual transfer has no callback — the tenant uploads/marks a transfer, and a **platform admin confirms** it, which transitions the invoice to `paid` (recorded in `audit_log` as a manual action). Model it as a gateway whose "webhook" is an authenticated admin confirmation endpoint, so the downstream activation logic is identical.

### Signature verification (the security core)

```go
// Midtrans: SHA512(order_id + status_code + gross_amount + ServerKey).
func (m *Midtrans) VerifyWebhook(r *http.Request, body []byte) (*WebhookEvent, error) {
    var p struct {
        OrderID     string `json:"order_id"`
        StatusCode  string `json:"status_code"`
        GrossAmount string `json:"gross_amount"`
        Signature   string `json:"signature_key"`
        TxStatus    string `json:"transaction_status"`
    }
    if err := json.Unmarshal(body, &p); err != nil {
        return nil, httperr.BadRequest("bad webhook body")
    }
    want := sha512Hex(p.OrderID + p.StatusCode + p.GrossAmount + m.serverKey)
    if subtle.ConstantTimeCompare([]byte(want), []byte(p.Signature)) != 1 {
        return nil, httperr.Forbidden() // forged callback — reject
    }
    return &WebhookEvent{GatewayRef: p.OrderID, Status: normalizeMidtrans(p.TxStatus) /*...*/}, nil
}

// Duitku: MD5(merchantCode + amount + merchantOrderId + apiKey). Same idea, different algo.
```

> A forged webhook = a free subscription. Signature verification is to billing what `INTERNAL_SSO_SECRET` constant-time compare is to the SSO endpoint (`auth-sso.md`): the **only** thing standing between an attacker and writing "paid" to your DB. Use `subtle.ConstantTimeCompare`, never `==`. Keep gateway server keys in env/secrets (`deployment.md`), never in logs.

---

## Webhook Handling — public path, signed, idempotent

> **The webhook endpoint is the inverse of `/api/internal/*`.** Internal endpoints are *hidden* (Tunnel + secret). Webhooks must be *publicly reachable* — Midtrans/Duitku servers POST to them with no session and no JWT — so the path is `public` in the Worker (like `/login`, see `auth-sso.md` `isPublicRoute`). Its authentication is the **signature**, not the network. Do NOT route it through the authenticated proxy.

```go
// POST /api/webhooks/payment/{gateway} — unauthenticated transport, signature-authenticated.
func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
    if err != nil {
        httperr.Write(w, httperr.BadRequest("invalid payload"))
        return
    }

    gw := h.gatewayFor(chi.URLParam(r, "gateway"))
    if gw == nil {
        httperr.Write(w, httperr.NotFound())
        return
    }

    // 1. Verify signature FIRST — reject forgeries before touching state.
    evt, err := gw.VerifyWebhook(r, body)
    if err != nil {
        httperr.Write(w, httperr.Forbidden())
        return
    }

    // 2. Apply idempotently in durable storage. Redis can be an optimization to reduce duplicate
    //    work, but it must not be the only "processed" marker on a money path: if Redis claims
    //    first and DB application then fails, a gateway retry could be swallowed. The service must
    //    insert/upsert a webhook event or payment transition with UNIQUE(gateway, gateway_ref/event_id)
    //    in the same transaction that applies the billing state change.
    result, err := h.svc.ApplyWebhookIdempotently(r.Context(), evt)
    if err != nil {
        httperr.Write(w, httperr.Internal("could not apply payment"))
        return // 5xx → gateway retries later; durable idempotency makes that safe.
    }
    if result.AlreadyProcessed {
        w.WriteHeader(http.StatusOK)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

Rules, all enforced above: **verify signature before any state change**; **idempotent in the database** (Redis is optional optimization only; DB `UNIQUE` is the durable backstop); **never trust the client's amount/status** — cross-check the verified event against the stored invoice amount before marking paid; **ack 200 only after success** so retries are safe.

---

## Subscription State Machine

```
            sign up
   (none) ─────────────▶ trial ──────────────▶ active
                            │   pay invoice        │  renewal invoice unpaid past due_at
                            │                       ▼
                            │                  past_due ──(grace ~7d, access ON + banner)
                            │                       │
                            │   still unpaid        ▼
                            └──────────────────▶ suspended ──(pay)──▶ active
   active ──(user cancels)──▶ canceled (access until current_period_end, then suspended)
```

- **trial** — new tenants get a trial (e.g. 14 days); full access, `trial_end` set.
- **active** — current period paid; `current_period_end` is the renewal boundary.
- **past_due** — renewal invoice unpaid at `due_at` → enter a **grace period (~7 days)**: access continues, dashboard shows a payment banner. This is the churn-friendly choice (a one-day-late customer isn't locked out).
- **suspended** — grace elapsed unpaid → access frozen until payment; data retained (deletion is a separate offboarding decision, see `backup-recovery.md` cascade).
- **canceled** — user opted out; access runs to `current_period_end`, then suspended.

> A scheduled job (asynq, `backend-golang.md`) sweeps subscriptions: trial→past_due at `trial_end` if unpaid, active→past_due at `due_at`, past_due→suspended after grace. Every transition updates the `tenants` cache + KV and writes an `audit_log` entry.

---

## Entitlement & Feature Gating

Entitlement reads the denormalized cache (fast, per-request), and features come from the plan:

```go
// Suspended/expired tenants keep authenticating but lose access to gated actions.
func (s *BillingService) RequireActive(ctx context.Context) *httperr.AppError {
    t := tenant.MustFromContext(ctx)
    switch t.Status {
    case "active", "trial", "past_due": // past_due is in grace → still allowed
        return nil
    default:
        return &httperr.AppError{Status: 402, Code: "subscription_inactive",
            Message: "Your subscription is not active"}
    }
}
```

The plan's `features[]` flows to the frontend `hasFeature()` (`frontend-vue-cloudflare.md`) and is enforced on the backend for feature-gated routes — same UI-cosmetic / backend-authoritative split as RBAC (`rbac.md`). Plan `limits` (max users, storage) are checked at the relevant boundaries.

---

## Dashboard

- **Platform dashboard** (superuser/admin, `RequirePlatformRole`): the **catalog editor** — set each plan's `price_monthly`, `features`, `limits`, `is_active`. Writes go through the platform pool (no RLS). This is the "harga & fitur diatur di dashboard" surface.
- **Tenant dashboard** (`billing:read` / `billing:manage`): current plan + status, the **plan picker** with a monthly/annual toggle that surfaces the *save 2 months* annual saving, invoice history, and a pay button that calls `CreateCharge` → gateway redirect/snap (or shows VA / manual-transfer instructions).

Both render with `BaseTable`/`BaseInput` (`ui-dashboard.md`). Every billing mutation — plan change, payment settled, manual confirmation, suspension — is written to `audit_log` (`observability.md`), `actor_kind` distinguishing tenant self-service from platform admin action.
