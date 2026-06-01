# frontend-owner

Admin dashboard for PortalOnline multi-tenant SaaS platform. Built with Vue 3 + TypeScript + Pinia + Tailwind CSS.

## Quick Start

```bash
# Install dependencies
npm install

# Start dev server (requires backend at VITE_API_BASE_URL or localhost:8080)
npm run dev

# Build for production
npm run build

# Run tests
npm test
```

## Environment Variables

Copy `.env.example` to `.env` and adjust:

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | Backend API base URL |

## Architecture

```
src/
├── app/            # App root (App.vue, env.ts, styles.css)
├── components/     # Reusable UI components (common/ + navigation/)
├── contracts/      # API type contracts — single source of truth
├── layouts/        # AppAdminLayout, AuthLayout
├── pages/          # Route-level page components (auth/, app/, errors/)
├── router/         # Vue Router config with auth guards + lazy loading
├── services/       # API clients (axios instances + service wrappers)
├── stores/         # Pinia stores (auth, ui)
└── utils/          # Formatting helpers
```

### API Contract

`src/contracts/api.ts` defines every API type the dashboard consumes. The backend returns `SuccessEnvelope<T>` (success) or `ErrorEnvelope` (failure). The `ErrorCode` union drives UI error handling (redirects for `password_change_required`, toast messages, etc.).

### Auth Flow

1. `useAuthStore` manages access/refresh tokens and user profile
2. `services/api.ts` creates Axios instances with interceptors for token attachment, 401 refresh, and 403 password-change redirect
3. `router/index.ts` guards routes with `requiresAuth`, `scope`, and `permission` metadata
4. RBAC is pure functions in `services/rbac.ts` — no side effects, testable without DOM

### RBAC Roles

| Role | Permissions |
|------|-------------|
| `owner-app` | All permissions |
| `admin` | All except `app.tenants.delete` |
| `super_admin` | All |
| `platform_admin` | All except `app.tenants.delete` |
| `support_agent` | Read tenants + users |
| `billing_admin` | Read tenants |
| `auditor` | Read tenants + audit |

## Testing

```bash
# Run all tests
npm test

# Watch mode
npm run test:watch
```

Tests use [Vitest](https://vitest.dev/) + [@vue/test-utils](https://test-utils.vuejs.org/) + [jsdom](https://github.com/jsdom/jsdom).

## Production Readiness Checklist

- [ ] Integrate Sentry (register `app.config.errorHandler` in `main.ts`)
- [ ] Add Web Vitals collection (LCP, CLS, INP)
- [ ] Add analytics (Plausible or PostHog)
- [ ] Configure CSP headers on the reverse proxy
- [ ] Protect `/metrics` endpoint behind auth or IP allowlist
- [ ] Deploy HSTS at the reverse-proxy layer
