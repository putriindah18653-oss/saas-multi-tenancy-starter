# UI Public — Server-Rendered Tenant Sites (SEO-Critical)

The **public** UI (`vue-public`): anonymous, read-heavy, and **SEO-critical** — each tenant's site is served on its own domain (`kabarsiang.id`, `theme.portalonline.id`) and must be indexable. This is a different world from the dashboard (`ui-dashboard.md`): no auth, no RBAC, no forms — the job is rendering tenant content fast and crawlable.

The **shared visual foundation** (Tailwind tokens + tenant branding via CSS variables) lives in `frontend-vue-cloudflare.md`; this file consumes those tokens. What differs from the dashboard is the **rendering model** and **how branding/SEO reach the first byte**.

---

## Rendering Model: SSR / Pre-render, NOT SPA

> **The public worker must server-render. An SPA shell is the wrong choice here.** The dashboard is an SPA because it sits behind login (crawlers never see it, a flash of empty shell is harmless). The public site is the opposite: search engines and social scrapers need real HTML — title, meta, and content — in the **first response**, not after a JS bundle boots. An SPA-only public site indexes as a blank page.

So the PUBLIC worker does **not** use `not_found_handling: single-page-application`. It renders HTML on the edge per request (cached — see Performance below), then the client hydrates.

```typescript
// server/index.ts (PUBLIC worker) — SSR, not asset-SPA passthrough
import { renderToString } from 'vue/server-renderer'
import { createSSRApp } from 'vue'
import { resolveTenant } from './tenant'
import { renderShell } from './shell'
import { handleAPI } from './api' // two-layer public cache, see frontend-vue-cloudflare.md

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url)

    // API + static assets handled as before.
    if (url.pathname.startsWith('/api/')) return handleAPI(request, env, ctx)
    if (isAsset(url.pathname)) return env.ASSETS.fetch(request)

    const tenant = await resolveTenant(env, url.hostname)
    if (!tenant) return new Response('Not found', { status: 404 })

    // 1. Fetch the page's content (goes through the cached public API).
    const page = await fetchPageData(env, tenant, url.pathname) // { content, seo }
    if (!page) return renderNotFound(tenant)

    // 2. Server-render the Vue app to HTML string.
    const app = createSSRApp(PublicApp, { tenant, page })
    const appHtml = await renderToString(app)

    // 3. Assemble the full document: SEO head + branding + rendered app + hydration data.
    const html = renderShell({ tenant, page, appHtml })
    return new Response(html, { headers: { 'Content-Type': 'text/html; charset=utf-8' } })
  },
}
```

> **Pre-render vs per-request SSR.** For mostly-static content (articles, marketing pages), per-request SSR + the two-layer edge cache is the pragmatic default — the first request renders, the rest are cache hits. True build-time pre-render (SSG) is an option only when content rarely changes; with per-tenant content that updates via the dashboard, edge-cached SSR is the better fit.

---

## SEO Essentials — Injected at Render, per Tenant

Everything a crawler reads must be in the server-rendered `<head>`, built from tenant + page data. The canonical URL is the **tenant's own domain**, not the platform's.

```typescript
// server/shell.ts — assemble the HTML document around the rendered app.
export function renderShell({ tenant, page, appHtml }: ShellInput): string {
  const canonical = `https://${tenant.primary_domain}${page.path}`
  return `<!DOCTYPE html>
<html lang="${tenant.lang ?? 'en'}">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${esc(page.seo.title)} — ${esc(tenant.name)}</title>
  <meta name="description" content="${esc(page.seo.description)}" />
  <link rel="canonical" href="${esc(canonical)}" />
  <link rel="icon" href="${esc(tenant.branding.favicon_url)}" />

  <!-- Open Graph / social -->
  <meta property="og:title" content="${esc(page.seo.title)}" />
  <meta property="og:description" content="${esc(page.seo.description)}" />
  <meta property="og:url" content="${esc(canonical)}" />
  <meta property="og:image" content="${esc(page.seo.ogImage ?? tenant.branding.logo_url)}" />
  <meta property="og:type" content="${page.seo.type ?? 'website'}" />

  <!-- Structured data (JSON-LD) — helps rich results; build from page content -->
  <script type="application/ld+json">${jsonLd(tenant, page)}</script>

  ${brandingStyle(tenant)}  <!-- critical branding + theme CSS, see below -->
  ${themeBootScript()}   <!-- sets html.dark before first paint -->
</head>
<body>
  <div id="app">${appHtml}</div>
  <!-- Hydration payload: the client reuses this instead of refetching -->
  <script>window.__TENANT__=${safeJson(tenant)};window.__PAGE__=${safeJson(page)}</script>
  <script type="module" src="/assets/entry-client.js"></script>
</body>
</html>`
}

