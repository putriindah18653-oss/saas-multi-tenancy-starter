# RBAC — Multi-Tenant Roles & Permissions

## Role Hierarchy

```
PLATFORM (platform owner)
├── superuser     → full access: all tenants, system configuration, billing
└── admin         → tenant management, support, CANNOT change system/billing

TENANT (per-tenant, stored in the DB)
├── owner-tenant  → full access within their tenant, manage members & tenant billing
├── admin         → full operations, CANNOT delete tenant / change billing
└── [custom role] → defined by owner-tenant, has its own permission matrix
```

---

## Database: Custom Roles per Tenant

```sql
-- Seed system roles when a new tenant is created
INSERT INTO tenant_roles (tenant_id, name, permissions, is_system) VALUES
  ($1, 'owner-tenant', '["*"]', TRUE),  -- wildcard = all permissions
  ($1, 'admin', '["content:*","user:read","user:invite"]', TRUE);

-- Custom role created by owner-tenant
INSERT INTO tenant_roles (tenant_id, name, permissions, is_system) VALUES
  ($1, 'editor', '["content:write","content:publish","content:read"]', FALSE),
  ($1, 'viewer', '["content:read"]', FALSE);
```

### Permission Format
```
{resource}:{action}
  content:read       → read content
  content:write      → create/edit content
  content:publish    → publish content
  content:delete     → delete content
  user:read          → view the user list
  user:invite        → invite a new user
  user:manage        → edit/delete user
  role:manage        → manage custom roles
  billing:read       → view billing info
  billing:manage     → change subscription
  settings:read      → view tenant settings
  settings:manage    → change tenant settings
  media:read         → list/view uploaded media
  media:upload       → upload images (see media-upload.md)
  media:delete       → delete media (removes row + bucket objects)
  audit:read         → view the tenant audit log (see observability.md)
  *                  → all permissions (owner-tenant)
  content:*          → all actions for content
```

---

## Two Authorization Axes (platform vs tenant)

There are **two separate authorization systems**, and they are not interchangeable. The `IsPlatform` claim selects which one applies:

| | Platform authz | Tenant authz |
|---|---|---|
| **Who** | platform owner staff (`IsPlatform = true`) | tenant members (`IsPlatform = false`) |
| **Model** | **role membership** — is the role in an allowed set? | **permission matrix** — does the permission list satisfy `resource:action`? |
| **Guard** | `RequirePlatformRole("superuser", "admin")` | `RequirePermission("content:write")` |
| **Granularity** | coarse (2 fixed roles) | fine (per-permission, custom roles per tenant) |
| **Source of truth** | fixed in code | `tenant_roles.permissions` in the DB, per tenant |
| **Token** | platform JWT (no `tenant_id`) | tenant JWT (always has `tenant_id`) |

Rules that keep the two from leaking into each other:
- A **platform** route must use `RequirePlatformRole` and never `RequirePermission` (platform tokens carry no tenant permission matrix).
- A **tenant** route must use `RequirePermission` (+ `TenantContextMiddleware`) and never `RequirePlatformRole`.
- The two guards both read the *same* `Claims`, but branch on `IsPlatform`. A tenant user can never satisfy `RequirePlatformRole` because their token has `IsPlatform = false`; a platform user satisfies `RequirePermission` only via the explicit superuser bypass below.

---

## Where Claims Come From — AuthMiddleware (JWT verification)

Every guard below reads `Claims` via `ClaimsFromContext`. Those claims are produced by **verifying the JWT signature** on the incoming `Authorization: Bearer` token — this is the real security boundary (see `auth-sso.md` → "Trust Model"). The backend does **not** trust `X-User-*` headers.

### `internal/auth/middleware.go`
```go
// AuthMiddleware — verify the JWT signature and put the resulting Claims into context.
// The signature check is what makes a forged "X-User-Role: superuser" header worthless.
func AuthMiddleware(jwt *JWTService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            raw := bearerToken(r) // strip "Bearer " from the Authorization header
            if raw == "" {
                httperr.Write(w, httperr.Unauthorized())
                return
            }

            // Verify signature + standard claims (exp, etc). On any failure → 401.
            claims, err := jwt.Verify(raw)
            if err != nil {
                httperr.Write(w, httperr.Unauthorized())
                return
            }

            ctx := WithClaims(r.Context(), claims) // ClaimsFromContext reads this
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## Golang: Middleware RBAC

### `internal/auth/middleware.go`
```go
package auth

import (
    "net/http"
    "strings"
    "github.com/yourapp/internal/tenant"
    "github.com/yourapp/internal/httperr" // envelope error helpers (backend-golang.md)
)

