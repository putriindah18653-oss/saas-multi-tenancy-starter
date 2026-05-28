# Skill: Dashboard UI/UX — Agent Specification

> Hard rules for AI agents when building UI/UX in this project.
> This document is the **single source of truth** for all frontend decisions.

---

## 1. Absolute Prohibitions

| # | Rule | Reason |
|---|------|--------|
| 1 | **NO AI SLOOP** — never repeat a failed approach, never patch incrementally without diagnosing root cause | Wastes time, produces garbage code |
| 2 | **NO** third-party UI frameworks (Nuxt UI, PrimeVue, Vuetify, Headless UI, Radix Vue, shadcn-vue) | Full control, small bundle |
| 3 | **NO** adding dependencies without a narrow, documented justification | Bloat |
| 4 | **NO** storing server-state in Pinia — use `@tanstack/vue-query` | Single source of truth for cache |
| 5 | **NO** inline styles or CSS modules — Tailwind utility classes only | Consistency |
| 6 | **NO** creating components without a TypeScript props interface | Type safety |
| 7 | **NO** fetching data without loading state, error state, and empty state | Minimum UX |

---

## 2. Stack & Versions

| Layer | Technology | Notes |
|-------|------------|-------|
| Framework | Vue 3 (Composition API, `<script setup lang="ts">`) | SFC required |
| Build | Vite 5+ | — |
| Language | TypeScript strict | `noEmit` via `vue-tsc` |
| Styling | **Tailwind CSS 4** | Upgrade from current v3 |
| Font | **Quicksand** (self-hosted, not CDN) | Install in `src/assets/fonts/` |
| Router | Vue Router 4 | Route guards for auth & RBAC |
| Server State | **@tanstack/vue-query** | Fetch, cache, invalidation, pagination |
| Client State | Pinia | Only for: auth snapshot, theme, sidebar toggle, tenant selection |
| HTTP Client | Axios (already present) | Keep, wrap in service layer |
| Chart | Third-party allowed (e.g. Chart.js, Apache ECharts) | Charts only |
| Date Picker | Third-party allowed (e.g. VCalendar) | Date picker only |
| Combobox/Select | Third-party allowed if search + virtualization needed | Complex combobox only |
| Virtualized Table | Third-party allowed (e.g. @tanstack/vue-virtual) | 1000+ row tables only |

---

## 3. Folder Structure (Target)

Both apps (`frontend-owner` and `frontend-tenant`) MUST follow this structure:

```
src/
├── app/
│   ├── App.vue
│   ├── env.ts
│   └── styles.css              # Tailwind v4 entry (@import "tailwindcss")
├── assets/
│   └── fonts/
│       ├── Quicksand-Variable.woff2
│       └── fonts.css           # @font-face declarations
├── components/
│   ├── ui/                     # Reusable, domain-agnostic
│   │   ├── UiButton.vue
│   │   ├── UiInput.vue
│   │   ├── UiPasswordInput.vue # Input + eye toggle
│   │   ├── UiCard.vue
│   │   ├── UiModal.vue
│   │   ├── UiDataTable.vue     # Table + pagination + empty/loading
│   │   ├── UiSearchFilter.vue  # Search bar + filter chips
│   │   ├── UiPagination.vue
│   │   ├── UiBadge.vue
│   │   ├── UiAlert.vue
│   │   ├── UiSpinner.vue
│   │   └── UiDropdown.vue
│   └── navigation/             # App shell navigation
│       ├── AppSidebar.vue
│       ├── TopNav.vue
│       └── LogoutButton.vue
├── composables/                # Shared hooks
│   ├── useDebounce.ts
│   └── usePagination.ts
├── features/                   # Domain-specific
│   ├── auth/
│   │   └── components/
│   ├── tenants/
│   │   └── components/
│   ├── users/
│   │   └── components/
│   ├── audit/
│   │   └── components/
│   └── settings/
│       └── components/
├── layouts/
│   ├── AuthLayout.vue
│   └── AppAdminLayout.vue      # or TenantLayout.vue
├── pages/                      # Route-level views
├── router/
│   └── index.ts
├── services/                   # Axios wrappers (thin, typed)
├── stores/                     # Pinia (client state only)
├── types/                      # Shared TypeScript types
│   ├── api.ts                  # API response generics
│   └── models.ts               # Domain models
└── main.ts
```

---

## 4. Naming Conventions

| Entity | Convention | Example |
|--------|-----------|---------|
| UI component | `Ui` prefix + PascalCase | `UiButton.vue`, `UiDataTable.vue` |
| Domain component | Domain prefix + PascalCase | `TenantCard.vue`, `AuditLogTable.vue` |
| Composable | `use` prefix + camelCase | `useDebounce.ts`, `usePagination.ts` |
| Service | camelCase + `Service` suffix | `tenantsService`, `auditService` |
| Store | `use` prefix + `Store` suffix | `useAuthStore`, `useThemeStore` |
| Type/Interface | PascalCase | `Tenant`, `AuditEntry`, `PaginatedResponse<T>` |
| File | kebab-case for non-components | `api.ts`, `env.ts` |
| CSS class | Tailwind utility only | — |

