# Auth & SSO — Hybrid Flow

## Overview

| Condition | Flow | User Domain |
|---------|------|-------------|
| Tenant has a custom domain | Token Proxy (invisible SSO) | Stays on `manage.kabarsiang.id` |
| Tenant uses the owner domain | Redirect SSO | Redirect to `sso.portalonline.id` |
| Platform admin/superuser | Direct login | `manage.portalonline.id` (platform dashboard) |

---

## Trust Model — Read This First

The security boundary between the Worker and the Go backend is the **JWT signature**, not the network.

- The Worker forwards the user's `access_token` as `Authorization: Bearer <jwt>`. The backend **verifies the JWT signature** on every request and derives identity (`user_id`, `tenant_id`, `role`, `permissions`) from the verified claims. See `rbac.md` → `AuthMiddleware`.
- The Worker does **not** inject trusted `X-User-*` authz headers. A forged `X-User-Role: superuser` must be worthless, because identity comes only from a signed token.
- **Why the IP allowlist is not enough:** Nginx allow-listing Cloudflare IP ranges (see `deployment.md`) is defense-in-depth, not the boundary. Those ranges are shared by *every* Cloudflare customer — anyone's Worker can reach your origin from a Cloudflare IP. Treat origin requests as untrusted and rely on the JWT signature.
- **Two layers, two jobs:** the **edge** (Worker + `CACHE_SSO`) owns session lifecycle — revocation and expiry, checked per request. The **backend** owns identity — JWT signature verification. A revoked session is rejected at the edge; a forged token is rejected at the backend. Neither layer alone is sufficient.
- `X-Tenant-ID` is still forwarded for tenant *resolution*, but the backend cross-checks it against the JWT's `tenant_id` claim (`rbac.md` → `TenantContextMiddleware`) — it is a hint, never the authority.

---

## Cloudflare Worker: Auth Proxy Logic

### `server/index.ts` (Worker DASHBOARD)

