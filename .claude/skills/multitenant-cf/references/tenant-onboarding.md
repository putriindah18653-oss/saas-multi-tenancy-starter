# Tenant Onboarding — Provisioning a New Tenant

Creating a tenant is the one operation that touches *every* system at once: PostgreSQL (tenant, roles, owner user, trial subscription), Cloudflare KV (tenant config), the Cloudflare Worker (custom-domain registration), and email. The hard part isn't any single step — it's that **only PostgreSQL is transactional**; the CF API calls and email are not. Get the split wrong and a failure halfway through leaves a half-created tenant.

> **This is the canonical onboarding flow. It supersedes the two earlier partial sketches** — `rbac.md` → `CreateTenant` (which illustrates role seeding) and `deployment.md` → `OnboardNewTenant` (which illustrates the CF-API mechanics). Both of those put external calls in the wrong place (a CF KV call inside the DB transaction; synchronous side-effects with no retry). Use this instead.

It reuses: `store.InTenant` (`backend-golang.md`), the platform pool for the tenant row (`backend-golang.md`), the trial subscription (`billing.md`), KV + domain registration (`frontend-vue-cloudflare.md`, `deployment.md`), asynq (`backend-golang.md`), one-time codes (`backend-golang.md`), and `audit_log` (`observability.md`).

---

## Two Triggers, One Function

Both entry points converge on a single `OnboardTenant` so there is exactly one provisioning path to keep correct:

- **Self-service signup** — public, unauthenticated `POST /api/signup`. New tenant starts on **trial**. `actor_kind = system` in the audit trail.
- **Platform-admin create** — `RequirePlatformRole("superuser","admin")`, for sales-led / enterprise tenants. Same function, `actor_kind = platform_user`, and the admin may set a plan other than the trial default.

```go
// One canonical entry point. The trigger only differs in who calls it and the audit actor.
func (s *OnboardingService) OnboardTenant(ctx context.Context, in OnboardInput) (*Tenant, *httperr.AppError) {
    if appErr := validateOnboard(&in); appErr != nil { // slug format + uniqueness, owner email
        return nil, appErr
    }
    tenant, appErr := s.provisionCore(ctx, in) // transactional PG core (below)
    if appErr != nil {
        return nil, appErr
    }
    // External side-effects run async — NOT in the request, NOT in the DB tx (below).
    _ = s.jobs.EnqueueProvision(ctx, ProvisionPayload{TenantID: tenant.ID})
    return tenant, nil // 202-style: tenant exists, status=provisioning, finishing in the background
}
```

---

## The Cross-System Problem

> **PG is transactional; Cloudflare API and email are not.** You cannot wrap `cfService.SetKV(...)` or `mailer.Send(...)` in `tx.Begin/Commit` — a network call can't roll back. So onboarding is split in two:
> 1. A **transactional PG core** — everything that *can* be atomic (tenant + roles + owner + trial sub) commits together or not at all.
> 2. An **async provisioning job** — the non-transactional external side-effects (KV, domain, email), made **idempotent** so the job can retry safely until they all succeed.
>
> The tenant row commits with `status = 'provisioning'`; the async job flips it to `trial` once external setup lands. A failure between the two leaves a *recoverable* tenant (retriable job), never a corrupt one.

---

## 1. Transactional PG Core

A boundary case unique to onboarding: the owner user, roles, and trial subscription are **tenant-scoped (RLS)**, but they're created *before the tenant has ever been used*. The order is forced: insert the **tenant row first** (platform-level, no RLS, via the platform pool), then set the tenant context to the new id and insert the RLS-scoped rows via `store.InTenant`.