// TenantRoleMiddleware — validate the role within the tenant context
// Called AFTER ResolveTenantMiddleware and AuthMiddleware
func RequirePermission(permission string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := ClaimsFromContext(r.Context())
            if claims == nil {
                httperr.Write(w, httperr.Unauthorized())
                return
            }

            // DELIBERATE cross-axis exception: a platform superuser satisfies any
            // tenant permission check (for support/incident access). This is the ONLY
            // place a platform identity crosses into the tenant permission model.
            // It MUST be audit-logged (who, tenant, route, when) — see callout below.
            if claims.IsPlatform && claims.Role == "superuser" {
                next.ServeHTTP(w, r)
                return
            }

            // Check whether the user has the permission
            if !hasPermission(claims.Permissions, permission) {
                httperr.Write(w, httperr.Forbidden())
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func hasPermission(userPerms []string, required string) bool {
    for _, p := range userPerms {
        if p == "*" { return true }                    // full wildcard
        if p == required { return true }               // exact match

        // Per-resource wildcard: "content:*" matches "content:write"
        if strings.HasSuffix(p, ":*") {
            resource := strings.TrimSuffix(p, ":*")
            if strings.HasPrefix(required, resource+":") { return true }
        }
    }
    return false
}

// PlatformRoleMiddleware — platform routes only
func RequirePlatformRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := ClaimsFromContext(r.Context())
            if claims == nil || !claims.IsPlatform {
                httperr.Write(w, httperr.Forbidden())
                return
            }
            for _, role := range roles {
                if claims.Role == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            httperr.Write(w, httperr.Forbidden())
        })
    }
}
```

> **Audit the superuser bypass.** Because a platform superuser can satisfy any tenant `RequirePermission` check, that path is the highest-privilege action in the system. Record every bypass via `audit.Record` into the append-only `audit_log` (`observability.md`) — `actor_kind=platform_user`, the affected tenant, action, target, and `request_id`. Prefer scoped impersonation (a short-lived tenant token — see `backend-golang.md` → `ImpersonateTenant`) over raw superuser access for routine support work; that too is audit-logged.

### Usage in the Router
```go
// Example with the Chi router
r := chi.NewRouter()

// Tenant API routes
r.Group(func(r chi.Router) {
    r.Use(ResolveTenantMiddleware(tenantRepo))
    r.Use(AuthMiddleware(jwtService))
    r.Use(TenantContextMiddleware()) // ensure the user belongs to the correct tenant

    // All users can read content
    r.Get("/api/content", contentHandler.List)
    r.Get("/api/content/{id}", contentHandler.Get)

    // Requires a specific permission
    r.With(RequirePermission("content:write")).Post("/api/content", contentHandler.Create)
    r.With(RequirePermission("content:publish")).Put("/api/content/{id}/publish", contentHandler.Publish)
    r.With(RequirePermission("content:delete")).Delete("/api/content/{id}", contentHandler.Delete)

    // User management
    r.With(RequirePermission("user:read")).Get("/api/users", userHandler.List)
    r.With(RequirePermission("user:invite")).Post("/api/users/invite", userHandler.Invite)

    // Role management (owner-tenant only)
    r.With(RequirePermission("role:manage")).Route("/api/roles", func(r chi.Router) {
        r.Get("/", roleHandler.List)
        r.Post("/", roleHandler.Create)
        r.Put("/{id}", roleHandler.Update)
        r.Delete("/{id}", roleHandler.Delete)
    })
})

