# Frontend Production Hardening

This document adds production quality gates for the Vue frontends without changing their architecture.

Canonical architecture remains in `frontend-vue-cloudflare.md`:

- `vue-dashboard`: authenticated Vue SPA behind the Dashboard Worker.
- `vue-public`: SEO-critical server-rendered tenant site behind the Public Worker.
- Shared Tailwind tokens and tenant branding variables are defined in `frontend-vue-cloudflare.md`.
- Dashboard components live in `ui-dashboard.md`.
- Public SSR/content components live in `ui-public.md`.

Do not use this document to convert the public site into an SPA or to introduce React/Next.js patterns. It is Vue + Cloudflare Workers only.

---

## Non-Contradiction Rules

```text
[ ] Dashboard remains SPA; no SEO requirement behind login
[ ] Public site remains SSR/pre-render; not SPA-only
[ ] No third-party UI component library
[ ] No React/Next.js/theme patterns
[ ] Tailwind semantic tokens stay canonical in frontend-vue-cloudflare.md
[ ] Base dashboard components stay canonical in ui-dashboard.md
[ ] Public content/SEO/SSR shell stay canonical in ui-public.md
[ ] Public cache rules stay canonical in frontend-vue-cloudflare.md
```

This file covers production concerns: bundle size, performance budgets, accessibility gates, CSP compatibility, error handling UX, hydration safety, and release checks.

---

## Build and Type Gates

Every frontend release must run:

```bash
# dashboard
cd vue-dashboard
npm ci
npm run typecheck
npm run lint
npm run test:unit
npm run build

# public
cd ../vue-public
npm ci
npm run typecheck
npm run lint
npm run test:unit
npm run build
```

Recommended package scripts:

```json
{
  "scripts": {
    "typecheck": "vue-tsc --noEmit",
    "lint": "eslint .",
    "test:unit": "vitest run",
    "build": "vite build",
    "preview": "vite preview"
  }
}
```

Lockfiles are required. Do not deploy with uncommitted dependency changes.

---

## Bundle Budgets

Dashboard and public have different budgets.

Recommended starting budgets:

```text
Dashboard initial JS:     <= 250 KB gzip
Dashboard route chunk:    <= 150 KB gzip
Public hydration JS:      <= 120 KB gzip
Public critical CSS:      <= 30 KB gzip
Public SSR HTML TTFB:     target < 500ms at p95 including cache misses
```

Use bundle analyzer:

```bash
npm install -D rollup-plugin-visualizer
```

Vite config:

```ts
import { visualizer } from 'rollup-plugin-visualizer'

export default defineConfig({
  plugins: [
    vue(),
    visualizer({ filename: 'dist/stats.html', gzipSize: true, brotliSize: true })
  ]
})
```

Rules:

- Route-split dashboard modules.
- Lazy-load heavy editors/charts.
- Keep public site hydration minimal.
- Do not ship dashboard-only dependencies to public SSR bundle.

---

## Dashboard Production UX

Dashboard SPA must have consistent states:

```text
loading
empty
error
permission denied
session expired
offline/retry
```

Rules:

- Components use base components from `ui-dashboard.md`.
- API errors use `ApiError` from `src/utils/api.ts`.
- `422 validation_failed` maps to field errors.
- `403 forbidden` shows permission messaging.
- `404` on tenant-scoped resource shows not found, not cross-tenant detail.
- `429` shows retry message using `Retry-After` if available.
- Session expired sends user to login only after Worker silent refresh fails.

Add global error boundary:

```vue
<!-- App.vue -->
<template>
  <RouterView v-slot="{ Component }">
    <ErrorBoundary>
      <component :is="Component" />
    </ErrorBoundary>
  </RouterView>
</template>
```

Vue error handler:

```ts
app.config.errorHandler = (err, instance, info) => {
  console.error('vue_error', { err, info })
  // Send sanitized error event to logging provider if configured.
}
```

Do not log tokens, cookies, raw API bodies, or PII.

---

## Public SSR Production UX

Public site must optimize first response:

```text
server-render title/meta/canonical/content
inline tenant branding CSS variables
inline dark/light boot script with CSP nonce if needed
hydrate only interactive islands where possible
cache anonymous GET HTML only
```

Rules:

- Do not use `not_found_handling: single-page-application` for public Worker.
- Unknown tenant domain returns SSR/HTML 404 or safe API envelope for `/api/*`.
- Tenant-authored content must be escaped/sanitized before rendering.
- Hydration mismatch warnings are release blockers.
- Public pages must work without JS for basic content reading.

Hydration safety checklist:

```text
[ ] no Date.now()/Math.random() directly in SSR-rendered template
[ ] no locale/timezone mismatch in first render
[ ] tenant branding variables identical server/client
[ ] sanitized HTML identical server/client
[ ] client-only widgets wrapped in client-only guard
```

---

## CSP Compatibility

