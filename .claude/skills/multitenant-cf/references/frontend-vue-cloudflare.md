# Frontend Vue + Cloudflare Workers

## Architecture: 2 Workers, Many Domains

```
CF Worker PUBLIC  (vue-public)
  Domains: kabarsiang.id, mediahaji.com, theme.portalonline.id, ...
  KV: CACHE_KV (tenant config + article cache)
  → Backend Go /api/public/*

CF Worker DASHBOARD (vue-dashboard)
  Domains: manage.kabarsiang.id, manage.mediahaji.com, manage.portalonline.id, ...
  KV: CACHE_KV (tenant config), CACHE_SSO (edge access-session, 15m; refresh-family lives in Redis)
  → Backend Go /api/dashboard/*
```

---

## Project Setup

### Initialization (Cloudflare Vite Plugin)
```bash
# Public frontend
npm create cloudflare@latest vue-public -- --framework=vue
cd vue-public && npm install

# Dashboard frontend
npm create cloudflare@latest vue-dashboard -- --framework=vue
cd vue-dashboard && npm install
```

### `wrangler.jsonc` (Dashboard Worker)
```jsonc
{
  "name": "vue-dashboard",
  "main": "server/index.ts",
  "compatibility_date": "2025-01-01",
  "assets": {
    "directory": "./dist",
    "not_found_handling": "single-page-application"
  },
  "kv_namespaces": [
    { "binding": "CACHE_KV",  "id": "xxx" },
    { "binding": "CACHE_SSO", "id": "yyy" }
  ],
  "vars": {
    "BACKEND_URL": "https://api.yourdomain.com",
    "ENVIRONMENT": "production"
  },
  "secrets": ["INTERNAL_SSO_SECRET"]
}
```

> **Two workers, two rendering models — their `wrangler.jsonc` differ.** The DASHBOARD config above uses `not_found_handling: single-page-application` because the dashboard is an authenticated SPA (no SEO; crawlers never see it). The **PUBLIC worker must NOT be SPA-only** — its tenant sites must be crawlable, so it server-renders HTML on the edge (no `single-page-application` fallback; `main` does the SSR). The full public SSR worker, SEO head injection, and content components are in `ui-public.md`; the dashboard component system is in `ui-dashboard.md`.

### `vite.config.ts`
```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { cloudflare } from '@cloudflare/vite-plugin'
import path from 'path'

export default defineConfig({
  plugins: [vue(), cloudflare()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') }
  }
})
```

---

## Worker Entry Point

### `server/index.ts` (PUBLIC Worker — skeleton)
```typescript
import { Env } from './types'
import { resolveTenant } from './tenant'
import { handleAPI } from './api'
import { renderPage } from './ssr' // SSR entry — full version in ui-public.md

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url)

    // API routes → proxy to backend Go (ctx enables non-blocking cache writes)
    if (url.pathname.startsWith('/api/')) {
      return handleAPI(request, env, ctx)
    }

    // Build artifacts (JS/CSS/images) → serve from the Assets binding.
    if (isAsset(url.pathname)) {
      return env.ASSETS.fetch(request)
    }

    // Everything else → SERVER-RENDER the page (SEO-critical). NOT an SPA fallback.
    // renderPage resolves the tenant, fetches content via the cached API, and returns
    // crawlable HTML with SEO head + branding inlined. See ui-public.md.
    return renderPage(request, env, ctx)
  }
}
```

> The DASHBOARD worker keeps the simple SPA shape instead — its `fetch` proxies `/api/*` (with auth/silent-refresh, see `auth-sso.md`) and otherwise serves the SPA via `env.ASSETS.fetch` with `not_found_handling: single-page-application`. Public SSRs; dashboard doesn't.

### `server/tenant.ts`
```typescript
export interface TenantConfig {
  tenant_id: string
  tenant_name: string
  plan: string
  has_custom_domain: boolean
  PUBLIC_TENANT_API_KEY: string
  branding: {
    primary_color: string
    logo_url: string
    favicon_url: string
  }
  features: string[]  // ["comments", "newsletter", "analytics"]
}

export async function resolveTenant(env: Env, hostname: string): Promise<TenantConfig | null> {
  // Cached in Worker memory (per request, does not persist across requests)
  const cached = await env.CACHE_KV.get<TenantConfig>(`tenant:${hostname}`, 'json')
  return cached
}
```