```go
func (s *OnboardingService) provisionCore(ctx context.Context, in OnboardInput) (*Tenant, *httperr.AppError) {
    // (a) Insert the tenant row — PLATFORM-level table (no RLS), via the platform pool.
    //     status='provisioning' until the async job completes external setup.
    //     (var is `t`, not `tenant` — the latter is the imported package used just below.)
    t, err := s.platform.CreateTenant(ctx, CreateTenantParams{
        Slug: in.Slug, Name: in.Name, OwnerEmail: in.OwnerEmail,
        Status: "provisioning", Plan: in.Plan, // Plan defaults to the trial plan for self-service
    })
    if err != nil {
        if isUniqueViolation(err) { // slug already taken
            return nil, httperr.Validation(map[string]string{"slug": "already taken"})
        }
        return nil, httperr.Internal("could not create tenant")
    }

    // (b) Now the tenant exists → bind context to its id and insert all RLS-scoped rows in ONE tx.
    ctx = tenant.WithTenant(ctx, &tenant.Tenant{ID: t.ID})
    appErr := s.store.InTenant(ctx, func(q *db.Queries) error {
        // Seed system roles (rbac.md): owner-tenant (["*"]) + admin.
        if err := q.SeedSystemRoles(ctx, t.ID); err != nil { return err }
        // First user = owner-tenant. Password hashed; nil if SSO-only.
        if err := q.CreateOwnerUser(ctx, db.CreateOwnerUserParams{
            Email: in.OwnerEmail, Name: in.OwnerName, PasswordHash: hash(in.Password),
            Role: "owner-tenant",
        }); err != nil { return err }
        // Trial subscription (billing.md): status=trial, trial_end = now + 14d.
        if err := q.CreateTrialSubscription(ctx, db.CreateTrialSubscriptionParams{
            PlanCode: in.Plan, TrialEnd: time.Now().Add(14 * 24 * time.Hour),
        }); err != nil { return err }
        // Register owner-domain rows in tenant_domains (public + dashboard).
        return q.RegisterDomains(ctx, defaultDomainsFor(in))
    })
    if appErr != nil {
        // Tenant row exists but the RLS rows failed → roll it back so we don't leak a shell tenant.
        _ = s.platform.DeleteTenant(ctx, t.ID)
        return nil, httperr.Internal("could not initialize tenant")
    }

    s.audit.Record(ctx, audit.Entry{Action: "tenant.created", TargetType: "tenant", TargetID: t.ID})
    return t, nil
}
```

> Why the tenant row isn't in the same `store.InTenant` tx: `tenants` is a platform table with **no RLS**, written by the platform pool — `store.InTenant` (app pool) can't insert it, and `current_tenant_id()` wouldn't be set yet anyway. The two writes are sequential, with an explicit compensating delete if the second fails. (A single transaction spanning both pools is possible but couples the pools; the compensating delete is simpler and the window is tiny.)

---

## 2. Async Provisioning Job

The external side-effects run in an asynq worker (`backend-golang.md`). Each step is **idempotent** so a retry after partial failure re-runs cleanly without duplicating anything.

```go
func (w *Worker) HandleProvision(ctx context.Context, p ProvisionPayload) error {
    ctx = tenant.WithTenant(ctx, &tenant.Tenant{ID: p.TenantID})
    t, err := w.loadTenant(ctx, p.TenantID)
    if err != nil { return err } // asynq retries; if the tenant's gone, no-op

    // (a) Write tenant config to CF KV — idempotent: PUT overwrites, safe to repeat.
    cfg := buildTenantConfig(t) // tenant_id, API key, branding, features from plan
    if err := w.cf.SetKV(ctx, "CACHE_KV", "tenant:"+t.PublicDomain, cfg); err != nil { return err }
    if err := w.cf.SetKV(ctx, "CACHE_KV", "tenant:"+t.DashboardDomain, cfg); err != nil { return err }

    // (b) Register a custom domain on the Workers (only if the tenant brought its own domain).
    //     Idempotent: check-then-add — adding an existing domain is a no-op / handled error.
    if t.HasCustomDomain {
        if err := w.cf.EnsureDomain(ctx, "vue-public", t.PublicDomain); err != nil { return err }
        if err := w.cf.EnsureDomain(ctx, "vue-dashboard", t.DashboardDomain); err != nil { return err }
    }

    // (c) Welcome + email verification (soft — see below). Dedupe so retries don't re-send.
    if err := w.sendWelcomeOnce(ctx, t); err != nil { return err }

    // (d) Flip status provisioning → trial. Tenant is now fully usable.
    if err := w.store.InTenant(ctx, func(q *db.Queries) error {
        return q.ActivateTrial(ctx, p.TenantID) // tenants.status = 'trial'
    }); err != nil { return err }

    w.audit.Record(ctx, audit.Entry{Action: "tenant.provisioned", TargetType: "tenant", TargetID: p.TenantID})
    return nil
}
```