```typescript
export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const hostname = url.hostname; // manage.kabarsiang.id

    // 1. Resolve tenant from KV
    const tenantConfig = await env.CACHE_KV.get(`tenant:${hostname}`, 'json') as TenantConfig | null;

    if (!tenantConfig) {
      return new Response('Tenant not found', { status: 404 });
    }

    // 2. Auth check for all routes except /login and /public
    if (!isPublicRoute(url.pathname)) {
      let sessionResult = await validateSession(request, env, tenantConfig);

      // Access session expired/absent but a refresh cookie exists → try silent refresh.
      // This is what keeps an ACTIVE user logged in seamlessly: the 15m access token is
      // renewed behind the scenes, invisible to the Vue app, as long as the refresh token
      // is still inside its 12h sliding window (and under the 24h absolute cap, enforced
      // server-side). An IDLE user simply stops refreshing → next action re-logs in.
      let refreshed: RefreshResult | null = null;
      if (!sessionResult.valid && getCookie(request, '__refresh')) {
        refreshed = await trySilentRefresh(request, env, tenantConfig);
        if (refreshed) sessionResult = { valid: true, token: refreshed.accessToken };
      }

      if (!sessionResult.valid) {
        // Silent refresh failed too → session genuinely ended. Respond by REQUEST TYPE:
        //   - API/XHR calls (fetch in src/utils/api.ts) must get a 401 JSON. A 302 here
        //     would be transparently followed by fetch() and return the login HTML as if
        //     it were the API payload — breaking the client. This is the bug to avoid.
        //   - Top-level browser navigations get a 302 to /login so the user sees the page.
        if (isApiRequest(request, url)) {
          return Response.json({ error: { code: 'unauthorized', message: 'Unauthorized' } }, { status: 401 });
        }
        return Response.redirect(`${url.origin}/login?next=${encodeURIComponent(url.pathname)}`, 302);
      }

      // Forward the signed access_token; the backend verifies it and derives identity.
      const resp = await forwardToBackend(request, env, tenantConfig, sessionResult.token!);

      // If we refreshed, attach the rotated cookies so the browser picks up the new tokens.
      return refreshed ? withRefreshedCookies(resp, refreshed, hostname) : resp;
    }

    // 3. Handle login route
    if (url.pathname === '/api/auth/login') {
      return handleLogin(request, env, tenantConfig, hostname);
    }

    // 4. Forward to backend
    return forwardToBackend(request, env, tenantConfig, null);
  }
};

// ─── Tenant Resolution ───────────────────────────────────────────────────────

interface TenantConfig {
  tenant_id: string;
  tenant_name: string;
  plan: string;
  has_custom_domain: boolean;   // false = owner domain, uses redirect SSO
  backend_url: string;
  PUBLIC_TENANT_API_KEY: string;
  branding: { primary_color: string; logo_url: string };
}

// ─── Login Handler ────────────────────────────────────────────────────────────

async function handleLogin(
  request: Request,
  env: Env,
  tenant: TenantConfig,
  hostname: string
): Promise<Response> {
  // If the tenant does NOT have a custom domain → redirect to the owner SSO
  if (!tenant.has_custom_domain) {
    const ssoUrl = new URL('https://sso.portalonline.id/login');
    ssoUrl.searchParams.set('tenant_id', tenant.tenant_id);
    ssoUrl.searchParams.set('redirect', `https://${hostname}/auth/callback`);
    return Response.redirect(ssoUrl.toString(), 302);
  }

  // If the tenant has a custom domain → proxy to SSO (the user is unaware)
  const body = await request.json() as { email: string; password: string };

  const ssoResponse = await fetch('https://sso.portalonline.id/api/internal/token', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Internal-Secret': env.INTERNAL_SSO_SECRET,  // secret dedicated to Worker → SSO
      'X-Tenant-ID': tenant.tenant_id,
    },
    body: JSON.stringify(body),
  });

  if (!ssoResponse.ok) {
    const err = await ssoResponse.json();
    return Response.json(err, { status: ssoResponse.status });
  }

  const { access_token, refresh_token, user } = await ssoResponse.json() as TokenResponse;

  // Store session in KV (edge session store). TTL = access lifetime (15m); the edge
  // entry is what makes a token revocable (delete it → next request fails at the edge).
  const sessionHash = await hashToken(access_token);
  await env.CACHE_SSO.put(
    `session:${sessionHash}`,
    JSON.stringify({ user_id: user.id, tenant_id: tenant.tenant_id, role: user.role, permissions: user.permissions }),
    { expirationTtl: 900 } // 15 minutes — matches ACCESS_TTL
  );

  // Set httpOnly cookies on the tenant domain.
  const response = Response.json({ success: true, user: { name: user.name, role: user.role } });
  // __session = short-lived access token (15m).
  response.headers.set('Set-Cookie',
    `__session=${access_token}; HttpOnly; Secure; SameSite=Strict; Path=/; Domain=${hostname}; Max-Age=900`
  );
  // __refresh = sliding refresh token. Path=/ (NOT /api/auth/refresh) so the Worker
  // receives it on EVERY request and can silently refresh an expired access token.
  // Max-Age = idle window (12h); each rotation resets it. Absolute cap (24h) is enforced
  // server-side from the auth_time claim, independent of the cookie.
  response.headers.append('Set-Cookie',
    `__refresh=${refresh_token}; HttpOnly; Secure; SameSite=Strict; Path=/; Domain=${hostname}; Max-Age=43200`
  );

  return response;
}

// ─── Session Validation ───────────────────────────────────────────────────────