### `server/api.ts` — Proxy to Backend Go (two-layer public cache)

> **Caching is a cross-tenant leak risk — three non-negotiable rules:**
> 1. **Only cache unauthenticated public content** — `/api/public/*` API responses (this section) and SSR HTML pages (`ui-public.md`), both with **no `Authorization` header**. Never cache an authenticated response — it would serve one user's data to another.
> 2. **`tenant_id` is in every cache key.** A shared cache without tenant scoping serves tenant A's content on tenant B's domain.
> 3. **Reads only.** Never cache non-GET, and bypass cache entirely when a request is authenticated.

The same two-layer mechanism below is reused by the public SSR worker to cache rendered HTML — only the value differs (JSON vs HTML); the layering, version key, and tenant scoping are identical.

Two layers, fastest first: **Cache API** (`caches.default`, per-colo, microseconds) → **KV** (global, origin shield) → backend. The Cache API key embeds a **content version** read from KV; an active purge bumps that version so old edge entries become *unreachable* globally without needing a global flush (Workers Cache API cannot be purged across colos — see purge helper).

```typescript
function apiError(code: string, message: string, status: number): Response {
  return Response.json({ error: { code, message } }, { status })
}

export async function handleAPI(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
  const url = new URL(request.url)
  const tenant = await resolveTenant(env, url.hostname)
  if (!tenant) return apiError('tenant_not_found', 'Tenant not found', 404)

  const backendUrl = `${env.BACKEND_URL}${url.pathname}${url.search}`
  const headers = new Headers(request.headers)
  headers.set('X-Tenant-ID', tenant.tenant_id)
  headers.set('X-Tenant-API-Key', tenant.PUBLIC_TENANT_API_KEY)
  headers.delete('Cookie') // never forward the browser cookie to the public backend
  // One request id from the edge so logs correlate edge→backend (observability.md).
  if (!headers.has('X-Request-ID')) {
    headers.set('X-Request-ID', request.headers.get('CF-Ray') ?? crypto.randomUUID())
  }

  // Cacheable ONLY when: GET + /api/public/* + NOT authenticated.
  const cacheable =
    request.method === 'GET' &&
    url.pathname.startsWith('/api/public/') &&
    !request.headers.has('Authorization')

  if (!cacheable) {
    // Authenticated or mutating → straight through, no cache touched.
    // (Payment webhook callbacks POST /api/webhooks/payment/* land here too: passed straight
    //  to the backend, never cached, no auth added — they self-authenticate by signature, billing.md.)
    return fetch(backendUrl, {
      method: request.method,
      headers,
      body: request.method !== 'GET' ? request.body : undefined,
    })
  }

  // Content version (per tenant). Bumped on purge → old Cache API keys go unreachable.
  const version = (await env.CACHE_KV.get(`cachever:${tenant.tenant_id}`)) ?? '0'
  // Synthetic, version-stamped key. tenant_id scopes it; version busts it.
  const cacheKey = new Request(
    `https://edge-cache.internal/${tenant.tenant_id}/v${version}${url.pathname}${url.search}`
  )
  const edge = caches.default

  // Layer 1 — Cache API (per-colo).
  const hitEdge = await edge.match(cacheKey)
  if (hitEdge) {
    const r = new Response(hitEdge.body, hitEdge)
    r.headers.set('X-Cache', 'HIT-EDGE')
    return r
  }

  // Layer 2 — KV (global origin shield). Body stored under a tenant-scoped, path key.
  const kvBodyKey = `cachebody:${tenant.tenant_id}:${url.pathname}${url.search}`
  const hitKV = await env.CACHE_KV.get(kvBodyKey)
  if (hitKV) {
    const r = new Response(hitKV, { headers: { 'Content-Type': 'application/json', 'X-Cache': 'HIT-KV' } })
    // Backfill the colo-local Cache API so subsequent hits skip KV.
    ctx.waitUntil(edge.put(cacheKey, cloneForCache(r, 60)))
    return r
  }

  // Layer 3 — backend.
  const resp = await fetch(backendUrl, { method: 'GET', headers })
  const body = await resp.text()
  if (resp.ok) {
    // Write both layers. KV TTL = origin-shield window; Cache API TTL = per-colo backstop.
    ctx.waitUntil(env.CACHE_KV.put(kvBodyKey, body, { expirationTtl: 300 }))
    const cached = new Response(body, { headers: { 'Content-Type': 'application/json' } })
    ctx.waitUntil(edge.put(cacheKey, cloneForCache(cached, 60)))
  }
  return new Response(body, {
    status: resp.status,
    headers: { 'Content-Type': 'application/json', 'X-Cache': 'MISS' },
  })
}