---

## 5. UI Components — Required Specifications

### 5.1 UiPasswordInput

- Input type toggles between `password` and `text`
- Eye/eye-off icon as a button inside the input
- Props: `modelValue`, `placeholder`, `autocomplete`, `disabled`, `id`
- Emit: `update:modelValue`

### 5.2 UiDataTable

- Props: `columns`, `rows`, `loading`, `emptyText`
- Slot: `#cell-{columnKey}` for custom per-column rendering
- MUST show skeleton/spinner when loading
- MUST show empty state when rows is empty
- Integrates with `UiPagination`

### 5.3 UiSearchFilter (required above every complex table)

- Search input with debounce (300ms default)
- Slot for additional filters (dropdown, date range)
- Emit: `update:search`, `update:filters`

### 5.4 UiPagination

- Props: `page`, `pageSize`, `total`
- Emit: `update:page`
- Display "Showing X–Y of Z" info
- N+1 guard: fetch `pageSize + 1`, display `pageSize`, use row N+1 as `hasNext` indicator

---

## 6. Data Fetching Pattern

### MUST use @tanstack/vue-query

```typescript
// src/services/tenants.ts — thin axios wrapper
import { ownerApi } from '@/services/api'
import type { PaginatedResponse, Tenant } from '@/types'

export const tenantsService = {
  list: (params: { page: number; pageSize: number; search?: string }) =>
    ownerApi.get<PaginatedResponse<Tenant>>('/app/tenants', { params }),
  get: (id: string) => ownerApi.get<{ data: Tenant }>(`/app/tenants/${id}`),
}
```

```typescript
// In page/component — vue-query
import { useQuery } from '@tanstack/vue-query'
import { tenantsService } from '@/services/tenants'

const page = ref(1)
const search = ref('')

const { data, isLoading, isError, error } = useQuery({
  queryKey: ['tenants', page, search],
  queryFn: () => tenantsService.list({ page: page.value, pageSize: 20, search: search.value }),
  select: (res) => res.data,
})
```

### Pinia is ONLY for:
- `useAuthStore` — accessToken, refreshToken, user snapshot
- `useThemeStore` — dark/light, sidebar collapsed
- `useTenantStore` — selectedTenantId, memberships (tenant app only)

---

## 7. Tailwind CSS 4 — Setup

### styles.css (replaces @tailwind directives)

```css
@import "tailwindcss";
@import "../assets/fonts/fonts.css";

@theme {
  --font-sans: "Quicksand", system-ui, sans-serif;
  --color-primary-50: #eff6ff;
  --color-primary-100: #dbeafe;
  --color-primary-500: #3b82f6;
  --color-primary-600: #2563eb;
  --color-primary-700: #1d4ed8;
  --color-primary-900: #1e3a5f;
}
```

### No longer needed:
- `tailwind.config.ts` (remove)
- `postcss.config.js` (remove if Tailwind v4 standalone)

### Self-hosted font:

```css
/* src/assets/fonts/fonts.css */
@font-face {
  font-family: "Quicksand";
  src: url("./Quicksand-Variable.woff2") format("woff2-variations");
  font-weight: 300 700;
  font-display: swap;
}
```

---

## 8. Page Pattern

Every page displaying a data list MUST follow this pattern:

```vue
<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold text-slate-900">{{ title }}</h1>
      <UiButton v-if="canCreate" @click="...">Create</UiButton>
    </div>

    <!-- Search & Filter — REQUIRED above table -->
    <UiSearchFilter v-model:search="search" @update:filters="..." />

    <!-- Alert -->
    <UiAlert v-if="isError" variant="error">{{ error }}</UiAlert>

    <!-- Data Table -->
    <UiDataTable :columns="columns" :rows="rows" :loading="isLoading" empty-text="No data found.">
      <template #cell-action="{ row }">...</template>
    </UiDataTable>

    <!-- Pagination -->
    <UiPagination v-model:page="page" :page-size="pageSize" :total="total" />
  </div>
</template>
```

---

## 9. API Contract

### Base URL
- Development: `http://localhost:8080/api/v1`
- Env var: `VITE_API_BASE_URL`

### Headers
- `Authorization: Bearer <token>`
- `X-Tenant-ID: <uuid>` (tenant-scoped requests)
- `Content-Type: application/json`

### Response envelope
```typescript
// Success
{ "data": T }
{ "data": T[], "meta": { "total": number, "page": number, "page_size": number } }

// Error
{ "error": { "code": string, "message": string } }
```

### Available endpoints

