# Plan Lengkap: Frontend Tenant Dashboard UI Refactor

## Status eksekusi

- Commit UI refactor sudah dibuat sebelum plan ini ditulis.
- Commit: `06c48a6` (`refactor(tenant): polish dashboard ui`)
- Push: `origin/main` berhasil.

## Latar belakang

`frontend-tenant` adalah dashboard tenant PortalOnline berbasis Vue 3, Vite, TypeScript, Tailwind, Pinia, dan Vue Router. Sebelum refactor, UI tenant sudah fungsional tetapi masih campur raw `slate-*` classes, layout belum seragam dengan dashboard owner, dan beberapa page belum punya loading, empty, dan alert state yang konsisten.

Plan ini mendokumentasikan refactor tenant dashboard agar UI lebih konsisten, token-first, responsive, dan mudah dirawat tanpa mengubah kontrak API, route, auth, atau dependency.

## Scope

### In scope

1. Tenant design token di global stylesheet.
2. Reusable dashboard primitives:
   - `AppCard`
   - `AppButton`
   - `KpiCard`
   - `UiAlert`
   - `UiEmptyState`
3. Tenant shell/navigation:
   - `TopNav`
   - `AppSidebar`
   - `TenantLayout`
4. Tenant pages:
   - `TenantDashboard`
   - `TenantUsersPage`
   - `UserInviteForm`
   - `TenantSettingsPage`
   - `TenantAuditPage`
5. Loading skeleton, empty state, success/error alert, and permission-aware quick actions.
6. Typecheck and production build verification.

### Out of scope

1. No backend change.
2. No API contract change.
3. No auth/token storage change.
4. No route behavior change.
5. No new dependencies.
6. No broad folder architecture change.
7. No changes to pre-existing unrelated modified files.

## Files touched

```text
frontend-tenant/src/app/styles.css
frontend-tenant/src/components/common/AppButton.vue
frontend-tenant/src/components/common/AppCard.vue
frontend-tenant/src/components/common/KpiCard.vue
frontend-tenant/src/components/common/UiAlert.vue
frontend-tenant/src/components/common/UiEmptyState.vue
frontend-tenant/src/components/navigation/AppSidebar.vue
frontend-tenant/src/components/navigation/TopNav.vue
frontend-tenant/src/components/tenant-users/UserInviteForm.vue
frontend-tenant/src/layouts/TenantLayout.vue
frontend-tenant/src/pages/tenant/TenantAuditPage.vue
frontend-tenant/src/pages/tenant/TenantDashboard.vue
frontend-tenant/src/pages/tenant/TenantSettingsPage.vue
frontend-tenant/src/pages/tenant/users/TenantUsersPage.vue
```

## Implementation detail

### 1. Global tenant tokens

Add token layer in `frontend-tenant/src/app/styles.css`:

- `--tenant-bg-app`
- `--tenant-bg-sidebar`
- `--tenant-surface`
- `--tenant-surface-elevated`
- `--tenant-border`
- `--tenant-border-strong`
- `--tenant-text-primary`
- `--tenant-text-secondary`
- `--tenant-text-muted`
- `--tenant-accent`
- `--tenant-accent-hover`
- `--tenant-danger`
- `--tenant-warning`
- `--tenant-success`
- `--tenant-info`
- `--tenant-radius-card`
- `--tenant-radius-button`
- `--tenant-radius-kpi`
- `--tenant-focus-ring`

Also add component utility classes:

- `.tenant-input`
- `.tenant-label`
- `.tenant-helper`
- `.tenant-page-title`
- `.tenant-page-subtitle`
- `.tenant-table`

Reason:

- Reduce repeated raw Tailwind color/radius decisions.
- Make tenant dashboard visually distinct from owner dashboard while keeping same system.
- Keep future refactors small and predictable.

### 2. Common primitives

Create/normalize reusable components:

#### `AppCard`

Surface wrapper with tokenized card radius, border, background, and text color.

#### `AppButton`

Tokenized primary button using tenant accent color, minimum tap height, disabled state.

#### `KpiCard`

Compact metric component for dashboard summary.

Props:

```ts
{ label: string; value: string | number; hint?: string }
```