// Cache API honors Cache-Control for its own TTL; set it on the stored copy.
function cloneForCache(r: Response, ttlSeconds: number): Response {
  const c = new Response(r.body, r)
  c.headers.set('Cache-Control', `public, max-age=${ttlSeconds}`)
  return c
}
```

---

## Vue: Tenant-Aware Setup

### `src/composables/useTenant.ts`
```typescript
import { ref, readonly } from 'vue'
import { api } from '@/utils/api' // shared wrapper: envelope parsing + 401 handling

interface TenantBranding {
  primaryColor: string
  logoUrl: string
  faviconUrl: string
}

interface TenantInfo {
  id: string
  name: string
  plan: string
  features: string[]
  branding: TenantBranding
}

// Tenant info injected by the Worker via meta tag or first API call
const tenant = ref<TenantInfo | null>(null)

export function useTenant() {
  const hasFeature = (feature: string) => tenant.value?.features.includes(feature) ?? false

  const applyBranding = () => {
    if (!tenant.value?.branding) return
    const { primaryColor } = tenant.value.branding
    document.documentElement.style.setProperty('--color-primary', primaryColor)
  }

  const init = async () => {
    // Fetch tenant info via the shared wrapper so the envelope + 401 handling apply.
    // On failure, leave tenant null and rethrow — main.ts decides how to surface it.
    tenant.value = await api.get<TenantInfo>('/tenant/me')
    applyBranding()
  }

  return {
    tenant: readonly(tenant),
    hasFeature,
    applyBranding,
    init,
  }
}
```

### `src/main.ts`
```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useTenant } from './composables/useTenant'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Init tenant before mount
const { init } = useTenant()
init().then(() => app.mount('#app'))
```

### `src/utils/api.ts` — Fetch Wrapper
```typescript
const BASE_URL = '/api'

// Mirror of the backend envelope (backend-golang.md): { error: { code, message, fields? } }.
export class ApiError extends Error {
  constructor(
    public code: string,
    message: string,
    public fields?: Record<string, string>, // per-field validation messages, if any
    public status?: number,
  ) {
    super(message)
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'same-origin',  // send the httpOnly cookies; the Worker reads them
  })

  if (response.status === 401) {
    // NO client-side refresh here. Silent refresh is the Worker's job (see auth-sso.md):
    // on every request it transparently renews an expired access token from __refresh.
    // A 401 reaching this code means refresh ALSO failed → the session genuinely ended
    // (idle past the 12h window, hit the 24h absolute cap, or was revoked). Re-login.
    window.location.href = '/login'
    throw new ApiError('unauthorized', 'Unauthorized', undefined, 401)
  }

  if (!response.ok) {
    // Parse the envelope. NOTE: the error fields live under `error`, NOT at the top level —
    // reading `body.message` (the old bug) always missed them and showed "Request failed".
    const body = await response.json().catch(() => null)
    const e = body?.error
    throw new ApiError(
      e?.code ?? 'error',
      e?.message ?? 'Request failed',
      e?.fields,
      response.status,
    )
  }

  return response.json()
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) => request<T>(path, {
    method: 'POST', body: JSON.stringify(body)
  }),
  put: <T>(path: string, body: unknown) => request<T>(path, {
    method: 'PUT', body: JSON.stringify(body)
  }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}
```

> **Mapping field errors to a form.** On `422 validation_failed`, `ApiError.fields` is `{ email: "...", password: "..." }` — wire each entry to the matching `BaseInput`'s `error` prop (`ui-dashboard.md`):
> ```typescript
> try {
>   await api.post('/auth/login', form)
> } catch (e) {
>   if (e instanceof ApiError && e.fields) {
>     fieldErrors.value = e.fields            // bind to <BaseInput :error="fieldErrors.email" />
>   } else if (e instanceof ApiError) {
>     toast.error(e.message)                  // non-field error → toast
>   }
> }
> ```

---

## Multi-Domain Setup in Cloudflare

### Adding a Custom Domain to a Worker (via Wrangler)
```bash
# Add domain to Worker
wrangler domains add kabarsiang.id --env production
wrangler domains add manage.kabarsiang.id --env production