// Platform routes
r.Group(func(r chi.Router) {
    r.Use(AuthMiddleware(jwtService))
    r.Use(RequirePlatformRole("superuser", "admin"))

    r.Get("/platform/tenants", platformHandler.ListTenants)
    r.Post("/platform/tenants", platformHandler.CreateTenant)

    // superuser only
    r.With(RequirePlatformRole("superuser")).Delete("/platform/tenants/{id}", platformHandler.DeleteTenant)
    r.With(RequirePlatformRole("superuser")).Get("/platform/settings", platformHandler.GetSettings)
})
```

---

## Tenant Context Validation

```go
// Ensure the user can only access data in their own tenant
func TenantContextMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            t, ok := tenant.FromContext(r.Context())
            if !ok {
                // Programming error: tenant middleware didn't run before this. Generic 500.
                httperr.Write(w, &httperr.AppError{Status: 500, Code: "internal_error", Message: "Something went wrong"})
                return
            }

            claims := ClaimsFromContext(r.Context())
            if claims == nil {
                httperr.Write(w, httperr.Unauthorized())
                return
            }

            // Ensure JWT tenant_id == resolved tenant_id. A mismatch = a token from another
            // tenant used against this host → forbidden (the token IS valid, just not here).
            if claims.TenantID != t.ID {
                httperr.Write(w, httperr.Forbidden())
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

> **403 (route authz) vs 404 (resource) — two different non-leak rules.** These middlewares guard *routes*: the caller is authenticated and the route's existence isn't a secret, they just lack the permission/role → `403 forbidden` is correct and intended. That is distinct from *resource* access inside a handler: for a tenant-scoped row that exists but belongs to another tenant, return `404 not_found`, never `403` — a `403` would confirm the row exists and enable cross-tenant ID enumeration. RLS makes other tenants' rows invisible, so a cross-tenant lookup returns zero rows → 404 naturally (see `backend-golang.md` → Error Handling & Validation).

---

## Vue: Route Guard per Permission

> **The Vue `hasPermission` is cosmetic, not a security boundary.** It hides buttons and routes the user can't use — a UX nicety. The authoritative check is the Go `hasPermission` in the backend middleware; the client cannot be trusted (anyone can edit JS or call the API directly). Two consequences:
> - **Every** permission enforced in the UI MUST also be enforced by a `RequirePermission` on the backend route. Never rely on the hidden button alone.
> - The wildcard-matching logic is **duplicated** (Go `hasPermission` ↔ Vue `hasPermission`). They must stay in sync: `*` = all, exact match, and `resource:*` prefix. If you change the matching rules in one, change the other — otherwise the UI shows/hides the wrong controls. Consider this duplication a known maintenance cost, with Go as the source of truth.

### `composables/useAuth.ts`
```typescript
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function useAuth() {
  const store = useAuthStore()

  const hasPermission = (permission: string): boolean => {
    const perms = store.user?.permissions ?? []
    if (perms.includes('*')) return true
    if (perms.includes(permission)) return true
    // wildcard resource: content:* matches content:write
    const [resource] = permission.split(':')
    return perms.includes(`${resource}:*`)
  }

  const isPlatform = computed(() => store.user?.is_platform ?? false)
  const isSuperuser = computed(() => isPlatform.value && store.user?.role === 'superuser')
  const isOwnerTenant = computed(() => store.user?.role === 'owner-tenant')

  return { hasPermission, isPlatform, isSuperuser, isOwnerTenant }
}
```

### `router/index.ts` — Route Guard
```typescript
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  const { hasPermission } = useAuth()

  // Public route
  if (to.meta.public) return next()

  // Not logged in
  if (!authStore.isAuthenticated) {
    return next({ name: 'login', query: { next: to.fullPath } })
  }

  // Check permission if the route has meta.permission
  if (to.meta.permission && !hasPermission(to.meta.permission as string)) {
    return next({ name: 'forbidden' })
  }

  // Platform-only routes
  if (to.meta.platformOnly && !authStore.user?.is_platform) {
    return next({ name: 'forbidden' })
  }

  next()
})
```

### Route Definition
```typescript
const routes = [
  {
    path: '/content',
    component: ContentListPage,
    meta: { permission: 'content:read' }
  },
  {
    path: '/content/create',
    component: ContentCreatePage,
    meta: { permission: 'content:write' }
  },
  {
    path: '/users',
    component: UsersPage,
    meta: { permission: 'user:read' }
  },
  {
    path: '/roles',
    component: RolesPage,
    meta: { permission: 'role:manage' }
  },
  {
    path: '/settings/billing',
    component: BillingPage,
    meta: { permission: 'billing:read' }
  },
]
```

### Component with Permission Check
```vue
<template>
  <div>
    <button v-if="hasPermission('content:write')" @click="createContent">
      Create Content
    </button>
    <button v-if="hasPermission('content:publish')" @click="publishContent">
      Publish
    </button>
    <!-- Displayed differently for owner-tenant -->
    <section v-if="isOwnerTenant">
      <h3>Tenant Settings</h3>
    </section>
  </div>
</template>

<script setup lang="ts">
const { hasPermission, isOwnerTenant } = useAuth()
</script>
```

---

## Role Initialization When a New Tenant Is Created

> This shows only the **role-seeding** part of tenant creation. The full onboarding flow — validation, the platform-vs-RLS write ordering, trial subscription, async CF/email provisioning, and status handling — is in `tenant-onboarding.md`. Seeding runs inside that flow's `store.InTenant` transaction (RLS-scoped to the new tenant); do NOT make external calls (CF KV, email) inside a DB transaction — those are the async provisioning step.

```go
// Seed the system roles for a new tenant. Called inside store.InTenant (RLS active for the
// new tenant) as part of provisionCore (tenant-onboarding.md), NOT as a standalone tx.
func SeedSystemRoles(ctx context.Context, q *db.Queries) error {
    // owner-tenant = full access; admin = scoped operations. is_system=true → cannot be deleted.
    systemRoles := []struct{ name, perms string }{
        {"owner-tenant", `["*"]`},
        {"admin", `["content:*","user:read","user:invite","settings:read"]`},
    }
    for _, r := range systemRoles {
        if err := q.CreateSystemRole(ctx, db.CreateSystemRoleParams{
            Name: r.name, Permissions: []byte(r.perms), IsSystem: true,
        }); err != nil {
            return err
        }
    }
    return nil
}
```

The first user (owner-tenant), trial subscription, and domain rows are seeded in the same transaction — see `tenant-onboarding.md` → "Transactional PG Core" for the complete sequence.