// esc() HTML-escapes text; safeJson() escapes </script. NEVER interpolate unescaped
// tenant/user content into HTML — that's a stored-XSS vector on the public site.
```

> **Escaping is a security boundary, not a nicety.** Public pages render tenant-authored content (article titles, bodies). Any value placed into HTML or a `<script>` must be escaped (`esc`) / JSON-safe-encoded (`safeJson`) — an unescaped title is stored XSS served on the tenant's own domain.

```typescript
// Example JSON-LD for an article page.
function jsonLd(tenant: TenantConfig, page: PageData): string {
  return safeJson({
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: page.seo.title,
    datePublished: page.content.publishedAt,
    author: { '@type': 'Organization', name: tenant.name },
    publisher: { '@type': 'Organization', name: tenant.name, logo: tenant.branding.logo_url },
  })
}
```

---

## Tenant Branding + Dark / Light Theme at SSR — No FOUC

> **Public branding and theme must be in the first byte.** The public site is SSR and SEO-critical; a crawler snapshot or slow connection must not see unbranded or wrong-theme colors. So tenant brand variables and light/dark semantic tokens are inlined into the server-rendered `<head>`, and a tiny boot script sets `html.dark` before first paint.

```typescript
// server/shell.ts — inline the tenant's brand tokens and semantic light/dark tokens.
// Same CSS variables the Tailwind config maps to (frontend-vue-cloudflare.md), just
// delivered at SSR instead of set by JS.
function brandingStyle(tenant: TenantConfig): string {
  const b = tenant.branding
  return `<style>
:root{
  color-scheme: light;
  --color-primary:${cssColor(b.primary_color)};
  --color-primary-hover:${cssColor(b.primary_hover ?? b.primary_color)};
  --color-primary-light:${cssColor(b.primary_light ?? '#eff6ff')};
  --surface:255 255 255;
  --surface-secondary:248 250 252;
  --surface-border:226 232 240;
  --text:15 23 42;
  --text-muted:100 116 139;
  --text-inverse:255 255 255;
}
html.dark{
  color-scheme: dark;
  --surface:15 23 42;
  --surface-secondary:30 41 59;
  --surface-border:51 65 85;
  --text:248 250 252;
  --text-muted:148 163 184;
  --text-inverse:15 23 42;
}
</style>`
}
// cssColor() validates the value is a safe color (hex/rgb) before inlining — never
// pass raw tenant input into a <style> block.
```

```typescript
// server/shell.ts — no Vue/DOM dependency; runs before CSS paint.
function themeBootScript(): string {
  return `<script>(function(){try{var k='public-theme';var m=localStorage.getItem(k)||'auto';var d=m==='dark'||(m!=='light'&&matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.classList.toggle('dark',d);document.documentElement.dataset.theme=m;}catch(e){}})();</script>`
}
```

The client hydrates with the same `--color-primary` and theme class already set, so there is no recompute and no flash. Components use the same Tailwind `primary`/`surface`/`text` tokens as the dashboard.

For public hydrated islands that need a theme toggle, use VueUse on the client only:

```typescript
// src/composables/useTheme.ts
import { useColorMode } from '@vueuse/core'

export function useTheme() {
  const mode = useColorMode({
    selector: 'html',
    attribute: 'class',
    storageKey: 'public-theme',
    initialValue: 'auto',
    modes: {
      light: '',
      dark: 'dark',
      auto: 'auto',
    },
  })

  return { mode }
}
```

Keep theme choice visitor-local. Do not include visitor theme in the public HTML cache key unless the server varies HTML by theme. With this pattern, cached SSR HTML is shared safely; the tiny boot script applies each visitor's theme before first paint.

---

## Content Components (distinct from dashboard)

The public site renders **content**, not controls. These components are read-only and SSR-safe (no browser-only APIs at render time). They live alongside, not inside, the dashboard's `base/` set.

```
src/components/
  public/
    ArticleView.vue        ← single article: title, byline, body
    ArticleList.vue        ← paginated/feed list of article cards
    ContentRenderer.vue    ← renders tenant rich-content safely (sanitized HTML/blocks)
    SiteHeader.vue         ← tenant logo + nav from tenant config
    SiteFooter.vue
```

```vue
<!-- src/components/public/ContentRenderer.vue -->
<!-- Renders tenant-authored rich content. Sanitize on the way in (or trust a backend
     that already sanitized) — this is the same XSS boundary as the SSR shell. -->
<template>
  <article class="prose max-w-none text-text">
    <!-- If content is structured blocks, render per type; if HTML, it MUST be sanitized. -->
    <component
      v-for="(block, i) in blocks"
      :key="i"
      :is="blockComponent(block.type)"
      :data="block.data"
    />
  </article>
</template>

<script setup lang="ts">
defineProps<{ blocks: ContentBlock[] }>()
// blockComponent maps a block type → a small presentational component (paragraph,
// heading, image, embed). Avoid v-html; if unavoidable, sanitize server-side first.
</script>
```

> No `BaseInput`/`BaseModal`/form components here — those are dashboard-only (`ui-dashboard.md`). If a public page needs interaction (search, comments), keep it a small hydrated island, not a full SPA conversion.

---

## Performance — Tie-in to the Two-Layer Cache

Public SSR pages are the prime cache target. The two-layer public cache (`frontend-vue-cloudflare.md`) caches the **rendered HTML** for unauthenticated GETs, tenant-scoped and version-keyed:

- First request to a colo renders SSR → stored in Cache API (per-colo) + KV (global shield).
- Subsequent requests are edge hits — no render, no backend call.
- Content change in the dashboard → `PurgeTenantContent` bumps `cachever:{tenant_id}` → stale HTML becomes unreachable, next request re-renders.

> Public responses are cacheable precisely because they carry **no `Authorization`** and are tenant-scoped by key. Never cache a personalized public response under the shared key — if a page ever varies per visitor, exclude it from the cache (it's the same rule as the dashboard: only cache `/api/public/*` with no auth).

---

## What This Doc Does NOT Cover

- **Auth / sessions / RBAC** — the public site is anonymous. If a tenant later adds member-only public content, that login flow belongs in `auth-sso.md`, and gated content must be excluded from the public cache.
- **Forms, tables, modals, toasts** — dashboard concerns, see `ui-dashboard.md`.
- **Tailwind config + branding token definitions** — shared, see `frontend-vue-cloudflare.md`.