# Or via wrangler.jsonc
{
  "routes": [
    { "pattern": "kabarsiang.id/*", "custom_domain": true },
    { "pattern": "manage.kabarsiang.id/*", "custom_domain": true }
  ]
}
```

### Script: Register New Tenant Domain (from Backend Go)
```go
// When a new tenant is added, call the CF API to add the domain to the Worker
func (s *CFService) AddTenantDomain(ctx context.Context, workerName, domain string) error {
    // Cloudflare API: add custom domain to Worker
    url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/domains", s.accountID)

    payload := map[string]string{
        "hostname":    domain,
        "service":     workerName,   // "vue-dashboard" or "vue-public"
        "environment": "production",
    }

    // HTTP PUT to CF API with CF API Token
    // ...
}

// Update KV when tenant config changes
func (s *CFService) SetTenantKV(ctx context.Context, domain string, config TenantConfig) error {
    url := fmt.Sprintf(
        "https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/tenant:%s",
        s.accountID, s.kvNamespaceID, domain,
    )
    // HTTP PUT with JSON body
}

// PurgeTenantContent — active purge when a tenant's public content changes.
// Workers Cache API (caches.default) CANNOT be purged globally — cache.delete() only
// affects the colo that runs it. So we make stale entries UNREACHABLE instead of deleting:
//   1. Bump cachever:{tenant_id} in KV → every Cache API key embeds the version, so the
//      next request computes a new key and the old per-colo entries are never matched again.
//   2. Delete the KV body keys (cachebody:{tenant_id}:*) → the global origin-shield layer
//      reflects fresh content immediately.
//   3. (Optional) CF zone purge-by-URL for any content also cached at the CDN edge.
// Result: purge feels instant globally, without relying on a non-existent global Cache API flush.
func (s *CFService) PurgeTenantContent(ctx context.Context, tenantID string, paths []string) error {
    // 1. Bump the version counter (read-modify-write or atomic INCR-style via your KV wrapper).
    newVer := s.bumpCacheVersion(ctx, tenantID) // writes cachever:{tenantID}
    _ = newVer

    // 2. Delete KV body keys for the changed paths.
    for _, p := range paths {
        key := fmt.Sprintf("cachebody:%s:%s", tenantID, p)
        s.deleteKV(ctx, key)
    }

    // 3. Best-effort CDN purge-by-URL (if these paths are also edge-cached at the zone level).
    // POST https://api.cloudflare.com/client/v4/zones/{zone}/purge_cache  {"files":[...]}
    return nil
}
```

---

## Tailwind Config — Tenant Branding via CSS Variables

### `tailwind.config.js`
```javascript
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        // Tenant branding via CSS variables (set by useTenant.applyBranding)
        primary: {
          DEFAULT: 'var(--color-primary, #3b82f6)',
          hover:   'var(--color-primary-hover, #2563eb)',
          light:   'var(--color-primary-light, #eff6ff)',
        },
        // Semantic UI colors; light/dark values come from CSS variables
        // defined in ui-dashboard.md and inlined for public SSR in ui-public.md.
        surface: {
          DEFAULT: 'rgb(var(--surface) / <alpha-value>)',
          secondary: 'rgb(var(--surface-secondary) / <alpha-value>)',
          border: 'rgb(var(--surface-border) / <alpha-value>)',
        },
        text: {
          DEFAULT: 'rgb(var(--text) / <alpha-value>)',
          muted: 'rgb(var(--text-muted) / <alpha-value>)',
          inverse: 'rgb(var(--text-inverse) / <alpha-value>)',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      }
    }
  }
}
```

### Usage in a Component
```vue
<template>
  <!-- primary automatically uses the tenant color via CSS var -->
  <button class="bg-primary text-text-inverse hover:bg-primary-hover px-4 py-2 rounded-lg">
    Save
  </button>
</template>
```