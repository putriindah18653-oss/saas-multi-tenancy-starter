# Frontend Owner Dashboard UI Refactor Implementation Plan

> **For Hermes:** Use `dashboard-ui-ux`, `popular-web-designs`, and `writing-plans`. Use `safe-resource-gated-execution` for build/typecheck. If executing via workers, use `subagent-driven-development` task-by-task.

**Goal:** Refactor `frontend-owner` UI from plain slate admin screens into polished, consistent, responsive dashboard UI, starting with login, then app dashboard index, then tenant/audit/secondary pages.

**Architecture:** Keep current Vue 3 + Vite + TypeScript + Pinia + Vue Router + Tailwind stack. Do not add large UI frameworks. Introduce token-based dashboard styling and reusable owner UI components first, then migrate pages one flow at a time. Preserve existing auth/RBAC/API contracts.

**Design direction:** Dark-first Linear-inspired dashboard using `dashboard-ui-ux` rules: tokenized surfaces, compact KPI cards, responsive shell, accessible forms, loading/error/empty states. Avoid repeating raw colors inside pages; raw values live in global tokens only.

**Current inspected files:**
- `frontend-owner/package.json`
- `frontend-owner/src/app/styles.css`
- `frontend-owner/src/router/index.ts`
- `frontend-owner/src/layouts/AuthLayout.vue`
- `frontend-owner/src/layouts/AppAdminLayout.vue`
- `frontend-owner/src/components/common/AppButton.vue`
- `frontend-owner/src/components/common/AppCard.vue`
- `frontend-owner/src/components/navigation/AppSidebar.vue`
- `frontend-owner/src/components/navigation/TopNav.vue`
- `frontend-owner/src/pages/auth/LoginPage.vue`
- `frontend-owner/src/pages/auth/ChangePasswordPage.vue`
- `frontend-owner/src/pages/app/AppDashboard.vue`
- `frontend-owner/src/pages/app/TenantList.vue`
- `frontend-owner/src/pages/app/TenantCreate.vue`
- `frontend-owner/src/pages/app/TenantDetail.vue`
- `frontend-owner/src/pages/app/AppAuditPage.vue`

---

## Non-negotiable constraints

1. Do not change backend API contracts.
2. Do not change auth/token storage behavior unless explicitly requested.
3. Do not add dependencies for this refactor unless a later task proves need.
4. Do not commit `dist/`, `.env`, or secrets.
5. Build/typecheck commands are resource-gated for PortalOnline; run serially, not in parallel.
6. Preserve existing route names and permissions in `frontend-owner/src/router/index.ts`.
7. Keep implementation page-by-page: login → dashboard index → tenant list/create/detail → audit → error/change-password polish.
8. Every data page must show loading, error, empty, and success states.
9. Every form must have accessible labels, disabled submitting state, and inline errors.
10. Use tokens from `src/app/styles.css` for colors/background/borders/radii.

---

## Target visual system

### Token direction

Add dark-first CSS tokens in `frontend-owner/src/app/styles.css`:

- `--bg-app`
- `--bg-sidebar`
- `--surface`
- `--surface-elevated`
- `--border`
- `--border-strong`
- `--text-primary`
- `--text-secondary`
- `--text-muted`
- `--accent`
- `--accent-hover`
- `--danger`
- `--warning`
- `--success`
- `--info`
- `--radius-card`
- `--radius-button`
- `--radius-kpi`
- button severity tokens

Linear-inspired palette:

```css
:root {
  --bg-app: #08090a;
  --bg-sidebar: #0f1011;
  --surface: #141516;
  --surface-elevated: #191a1b;
  --border: rgba(255, 255, 255, 0.08);
  --border-strong: rgba(255, 255, 255, 0.14);
  --text-primary: #f7f8f8;
  --text-secondary: #d0d6e0;
  --text-muted: #8a8f98;
  --accent: #5e6ad2;
  --accent-hover: #7170ff;
  --danger: #ff6467;
  --warning: #f59e0b;
  --success: #27a644;
  --info: #2f9bff;
  --radius-card: 12px;
  --radius-button: 10px;
  --radius-kpi: 8px;
}
```

### UX direction

- App background: dark gradient + subtle glow.
- Shell: left sidebar persistent desktop, mobile drawer later if scope allows.
- Cards: subtle border, low-contrast surfaces, no heavy shadow.
- KPI cards: compact/slim, medium rounding, clear metric.
- Tables: tokenized header/body, horizontal scroll on mobile.
- Forms: tokenized fields, visible focus ring, helper/error text.
- Empty states: clear copy and CTA when allowed.
- Error states: inline `UiAlert`, not bare red text.