#### `UiAlert`

Accessible `role="alert"` component with tones:

- `danger`
- `success`
- `warning`
- `info`

#### `UiEmptyState`

Reusable empty-state block with optional action slot.

### 3. Shell/navigation

#### `TopNav`

- Dark tokenized header.
- Responsive wrapping on small screens.
- Keeps existing slot for tenant switcher.
- Keeps user email and logout behavior.

#### `AppSidebar`

- Tokenized sidebar surface.
- Horizontal overflow nav on mobile.
- Vertical nav on large screens.
- Active route uses tenant accent.

#### `TenantLayout`

- Uses `bg-[var(--tenant-bg-app)]` and token text.
- Responsive content width and spacing.
- Keeps existing route structure and `<RouterView />`.

### 4. Tenant Dashboard

Refactor `TenantDashboard.vue` to show:

- Page title/subtitle.
- Selected tenant name.
- Current role pill.
- KPI cards:
  - tenant aktif
  - memberships
  - permissions
- Workspace summary card.
- Permission-aware quick actions:
  - Manage users if `tenant.users.read`
  - Open settings if `tenant.settings.read`
  - View audit log if `tenant.audit.read`
- Empty state when no quick action available.

### 5. Tenant Users

Refactor `TenantUsersPage.vue`:

- Page shell with member count.
- `UiAlert` for invite success, load error, and missing tenant.
- Tokenized temp password show-once panel.
- Tokenized table.
- Loading skeleton rows.
- `UiEmptyState` when no members.
- Existing permission checks preserved:
  - `tenant.users.invite`
  - `tenant.users.update`
  - `tenant.users.remove`
- Existing service calls preserved:
  - `tenantUsersService.list()`
  - `tenantUsersService.changeRole()`
  - `tenantUsersService.remove()`

### 6. User Invite Form

Refactor `UserInviteForm.vue`:

- Tokenized card, labels, inputs, select, and button.
- `UiAlert` for invite failure.
- Existing submit behavior preserved.
- Existing emitted payload preserved:

```ts
{ message: string; temporaryPassword: string }
```

### 7. Tenant Settings

Refactor `TenantSettingsPage.vue`:

- Tokenized page title/subtitle.
- `UiAlert` for error and saved state.
- `AppCard` wrapper.
- Tokenized form inputs.
- Existing service behavior preserved:
  - `tenantSettingsService.get()`
  - `tenantSettingsService.update(form)`

### 8. Tenant Audit

Refactor `TenantAuditPage.vue`:

- Tokenized title/subtitle.
- `UiAlert` for load failure.
- `AppCard` table wrapper.
- Loading skeleton rows.
- `UiEmptyState` for empty audit log.
- Existing service behavior preserved:
  - `auditService.tenant()`

## Verification

Commands run from `frontend-tenant`:

```bash
npm run typecheck
npm run build
```

Result:

- `npm run typecheck` passed.
- `npm run build` passed.
- Vite production build completed.

Build output observed:

```text
dist/index.html                   0.43 kB │ gzip:  0.29 kB
dist/assets/index-CNc03SHo.css   16.15 kB │ gzip:  4.00 kB
dist/assets/index-DE-m2xc9.js   173.01 kB │ gzip: 62.90 kB
```

## Git workflow

1. Removed earlier uncommitted tenant plan draft before commit, because user requested commit/push first.
2. Staged only files changed by tenant UI refactor.
3. Did not stage unrelated pre-existing modified files.
4. Commit created:

```bash
git commit -m "refactor(tenant): polish dashboard ui"
```

5. Push completed:

```bash
git push origin main
```

## Risks and follow-up

### Risks

- Visual-only refactor may still need browser QA against real tenant data.
- Select dropdown option colors may depend on browser default styling in dark mode.
- No automated UI tests exist for tenant dashboard interactions.

### Recommended follow-up

1. Run app locally and click through:
   - login
   - tenant switcher
   - dashboard
   - users invite/change role/remove
   - settings save
   - audit log
2. Add visual snapshots or component tests later if project adds test tooling.
3. Consider extracting shared owner/tenant dashboard primitives only after more duplication appears. Do not over-abstract now.