| Method | Path | Scope | Permission |
|--------|------|-------|------------|
| POST | /auth/login | public | — |
| POST | /auth/refresh | public | — |
| POST | /auth/logout | auth | — |
| GET | /me | auth | — |
| POST | /me/password | auth | — |
| GET | /me/sessions | auth | — |
| DELETE | /me/sessions/{id} | auth | — |
| GET | /me/tenants | auth | — |
| GET | /me/permissions | tenant | — |
| GET | /app/tenants | owner | app.tenants.read |
| POST | /app/tenants | owner | app.tenants.create |
| GET | /app/tenants/{id} | owner | app.tenants.read |
| PATCH | /app/tenants/{id} | owner | app.tenants.update |
| DELETE | /app/tenants/{id} | owner | app.tenants.delete |
| GET | /app/audit | owner | app.audit.read |
| GET | /tenant/dashboard | tenant | tenant.dashboard.read |
| GET | /tenant/me | tenant | — |
| GET | /tenant/users | tenant | tenant.users.read |
| POST | /tenant/users/invite | tenant | tenant.users.invite |
| PATCH | /tenant/users/{id}/role | tenant | tenant.users.update |
| DELETE | /tenant/users/{id} | tenant | tenant.users.remove |
| GET | /tenant/settings | tenant | tenant.settings.read |
| PATCH | /tenant/settings | tenant | tenant.settings.update |
| GET | /tenant/audit | tenant | tenant.audit.read |

---

## 10. TypeScript Types (Shared)

```typescript
// src/types/api.ts
export interface ApiResponse<T> {
  data: T
}

export interface PaginatedResponse<T> {
  data: T[]
  meta: { total: number; page: number; page_size: number }
}

export interface ApiError {
  error: { code: string; message: string }
}

// src/types/models.ts
export interface Tenant {
  id: string
  name: string
  slug: string
  status: string
  created_at: string
}

export interface TenantMember {
  id: string
  name: string
  email: string
  role: string
  is_active: boolean
}

export interface AuditEntry {
  id: string
  action: string
  tenant_id?: string
  resource_type: string
  resource_id?: string
  actor_user_id?: string
  created_at: string
}

export interface TenantSettings {
  display_name: string
  logo_url: string
  timezone: string
  locale: string
  currency: string
}

export interface UserSession {
  id: string
  ip_address: string
  user_agent: string
  created_at: string
  last_used_at: string
}
```

---

## 11. Cross-Platform Consistency

When the same component exists in both `frontend-owner` and `frontend-tenant`:
- **File name** must be identical
- **Props interface** must be identical
- **Behavior** must be identical
- **Visual treatment** must be identical

Examples: `UiDataTable.vue`, `UiButton.vue`, `UiPasswordInput.vue` — identical in both apps.

> If a shared package is needed later, extract to `packages/ui`. For now, keep identical copies.

---

## 12. Migration Checklist (from current state)

Execution order when the agent starts refactoring:

1. **Install dependencies** — `@tanstack/vue-query`, download Quicksand font
2. **Upgrade Tailwind v3 → v4** — remove `tailwind.config.ts`, `postcss.config.js`, update `styles.css`
3. **Setup font** — `src/assets/fonts/` + `fonts.css` + `@theme` config
4. **Create `src/components/ui/`** — start with `UiButton`, `UiInput`, `UiPasswordInput`, `UiCard`, `UiSpinner`
5. **Create `UiDataTable` + `UiPagination` + `UiSearchFilter`**
6. **Create `src/types/`** — `api.ts`, `models.ts`
7. **Refactor services** — add typed responses, pagination params
8. **Refactor pages** — replace manual fetch with `useQuery`, replace old components with `Ui*`
9. **Create `src/features/`** — move domain components
10. **Verify** — `vue-tsc --noEmit` must pass, `vite build` must succeed

---

## 13. Accessibility Minimum

- All `<button>` must have text content or `aria-label`
- All `<input>` must have an associated `<label>` or `aria-label`
- Focus must be visible (Tailwind `focus-visible:ring-2`)
- Tables must use `<thead>`, `<th scope="col">`
- Modals must trap focus and close on Escape
- Color contrast minimum WCAG AA (4.5:1 for normal text)

---

## 14. Performance Rules

- Lazy load route-level pages (`() => import(...)`)
- Images use `loading="lazy"`
- Never import entire icon libraries — import per-icon
- `@tanstack/vue-query` staleTime minimum 30 seconds for list data
- Debounce search input (300ms)
- Do not re-fetch on window refocus for rarely-changing data (`refetchOnWindowFocus: false`)

---

## 15. Security Rules (Frontend)

- Never store tokens in localStorage — use sessionStorage or memory (already correct)
- Never display passwords/tokens in console.log
- Sanitize user input before rendering if using `v-html` (avoid `v-html`)
- RBAC check in route guard AND in template (defense in depth)
- Temporary password displayed once only, then dismiss

---

## 16. Git & Workflow

- Each feature/refactor = 1 branch, 1 PR
- Commit message: `feat(dashboard): add UiDataTable component`
- Never commit `node_modules`, `.env`, or build output (`dist/`)
- Before commit: `vue-tsc --noEmit` + `vite build` must pass