async function validateSession(
  request: Request,
  env: Env,
  tenant: TenantConfig
): Promise<{ valid: boolean; user?: SessionUser; token?: string }> {
  const cookie = getCookie(request, '__session');
  if (!cookie) return { valid: false };

  const sessionHash = await hashToken(cookie);
  const sessionData = await env.CACHE_SSO.get(`session:${sessionHash}`, 'json') as SessionUser | null;

  // Edge owns session lifecycle: a missing/expired KV entry = revoked or timed out.
  // (The backend will still independently verify the JWT signature — defense in depth.)
  if (!sessionData) return { valid: false };

  // Ensure the session belongs to the correct tenant
  if (sessionData.tenant_id !== tenant.tenant_id) return { valid: false };

  // Return the cookie value (the access_token JWT) so it can be forwarded to the backend.
  return { valid: true, user: sessionData, token: cookie };
}

// ─── Silent Refresh (Worker-side, invisible to the Vue app) ────────────────────

interface RefreshResult {
  accessToken: string;
  refreshToken: string;
}

// Called when the access session is gone but a __refresh cookie is present.
// Forwards the refresh token (+ UA & real IP for the fingerprint check) to the backend.
// On success: re-seed the edge access-session in CACHE_SSO and return the rotated tokens.
async function trySilentRefresh(
  request: Request,
  env: Env,
  tenant: TenantConfig
): Promise<RefreshResult | null> {
  const refresh = getCookie(request, '__refresh');
  if (!refresh) return null;

  const resp = await fetch(`${tenant.backend_url}/api/auth/refresh`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${refresh}`,
      'X-Tenant-ID': tenant.tenant_id,
      // Forward the original context so the backend fingerprint matches (UA + coarse IP).
      'User-Agent': request.headers.get('User-Agent') ?? '',
      'CF-Connecting-IP': request.headers.get('CF-Connecting-IP') ?? '',
    },
  });

  if (!resp.ok) return null; // expired / reused / context changed → caller redirects to login

  const { access_token, refresh_token, role, permissions } =
    await resp.json() as { access_token: string; refresh_token: string; role: string; permissions: string[] };

  // Refresh the edge access-session (15m) keyed by the new token's hash.
  const sessionHash = await hashToken(access_token);
  await env.CACHE_SSO.put(
    `session:${sessionHash}`,
    JSON.stringify({ tenant_id: tenant.tenant_id, role, permissions }),
    { expirationTtl: 900 }
  );

  return { accessToken: access_token, refreshToken: refresh_token };
}

// Attach rotated cookies to the proxied response so the browser stores the new tokens.
function withRefreshedCookies(resp: Response, r: RefreshResult, hostname: string): Response {
  const out = new Response(resp.body, resp); // clone (headers mutable)
  out.headers.append('Set-Cookie',
    `__session=${r.accessToken}; HttpOnly; Secure; SameSite=Strict; Path=/; Domain=${hostname}; Max-Age=900`
  );
  out.headers.append('Set-Cookie',
    `__refresh=${r.refreshToken}; HttpOnly; Secure; SameSite=Strict; Path=/; Domain=${hostname}; Max-Age=43200`
  );
  return out;
}

// ─── Forward to Go Backend ────────────────────────────────────────────────────

async function forwardToBackend(
  request: Request,
  env: Env,
  tenant: TenantConfig,
  token: string | null   // the access_token JWT, or null for public routes
): Promise<Response> {
  const url = new URL(request.url);
  const backendUrl = `${tenant.backend_url}${url.pathname}${url.search}`;

  const headers = new Headers(request.headers);
  // X-Tenant-ID is a resolution hint only; the backend cross-checks it against the JWT claim.
  headers.set('X-Tenant-ID', tenant.tenant_id);
  headers.set('X-Tenant-API-Key', tenant.PUBLIC_TENANT_API_KEY);

  // Identity is carried by the signed JWT, NOT by trusted X-User-* headers.
  // The backend verifies the signature and derives user_id/role/permissions from claims.
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  // Defensively strip any X-User-* the client may have tried to spoof — the backend
  // ignores them anyway, but don't let them propagate.
  headers.delete('X-User-ID');
  headers.delete('X-User-Role');
  headers.delete('X-User-Permissions');

  // Remove cookie before forwarding (the backend uses the Bearer token, not the cookie)
  headers.delete('Cookie');

  // Propagate ONE request id from the edge so logs correlate edge→backend (observability.md).
  // Reuse Cloudflare's CF-Ray if present, else mint a UUID. The backend trusts this header
  // (only the Worker reaches the origin) and echoes it back; its Recover 500 quotes the same id.
  if (!headers.has('X-Request-ID')) {
    headers.set('X-Request-ID', request.headers.get('CF-Ray') ?? crypto.randomUUID());
  }

  return fetch(backendUrl, {
    method: request.method,
    headers,
    body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : undefined,
  });
}

// ─── SSO Callback (for tenants without a custom domain) ─────────────────────────

async function handleSSOCallback(request: Request, env: Env, tenant: TenantConfig): Promise<Response> {
  const url = new URL(request.url);
  const token = url.searchParams.get('token');
  if (!token) return new Response('Invalid callback', { status: 400 });

  // Verify the token from the owner SSO
  const verifyResponse = await fetch('https://sso.portalonline.id/api/internal/verify', {
    headers: {
      'Authorization': `Bearer ${token}`,
      'X-Internal-Secret': env.INTERNAL_SSO_SECRET,
    }
  });

  if (!verifyResponse.ok) return Response.redirect('/login?error=invalid_token');

  const { user, access_token, refresh_token } = await verifyResponse.json();
  // Set cookie and session the same way as the proxy flow
  // ...
}

// ─── Utils ────────────────────────────────────────────────────────────────────

function getCookie(request: Request, name: string): string | null {
  const cookieHeader = request.headers.get('Cookie') || '';
  const cookies = Object.fromEntries(cookieHeader.split('; ').map(c => c.split('=')));
  return cookies[name] || null;
}

async function hashToken(token: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(token);
  const hash = await crypto.subtle.digest('SHA-256', data);
  return Array.from(new Uint8Array(hash)).map(b => b.toString(16).padStart(2, '0')).join('');
}

function isPublicRoute(pathname: string): boolean {
  // /api/webhooks/payment/* are gateway callbacks (Midtrans/Duitku) — they arrive with NO
  // session/JWT, so they bypass the auth proxy. They are authenticated by SIGNATURE at the
  // backend (billing.md), NOT by the network — the opposite of /api/internal/* (which is
  // hidden). Do not confuse the two: webhooks are publicly reachable on purpose.
  const publicRoutes = ['/login', '/auth/callback', '/health', '/api/webhooks/payment/'];
  return publicRoutes.some(r => pathname.startsWith(r));
}

// Distinguish API/XHR calls (want 401 JSON) from browser navigations (want 302 redirect).
// An API call is either under /api/* or explicitly asks for JSON / is a fetch (not a document nav).
function isApiRequest(request: Request, url: URL): boolean {
  if (url.pathname.startsWith('/api/')) return true;
  const accept = request.headers.get('Accept') ?? '';
  if (accept.includes('application/json')) return true;
  // Fetch sets Sec-Fetch-Mode: cors|same-origin; top-level navigations use "navigate".
  const mode = request.headers.get('Sec-Fetch-Mode');
  return mode !== null && mode !== 'navigate';
}
```

---

## Backend Go: SSO Internal Endpoint

```go
import (
    "crypto/subtle"
    "time"

    "github.com/google/uuid"
    "github.com/yourapp/internal/httperr" // envelope error helpers (backend-golang.md)
)

// RefreshClaims — signed payload of a refresh token. Everything here is tamper-proof
// (part of the JWT), so absolute cap + fingerprint need no extra storage.
type RefreshClaims struct {
    UserID   string `json:"sub"`
    TenantID string `json:"tenant_id"`
    FamilyID string `json:"family_id"` // groups all rotations of one login
    JTI      string `json:"jti"`       // this token's id; must match Redis current_jti
    AuthTime int64  `json:"auth_time"` // original login (unix); absolute cap measured from here
    FP       string `json:"fp"`        // context fingerprint (UA + coarse IP)
    jwt.RegisteredClaims
}

// validInternalSecret — constant-time compare against current + previous secret.
// Dual-key window lets you rotate INTERNAL_SSO_SECRET with zero downtime: set the new
// value as Current, keep the old as Previous, deploy Worker with the new value, then
// drop Previous on the next release. NEVER log the secret or the incoming header.
func (h *SSOHandler) validInternalSecret(got string) bool {
    if got == "" {
        return false
    }
    g := []byte(got)
    for _, want := range [][]byte{h.config.InternalSecretCurrent, h.config.InternalSecretPrevious} {
        if len(want) > 0 && subtle.ConstantTimeCompare(g, want) == 1 {
            return true
        }
    }
    return false
}

// Internal endpoint — see deployment.md: served on a separate internal listener
// reachable only via Cloudflare Tunnel, NOT on the public Nginx server block.
func (h *SSOHandler) InternalToken(w http.ResponseWriter, r *http.Request) {
    // Validate the Worker secret (constant-time; do not log the value)
    if !h.validInternalSecret(r.Header.Get("X-Internal-Secret")) {
        httperr.Write(w, httperr.Forbidden())
        return
    }

    tenantID := r.Header.Get("X-Tenant-ID")

    // Validate input at the boundary (see backend-golang.md: decodeAndValidate).
    // LoginInput: Email `required,email,max=255`, Password `required,min=8,max=200`.
    var in LoginInput
    if appErr := decodeAndValidate(r, &in); appErr != nil {
        httperr.Write(w, appErr) // 422 with per-field messages, or 400 on malformed JSON
        return
    }

    // Login = boundary case: no JWT yet, so the tenant is taken from X-Tenant-ID
    // (set by the Worker) then placed into context. The auth query still goes through store.InTenant
    // → RLS automatically restricts the user lookup to this tenant (cannot auth a user from another tenant).
    ctx := tenant.WithTenant(r.Context(), &tenant.Tenant{ID: tenantID})

    user, err := h.userSvc.AuthenticateInTenant(ctx, in.Email, in.Password)
    if err != nil {
        // Generic message regardless of "no such user" vs "wrong password" — don't reveal
        // which emails exist in this tenant (account enumeration).
        httperr.Write(w, &httperr.AppError{Status: 401, Code: "invalid_credentials", Message: "Invalid email or password"})
        return
    }

    // Get permissions from the role (also through store.InTenant) — don't swallow the error.
    permissions, err := h.roleSvc.GetPermissions(ctx, user.Role)
    if err != nil {
        httperr.Write(w, &httperr.AppError{Status: 500, Code: "internal_error", Message: "Could not load permissions"})
        return
    }

    // Access token = short (15m). It is the bearer credential the backend verifies per request.
    accessToken, _ := h.jwt.GenerateTenantToken(user.ID, tenantID, user.Role, permissions, 15*time.Minute)

    // Refresh token = a new family. It carries (signed, tamper-proof):
    //   family_id  — groups all rotations of this login; reuse detection key
    //   jti        — this token's unique id; must equal Redis current_jti to be valid
    //   auth_time  — original login time; absolute cap (24h) is measured from here
    //   fp         — context fingerprint (UA + coarse IP) for light device binding
    fp := fingerprint(r) // see /api/auth/refresh below — same helper
    familyID := uuid.NewString()
    jti := uuid.NewString()
    refreshToken, _ := h.jwt.GenerateRefreshToken(RefreshClaims{
        UserID: user.ID, TenantID: tenantID,
        FamilyID: familyID, JTI: jti, AuthTime: time.Now().Unix(), FP: fp,
    }, 12*time.Hour) // idle window; absolute cap enforced separately from auth_time

    // Seed the refresh family in Redis: family_id → current_jti. This is the ONE piece of
    // refresh-session state that lives in Redis (atomic CAS for reuse detection — KV cannot
    // do this). Edge access-sessions stay in CACHE_SSO; see backend-golang.md.
    if err := h.refreshStore.SeedFamily(ctx, familyID, jti, 12*time.Hour); err != nil {
        httperr.Write(w, &httperr.AppError{Status: 500, Code: "internal_error", Message: "Could not start session"})
        return
    }

    json.NewEncoder(w).Encode(map[string]any{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "user": map[string]any{
            "id":          user.ID,
            "name":        user.Name,
            "role":        user.Role,
            "permissions": permissions,
        },
    })
}
```

> **Hardening `INTERNAL_SSO_SECRET` — all four are required:**
> 1. **Constant-time compare** (`subtle.ConstantTimeCompare`), never `==`. A plain string compare leaks the secret byte-by-byte via timing.
> 2. **Dual-key rotation** (`Current` + `Previous`): rotate with zero downtime — promote new → Current, keep old → Previous, redeploy the Worker, then drop Previous next release.
> 3. **Never log it.** Keep `X-Internal-Secret` out of Nginx access logs and app logs (it is a bearer credential). A leaked secret = anyone can forge `X-Tenant-ID` and mint a token for *any* tenant.
> 4. **Network isolation.** `/api/internal/*` must NOT be reachable from the public origin — serve it on a separate internal listener reached only via Cloudflare Tunnel. See `deployment.md`. The secret is the auth; the network isolation is defense-in-depth so a leaked secret alone is not exploitable.

---

## Backend Go: Refresh Endpoint (`/api/auth/refresh`)

Sliding refresh with **rotation + reuse detection**. Each refresh mints a new access token (15m) and a new refresh token in the same family; the old refresh token is invalidated. If an already-rotated refresh token is presented again, that means it was stolen → the whole family is revoked.

```go
import (
    "crypto/sha256"
    "encoding/hex"
    "net"
    "time"
)

// fingerprint — light context binding: UA + coarse IP (/24 v4, /48 v6).
// Coarse so a normal IP change within an ISP block does not force re-auth,
// but a token replayed from a different network/device is rejected.
func fingerprint(r *http.Request) string {
    ip := r.Header.Get("CF-Connecting-IP") // real client IP injected by Cloudflare
    coarse := coarsenIP(ip)
    sum := sha256.Sum256([]byte(r.UserAgent() + "|" + coarse))
    return hex.EncodeToString(sum[:])
}

func (h *SSOHandler) Refresh(w http.ResponseWriter, r *http.Request) {
    raw := bearerToken(r) // Worker forwards __refresh as Authorization: Bearer on this route
    claims, err := h.jwt.VerifyRefresh(raw) // signature + exp (the 12h idle window)
    if err != nil {
        httperr.Write(w, &httperr.AppError{Status: 401, Code: "invalid_refresh", Message: "Session is no longer valid"})
        return
    }

    // Absolute cap (24h) measured from the ORIGINAL login, carried in auth_time.
    // Sliding refresh cannot extend a session past this — forces periodic re-login.
    if time.Since(time.Unix(claims.AuthTime, 0)) > 24*time.Hour {
        httperr.Write(w, &httperr.AppError{Status: 401, Code: "session_expired", Message: "Please sign in again"})
        return
    }

    // Light device binding: fingerprint must match the one minted at login.
    if claims.FP != fingerprint(r) {
        httperr.Write(w, &httperr.AppError{Status: 401, Code: "context_changed", Message: "Please sign in again"})
        return
    }

    ctx := tenant.WithTenant(r.Context(), &tenant.Tenant{ID: claims.TenantID})

    // Atomic rotate: only succeeds if claims.JTI == current_jti for this family.
    // If it does NOT match → this is a reused (rotated-away) token → REVOKE the family.
    newJTI := uuid.NewString()
    ok, err := h.refreshStore.RotateFamily(ctx, claims.FamilyID, claims.JTI, newJTI, 12*time.Hour)
    if err != nil {
        httperr.Write(w, &httperr.AppError{Status: 500, Code: "internal_error", Message: "Could not refresh session"})
        return
    }
    if !ok {
        // Reuse detected (or family already revoked). Kill the whole family.
        _ = h.refreshStore.RevokeFamily(ctx, claims.FamilyID)
        httperr.Write(w, &httperr.AppError{Status: 401, Code: "refresh_reused", Message: "Please sign in again"})
        return
    }

    // Re-read role/permissions so a permission change takes effect on refresh (≤15m lag).
    permissions, err := h.roleSvc.GetPermissionsByUser(ctx, claims.UserID)
    if err != nil {
        httperr.Write(w, &httperr.AppError{Status: 500, Code: "internal_error", Message: "Could not refresh session"})
        return
    }
    role, err := h.roleSvc.GetUserRole(ctx, claims.UserID)
    if err != nil {
        httperr.Write(w, &httperr.AppError{Status: 500, Code: "internal_error", Message: "Could not refresh session"})
        return
    }

    accessToken, _ := h.jwt.GenerateTenantToken(claims.UserID, claims.TenantID, role, permissions, 15*time.Minute)
    refreshToken, _ := h.jwt.GenerateRefreshToken(RefreshClaims{
        UserID: claims.UserID, TenantID: claims.TenantID,
        FamilyID: claims.FamilyID,   // same family
        JTI:      newJTI,            // rotated id
        AuthTime: claims.AuthTime,   // preserve original login time → absolute cap unchanged
        FP:       claims.FP,
    }, 12*time.Hour)

    json.NewEncoder(w).Encode(map[string]any{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "role":          role,
        "permissions":   permissions,
    })
}
```

> **Why `refresh_family` lives in Redis, not KV.** Reuse detection needs an *atomic* compare-and-set on `current_jti` (see `backend-golang.md` → `RotateFamily`). Cloudflare KV is eventually consistent — two concurrent refreshes could both "win", defeating detection. So the refresh family's rotation state is the one piece of session data kept in Redis. This refines the earlier rule "Redis NOT session": **edge access-session = KV** (per-request revocation), **refresh-rotation state = Redis** (atomic reuse detection). They are different concerns.

---

## Rate Limiting Login

```typescript
// In the CF Worker, before forwarding the login request
async function checkRateLimit(env: Env, ip: string, tenantId: string): Promise<boolean> {
  const key = `ratelimit:${tenantId}:${ip}`;
  const current = await env.CACHE_SSO.get(key);
  const count = current ? parseInt(current) : 0;

  if (count >= 5) return false; // max 5 attempts per 15 minutes

  await env.CACHE_SSO.put(key, String(count + 1), { expirationTtl: 900 });
  return true;
}
```

---

## Auth Flow Diagram

```
TENANT WITH CUSTOM DOMAIN (Proxy):
User → manage.kabarsiang.id/api/auth/login (POST email+password)
  → CF Worker: check has_custom_domain = true
  → POST to sso.portalonline.id/api/internal/token + X-Internal-Secret
  → Get token → store in CACHE_SSO
  → Set httpOnly cookie on manage.kabarsiang.id
  → Return { success: true } ← the user is unaware SSO is involved

TENANT WITHOUT CUSTOM DOMAIN (Redirect):
User → manage.portalonline.id/api/auth/login
  → CF Worker: check has_custom_domain = false
  → Redirect to sso.portalonline.id/login?tenant_id=X&redirect=...
  → User logs in at sso.portalonline.id
  → Redirect back to manage.portalonline.id/auth/callback?token=X
  → Worker verifies the token, sets cookie
  → Redirect to /dashboard
```