Security headers are defined in `security-hardening.md`; frontend must be compatible with them.

Dashboard:

- Avoid inline scripts.
- Avoid arbitrary `v-html`.
- Use static asset imports.
- Keep `connect-src` limited to API/Worker origins.

Public:

- If using inline theme boot script from `ui-public.md`, use nonce-based CSP.
- Do not add broad `unsafe-inline` in production unless there is a documented exception.
- Sanitize any rich content rendered via `v-html` or avoid `v-html` entirely with structured blocks.

Example nonce usage:

```html
<script nonce="{{nonce}}">/* themeBootScript() */</script>
```

CSP header:

```text
script-src 'self' 'nonce-{{nonce}}'
```

---

## Accessibility Gates

Minimum:

```text
[ ] keyboard navigation for dashboard forms/tables/modals/dropdowns
[ ] visible focus states
[ ] modal focus trap and restore
[ ] aria labels for icon-only buttons
[ ] form errors linked with aria-describedby
[ ] color contrast passes WCAG AA
[ ] public headings are semantic
[ ] public nav/footer landmarks exist
[ ] reduced-motion respected for animations
```

Use tests:

```bash
npm install -D @axe-core/playwright
npx playwright test accessibility.spec.ts
```

---

## SEO and Social Validation for Public Site

Public SSR pages must include:

```text
title
meta description
canonical URL
Open Graph tags
Twitter card tags if needed
JSON-LD where relevant
robots directives
sitemap/robots strategy
```

Per-tenant domain rules:

- Canonical URL uses tenant public domain.
- Do not canonical every tenant to platform domain unless intentionally white-label-disabled.
- Tenant branding and content must be in initial HTML.
- 404 pages return actual 404 status.

Validation:

```bash
curl -s https://tenant.example.com/article/test | grep '<title>'
curl -I https://tenant.example.com/not-found
```

---

## Performance and Lighthouse CI

Recommended CI check for public pages:

```bash
npm install -D @lhci/cli
npx lhci autorun
```

Starting budgets:

```text
Performance:       >= 85
Accessibility:     >= 95
Best Practices:    >= 95
SEO:               >= 95
```

Do not treat Lighthouse as the only source of truth; also monitor real latency/cache metrics in `metrics-alerting.md`.

---

## Asset Caching

Vite hashed assets:

```text
/assets/app.[hash].js
/assets/app.[hash].css
```

Headers:

```text
Cache-Control: public, max-age=31536000, immutable
```

HTML/SSR:

```text
Cache-Control: public, max-age=60, stale-while-revalidate=300
```

Only for anonymous public pages. Dashboard HTML/app shell may be cached as static asset if it contains no tenant/user data; authenticated API responses are never public-cached.

---

## Environment Safety

Only expose `VITE_*` variables that are safe for browsers.

Allowed:

```text
VITE_APP_ENV
VITE_BACKEND_URL
VITE_PUBLIC_SITE_BASE
```

Forbidden:

```text
JWT_SECRET
INTERNAL_SSO_SECRET
MAIL_API_KEY
PAYMENT_SERVER_KEY
DATABASE_URL
REDIS_URL
S3_SECRET_KEY
```

Tests should scan built assets for known secret names/prefixes.

---

## Error Reporting

If using Sentry or similar:

```text
[ ] scrub request headers
[ ] scrub cookies
[ ] scrub Authorization
[ ] scrub tenant PII where required
[ ] include release version/git SHA
[ ] include request_id when available
[ ] source maps uploaded privately, not publicly exposed
```

Public error pages should be safe and not leak stack traces.

---

## Browser Support

Recommended:

```text
latest 2 Chrome/Edge/Firefox/Safari
current iOS Safari
current Android Chrome
```

Do not over-polyfill public site. If enterprise customers require older browsers, document it and adjust build targets.

---

## Frontend Release Checklist

```text
[ ] Vue typecheck/lint/unit/build pass
[ ] Bundle budgets reviewed
[ ] Dashboard states tested: loading/empty/error/403/404/429/session expired
[ ] Public SSR HTML contains title/meta/content before JS
[ ] No public SPA fallback config
[ ] Hydration warnings absent
[ ] CSP works without unsafe production exceptions
[ ] Accessibility smoke passes
[ ] Lighthouse/public performance budget checked
[ ] Built assets scanned for secret names
[ ] Cache headers verified
[ ] Sentry/error reporting scrubbed if enabled
```

---

## Definition of Done

```text
[ ] Does not contradict frontend-vue-cloudflare.md, ui-dashboard.md, or ui-public.md
[ ] Dashboard remains SPA and uses base components
[ ] Public remains SSR/pre-render and SEO-valid
[ ] Shared tokens/branding are reused, not redefined
[ ] Bundle/performance/accessibility budgets pass
[ ] CSP and secret exposure checks pass
[ ] Public cache/auth boundaries remain intact
```