---

## Phase 0 — Baseline and safety

### Task 0.1: Confirm clean working tree

**Objective:** Avoid mixing UI refactor with unrelated changes.

**Files:** none.

**Steps:**
1. Run:
   ```bash
   git status --short
   ```
2. Expected: clean or only planned files after this plan is created.
3. If dirty, inspect and decide before edits.

**Verification:** No unrelated files changed.

---

### Task 0.2: Baseline build

**Objective:** Know current build state before refactor.

**Files:** none.

**Steps:**
1. Acquire PortalOnline build/test resource lane.
2. Run serially:
   ```bash
   cd frontend-owner
   npm run build
   ```
3. Record output.

**Expected:** build passes.

**If fail:** stop and fix/record baseline failure before UI work.

---

## Phase 1 — Design tokens and reusable primitives

### Task 1.1: Add global dashboard tokens

**Objective:** Establish one source of truth for dashboard colors, radii, focus, and body background.

**Files:**
- Modify: `frontend-owner/src/app/styles.css`

**Steps:**
1. Keep Tailwind directives.
2. Add `@layer base` token definitions.
3. Add body defaults:
   - background `var(--bg-app)`
   - text `var(--text-primary)`
   - font smoothing
4. Add reusable utility classes only if they reduce repetition:
   - `.focus-ring`
   - `.dashboard-surface`
   - `.dashboard-muted`

**Acceptance:** Pages can use Tailwind arbitrary values such as `bg-[var(--surface)]` and `text-[var(--text-primary)]`.

**Verification:**
```bash
cd frontend-owner
npm run build
```

---

### Task 1.2: Replace `AppButton` with typed tokenized button

**Objective:** Make buttons reusable and accessible across login/forms/tables.

**Files:**
- Modify: `frontend-owner/src/components/common/AppButton.vue`

**Props:**
```ts
variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
size?: 'sm' | 'md' | 'lg'
loading?: boolean
disabled?: boolean
type?: 'button' | 'submit' | 'reset'
ariaLabel?: string
```

**Rules:**
- Default `type="button"`.
- Loading disables interaction.
- Use token colors.
- Preserve slot.
- Include visible focus ring.

**Verification:** Build passes and existing imports remain valid.

---

### Task 1.3: Replace `AppCard` with tokenized card shell

**Objective:** Standardize dashboard card look.

**Files:**
- Modify: `frontend-owner/src/components/common/AppCard.vue`

**Props:**
```ts
padding?: 'none' | 'sm' | 'md' | 'lg'
interactive?: boolean
```

**Rules:**
- Use `bg-[var(--surface)]`, `border-[var(--border)]`, `rounded-[var(--radius-card)]`.
- Optional hover border when interactive.

**Verification:** Existing `<AppCard>` usage still renders.

---

### Task 1.4: Add base form/input components

**Objective:** Avoid repeated input markup in login/change-password/tenant forms.

**Files:**
- Create: `frontend-owner/src/components/common/AppInput.vue`
- Create: `frontend-owner/src/components/common/AppSelect.vue`
- Optional later: `AppPasswordInput.vue` if show/hide password is included in login task.

**Props for `AppInput`:**
```ts
modelValue: string
id?: string
name?: string
label?: string
placeholder?: string
type?: string
helperText?: string
error?: string
disabled?: boolean
required?: boolean
autocomplete?: string
maxlength?: number
minlength?: number
```

**Rules:**
- Support `v-model` via `update:modelValue`.
- Link helper/error with `aria-describedby`.
- Set `aria-invalid` when error exists.
- Tokenized input style.

**Verification:** Local import compiles.

---

### Task 1.5: Add feedback components

**Objective:** Replace bare red text and empty rows with consistent feedback UI.

**Files:**
- Create: `frontend-owner/src/components/common/AppAlert.vue`
- Create: `frontend-owner/src/components/common/AppEmptyState.vue`
- Create: `frontend-owner/src/components/common/AppSkeleton.vue`
- Optional: `frontend-owner/src/components/common/AppBadge.vue`

**Rules:**
- `AppAlert` variants: `info`, `success`, `warning`, `error`; `role="alert"` for error.
- `AppEmptyState` accepts title, description, action slot.
- `AppSkeleton` tokenized background, no layout shift.
- `AppBadge` semantic status variants.

**Verification:** Build passes.

---

### Task 1.6: Add page shell helpers

**Objective:** Standardize page headings/actions.