> **Partial-failure is the normal case, not the exception.** If the job dies after writing KV but before sending email, asynq retries the whole handler — step (a) re-PUTs the same KV (harmless), step (b) sees the domain already exists (no-op), step (c) sees the email already sent (skip), step (d) sets trial. Idempotency is what makes "retry the whole thing" correct. A sweeper also re-enqueues any tenant stuck in `provisioning` past a threshold.

---

## 3. Soft Email Verification (non-blocking)

The tenant goes straight to **trial with full access** — verification does not gate activation (low signup friction). The owner gets a verification link; clicking it sets a flag. You can *enforce* verification later (e.g. before upgrading from trial) without changing the onboarding path.

```go
// Issue a one-time code (Redis, backend-golang.md) and email the link. Non-blocking.
func (w *Worker) sendWelcomeOnce(ctx context.Context, t *Tenant) error {
    code := randToken()
    // store: otc:verify_email:{code} → tenant_id, single-use, e.g. 48h TTL
    if err := w.cache.PutOneTimeCode(ctx, "verify_email", code, t.ID, 48*time.Hour); err != nil {
        return err
    }
    return w.mailer.SendWelcome(ctx, t.OwnerEmail, t.Name, verifyURL(t, code)) // mailer dedupes by tenant
}

// GET /api/auth/verify-email?code=... — public route, marks email_verified.
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
    tenantID, err := h.cache.ConsumeOneTimeCode(r.Context(), "verify_email", r.URL.Query().Get("code"))
    if err != nil { // wrong/expired/used
        httperr.Write(w, &httperr.AppError{Status: 400, Code: "invalid_code", Message: "Link is invalid or expired"})
        return
    }
    ctx := tenant.WithTenant(r.Context(), &tenant.Tenant{ID: tenantID})
    _ = h.store.InTenant(ctx, func(q *db.Queries) error { return q.MarkEmailVerified(ctx) })
    // redirect to the dashboard with a success notice
}
```

`email_verified` lives on `tenants` (see `database-schema.md`). The verify route is public (no session yet at signup time).

---

## 4. Input Validation

Validate at the boundary (`backend-golang.md` → `decodeAndValidate`) before any write:

```go
type OnboardInput struct {
    Slug       string `json:"slug"       validate:"required,lowercase,min=3,max=63,hostname_rfc1123"`
    Name       string `json:"name"       validate:"required,max=255"`
    OwnerEmail string `json:"owner_email" validate:"required,email,max=255"`
    OwnerName  string `json:"owner_name" validate:"required,max=255"`
    Password   string `json:"password"   validate:"required,min=8,max=200"`
    Plan       string `json:"plan"       validate:"omitempty,oneof=starter pro enterprise"`
    CustomDomain string `json:"custom_domain" validate:"omitempty,fqdn"`
}
```

- **Slug** must be a valid hostname label (it becomes `{slug}.portalonline.id`) and **unique** — uniqueness is enforced by the DB constraint, surfaced as a `422` field error (not a 500).
- **Custom domain**, if provided, drives the async domain-registration step; if absent, the owner domain `{slug}.portalonline.id` is used.

---

## Step Summary

```
POST /api/signup (self-service)  ─┐
RequirePlatformRole admin create ─┴─▶ OnboardTenant
  1. validate (slug/email)                              [reject → 422]
  2. provisionCore  (ONE pg path, mixed pools):
       a. INSERT tenant (platform pool, no RLS, status=provisioning)
       b. store.InTenant tx: seed roles + owner user + trial sub + domains   [fail → delete tenant]
       audit: tenant.created
  3. enqueue provision job → return (tenant exists, status=provisioning)
        ── async (asynq, idempotent, retriable) ──
  4. CF KV config (public + dashboard)
  5. register custom domain on Workers (if any)
  6. welcome + verification email (deduped)
  7. status provisioning → trial    ← tenant fully usable
     audit: tenant.provisioned
  (soft) owner clicks verify link → email_verified = true   [non-blocking]
```

Every step writes to `audit_log` (`observability.md`): `actor_kind = system` for self-service signup, `platform_user` for admin-created. The whole flow is the front door to a tenant's lifecycle — the billing state machine (`billing.md`) takes over from `trial` onward.