**Files:**
- Create: `frontend-owner/src/components/common/PageHeader.vue`
- Optional: `frontend-owner/src/components/common/KpiCard.vue`

**`PageHeader` props:**
```ts
title: string
description?: string
eyebrow?: string
```

**Slots:**
- `actions`
- optional `meta`

**`KpiCard` props:**
```ts
label: string
value: string | number
helper?: string
trend?: string
variant?: 'default' | 'success' | 'warning' | 'danger' | 'info'
```

**Rules:** KPI card must be compact and `rounded-[var(--radius-kpi)]`.

**Verification:** Build passes.

---

## Phase 2 — Auth flow first: Login page

### Task 2.1: Inspect and preserve login behavior

**Objective:** Ensure UI refactor does not break auth logic.

**Files:**
- Read/modify: `frontend-owner/src/pages/auth/LoginPage.vue`
- Reference: `frontend-owner/src/services/auth.ts`
- Reference: `frontend-owner/src/stores/auth.ts`

**Existing behavior to preserve:**
- Calls `authService.login`.
- Fetches `authService.permissions`.
- Stores session with permissions.
- Routes to `auth.defaultHomeRoute`.
- Shows backend error fallback `Login gagal`.

**Verification:** No API payload or route name changes.

---

### Task 2.2: Refactor Login UI to premium owner portal

**Objective:** Make login first polished page.

**Files:**
- Modify: `frontend-owner/src/pages/auth/LoginPage.vue`
- Possibly modify: `frontend-owner/src/layouts/AuthLayout.vue`

**Design requirements:**
- Full-height dark auth page.
- Left/upper brand panel: “Owner Console” / “PortalOnline Admin”.
- Login card with tokenized surface.
- Email/password using `AppInput` or `AppPasswordInput`.
- Submit using `AppButton` with `loading`.
- Error using `AppAlert`.
- Helper copy: admin-provided account.
- Add security microcopy: “Protected owner-only workspace”.
- Responsive: single-column at mobile, split/visual panel on desktop.

**Accessibility:**
- Email label linked to input.
- Password label linked to input.
- Error alert role.
- Button disabled while loading.

**Verification:**
```bash
cd frontend-owner
npm run build
```

---

### Task 2.3: Polish Change Password page with same auth visual system

**Objective:** Keep forced password flow consistent with login.

**Files:**
- Modify: `frontend-owner/src/pages/auth/ChangePasswordPage.vue`

**Requirements:**
- Use same auth card style.
- Use `AppInput`/password inputs.
- Show password rules in helper block.
- Client-side confirm password mismatch remains.
- Error via `AppAlert`.
- Success via `AppAlert` or inline success.
- Preserve route push to `auth.defaultHomeRoute`.

**Verification:** Build passes.

---

## Phase 3 — App shell / navigation foundation

### Task 3.1: Refactor `AppAdminLayout` to dashboard shell

**Objective:** Create consistent dark dashboard shell for all owner pages.

**Files:**
- Modify: `frontend-owner/src/layouts/AppAdminLayout.vue`
- Modify if needed: `frontend-owner/src/components/navigation/TopNav.vue`
- Modify if needed: `frontend-owner/src/components/navigation/AppSidebar.vue`

**Requirements:**
- `min-h-screen bg-[var(--bg-app)] text-[var(--text-primary)]`.
- Sidebar region with `--bg-sidebar`.
- Main region with responsive padding.
- Add skip-to-content link if practical.
- Maintain current routes:
  - `/app`
  - `/app/tenants`
  - `/app/audit`
- Active route clearly highlighted.
- User info/logout remains accessible.

**Mobile baseline:**
- If full drawer is too much for first pass, allow top-stacked nav that does not overflow.
- Document drawer as follow-up if not implemented.

**Verification:** Build passes.

---

### Task 3.2: Tokenize TopNav and Sidebar

**Objective:** Remove light slate visual mismatch.

**Files:**
- Modify: `frontend-owner/src/components/navigation/TopNav.vue`
- Modify: `frontend-owner/src/components/navigation/AppSidebar.vue`
- Modify: `frontend-owner/src/components/navigation/LogoutButton.vue`

**Requirements:**
- Use token text/border/surface classes.
- Sidebar links: active, hover, focus states.
- Logout button uses `AppButton` variant ghost/secondary.
- No hardcoded `slate-*` where token exists.

**Verification:** Keyboard focus visible on nav links/buttons.

---

## Phase 4 — Dashboard index second

### Task 4.1: Define dashboard data strategy

**Objective:** Dashboard index can ship without backend changes.

**Files:**
- Modify: `frontend-owner/src/pages/app/AppDashboard.vue`
- Reference: `frontend-owner/src/services/tenants.ts`
- Reference: `frontend-owner/src/services/audit.ts`

**Data available now:**
- Current user from `useAuthStore`.
- Tenants from `tenantsService.list()`.
- App audit logs from `auditService.app()` if permission allows.

**Plan:**
- Load tenants list for KPI counts.
- Optionally load latest audit logs if user can `app.audit.read`.
- Derive metrics client-side:
  - total tenants
  - active tenants
  - inactive/deleted tenants
  - recent audit count/latest action

**Do not:** Add new backend endpoint yet.

---

### Task 4.2: Build dashboard hero and KPI section

**Objective:** Make `/app` useful and polished.

**Files:**
- Modify: `frontend-owner/src/pages/app/AppDashboard.vue`
- Use: `PageHeader.vue`, `KpiCard.vue`, `AppCard.vue`, `AppAlert.vue`, `AppSkeleton.vue`, `AppEmptyState.vue`

**UI sections:**
1. Header:
   - title: “Owner Dashboard”
   - description: role-aware welcome copy.
   - primary action: “Create tenant” if `app.tenants.create`.
2. KPI grid compact cards:
   - Total tenants
   - Active tenants
   - Inactive/deleted tenants
   - Recent audit events
3. Quick actions card:
   - Manage tenants
   - View audit log
4. Recent activity card:
   - latest 5 audit actions or empty state.

**States:**
- Loading skeleton for KPI cards.
- Error alert if tenant load fails.
- Empty state if no tenants.

**Verification:** Build passes.

---

### Task 4.3: Add permission-aware dashboard content

**Objective:** Avoid showing actions user cannot perform.

**Files:**
- Modify: `frontend-owner/src/pages/app/AppDashboard.vue`

**Rules:**
- Use `canOwner(auth.user, 'app.tenants.create')` for Create tenant CTA.
- Use `canOwner(auth.user, 'app.audit.read')` before loading/showing audit log card.
- If no permission, show neutral “limited access” state, not broken UI.

**Verification:** Typecheck/build.

---

## Phase 5 — Tenant pages

### Task 5.1: Refactor Tenant List page

**Objective:** Convert tenants table into dashboard data page.

**Files:**
- Modify: `frontend-owner/src/pages/app/TenantList.vue`

**Requirements:**
- Use `PageHeader` with description and Create Tenant action.
- Use `AppAlert` for load error.
- Use loading table skeleton.
- Use `AppEmptyState` when no tenants.
- Tokenized table with horizontal scroll on mobile.
- Status displayed with `AppBadge`.
- Detail action as accessible button/link.
- Preserve permission `canCreate`.

**Optional, only if simple:**
- Add local search input for name/slug/status filtering.
- Do not add remote search API.

**Verification:** Build passes.

---

### Task 5.2: Refactor Tenant Create form

**Objective:** Make create flow consistent and accessible.

**Files:**
- Modify: `frontend-owner/src/pages/app/TenantCreate.vue`

**Requirements:**
- Use `PageHeader` with breadcrumb/back link.
- Use `AppCard`, `AppInput`, `AppButton`, `AppAlert`.
- Add client validation:
  - name required
  - slug optional but if present only lowercase letters/numbers/hyphen if backend expects slug format; if unsure, helper only, no strict validation.
- Disable submit while loading.
- Cancel link styled as secondary button.
- Preserve `tenantsService.create` payload and redirect to detail.

**Verification:** Build passes.

---

### Task 5.3: Refactor Tenant Detail form

**Objective:** Improve edit/status/delete UX and remove native confirm.

**Files:**
- Modify: `frontend-owner/src/pages/app/TenantDetail.vue`
- Optional create: `frontend-owner/src/components/common/AppConfirmDialog.vue`

**Requirements:**
- Use `PageHeader` with tenant name/slug once loaded.
- Show skeleton while loading.
- Error via `AppAlert` with retry button.
- Form card uses `AppInput` and `AppSelect`.
- Status uses `AppBadge` in header and select in form.
- Save button permission-aware.
- Soft delete uses confirmation dialog, not `window.confirm`, if dialog component is created.
- Preserve `canManage`, `canDelete`, update/remove service calls.

**Verification:** Build passes.

---

## Phase 6 — Audit page and utility states

### Task 6.1: Refactor App Audit page

**Objective:** Make audit log readable and data-dense.

**Files:**
- Modify: `frontend-owner/src/pages/app/AppAuditPage.vue`

**Requirements:**
- Use `PageHeader`.
- Use `AppAlert` for errors.
- Use table skeleton loading.
- Use `AppEmptyState` if no logs.
- Tokenized table.
- Format date consistently using local helper in page or new `formatDateTime` helper.
- Show action as monospace/pill if useful.
- Keep actor/tenant/resource columns readable on mobile via horizontal scroll.

**Optional helper:**
- Create `frontend-owner/src/utils/format.ts` with `formatDateTime(value: string)`.

**Verification:** Build passes.

---

### Task 6.2: Refactor Forbidden page

**Objective:** Make permission failure fit dashboard/auth visual system.

**Files:**
- Modify: `frontend-owner/src/pages/errors/ForbiddenPage.vue`

**Requirements:**
- Tokenized centered card.
- Explain no permission.
- Button back to dashboard if authenticated.
- Avoid scary/security-leaking wording.

**Verification:** Build passes.

---

## Phase 7 — Responsive and accessibility pass

### Task 7.1: Mobile check for auth and dashboard shell

**Objective:** Ensure no broken layout at 360px.

**Files:**
- Adjust files changed above.

**Checklist:**
- Login works at 360px width.
- Dashboard shell no horizontal overflow.
- Sidebar/nav usable.
- Buttons at least ~44px tap target where practical.
- Forms stack vertically.
- Tables are horizontally scrollable or use card strategy.

**Verification:** Browser manual or screenshot if app can be served.

---

### Task 7.2: Accessibility check

**Objective:** Catch basic keyboard/screen-reader issues.

**Checklist:**
- All inputs have labels.
- Error text linked via `aria-describedby` where using `AppInput`.
- Error alerts use role alert.
- Buttons have type attribute.
- Icon-only buttons have `aria-label`.
- Focus ring visible.
- No color-only status meaning.

**Verification:** Manual plus build.

---

## Phase 8 — Verification and review

### Task 8.1: Final static verification

**Objective:** Confirm app compiles.

**Commands:**
```bash
cd frontend-owner
npm run build
npm audit --audit-level=moderate --omit=dev
```

**Expected:**
- Build passes.
- Audit has 0 moderate+ prod vulnerabilities or report findings.

**Note:** `npm run typecheck` can run separately if desired; `npm run build` already runs `vue-tsc --noEmit && vite build`.

---

### Task 8.2: Runtime smoke test

**Objective:** Catch visual/runtime errors not seen by TypeScript.

**Commands:**
```bash
cd frontend-owner
npm run dev -- --host 0.0.0.0
```

**Manual paths:**
- `/auth/login`
- `/auth/change-password`
- `/app`
- `/app/tenants`
- `/app/tenants/create`
- `/app/audit`
- `/forbidden`

**Check:**
- Console has no Vue runtime errors.
- Main layouts render.
- Loading/error/empty states visible where applicable.

---

### Task 8.3: Review using `audit-code-review`

**Objective:** Since this is a review/closeout step, use the general review routing rule.

**Review requirements:**
- Cite changed files with line ranges.
- Confirm no auth/RBAC/API behavior changes beyond UI.
- Confirm no new dependency introduced unless justified.
- Confirm loading/error/empty states exist for data pages.
- Confirm build/audit outputs recorded.

---

## Suggested execution order and commits

1. `refactor(owner-ui): add dashboard tokens and primitives`
   - Tasks 1.1–1.6
2. `refactor(owner-auth): redesign login and password pages`
   - Tasks 2.1–2.3
3. `refactor(owner-shell): update dashboard navigation shell`
   - Tasks 3.1–3.2
4. `feat(owner-dashboard): add KPI dashboard overview`
   - Tasks 4.1–4.3
5. `refactor(owner-tenants): polish tenant management pages`
   - Tasks 5.1–5.3
6. `refactor(owner-audit): polish audit and error pages`
   - Tasks 6.1–6.2
7. `test(owner-ui): verify responsive dashboard refactor`
   - Tasks 7.1–8.3

---

## Definition of Done

- Login page has new tokenized dashboard style.
- Dashboard index shows useful owner metrics/actions/activity.
- Tenant list/create/detail pages are tokenized and stateful.
- Audit page is readable, responsive, and stateful.
- Error/change-password pages visually consistent.
- Reusable primitives exist and are typed.
- No broad dependency added.
- `npm run build` passes.
- `npm audit --audit-level=moderate --omit=dev` passes or reported.
- Review performed with `audit-code-review` before close.
