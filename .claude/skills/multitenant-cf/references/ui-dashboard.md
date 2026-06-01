# UI Dashboard — Custom Design System (No Third-Party Library)

The **dashboard** UI (`vue-dashboard`): authenticated, interactive, behind RBAC. This is the component system for forms, tables, modals, and toasts — the building blocks of tenant management screens. It is an SPA (no SEO concern; everything is behind login).

For the **public** site UI (anonymous, SEO-critical, content-rendering) see `ui-public.md` — a different rendering model (SSR) and different components. The **shared visual foundation** both consume — Tailwind tokens and tenant branding via CSS variables — lives in `frontend-vue-cloudflare.md` (Tailwind config section); this file builds on those tokens, it does not redefine them.

## Principles

- **Tailwind utility-first** — no custom CSS except branding CSS variables
- **Composables for logic** — no logic in the template
- **Consistent via base components** — all UI uses components from here, not raw HTML
- **Accessible** — ARIA attributes, keyboard navigation, focus management

---

## Dark / Light Theme — Vue Dashboard

The dashboard is a Vue SPA, so theme state is client-side and can use Vue composables. Do **not** use React/Next.js theme patterns here.

Use Tailwind class-based dark mode and VueUse color mode:

```bash
npm install @vueuse/core
```

```javascript
// tailwind.config.js
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: 'var(--color-primary, #3b82f6)',
          hover: 'var(--color-primary-hover, #2563eb)',
          light: 'var(--color-primary-light, #eff6ff)',
        },
        surface: {
          DEFAULT: 'rgb(var(--surface) / <alpha-value>)',
          secondary: 'rgb(var(--surface-secondary) / <alpha-value>)',
          border: 'rgb(var(--surface-border) / <alpha-value>)',
        },
        text: {
          DEFAULT: 'rgb(var(--text) / <alpha-value>)',
          muted: 'rgb(var(--text-muted) / <alpha-value>)',
          inverse: 'rgb(var(--text-inverse) / <alpha-value>)',
        },
      },
    },
  },
}
```

```css
/* src/assets/theme.css */
:root {
  color-scheme: light;
  --surface: 255 255 255;
  --surface-secondary: 248 250 252;
  --surface-border: 226 232 240;
  --text: 15 23 42;
  --text-muted: 100 116 139;
  --text-inverse: 255 255 255;
}

html.dark {
  color-scheme: dark;
  --surface: 15 23 42;
  --surface-secondary: 30 41 59;
  --surface-border: 51 65 85;
  --text: 248 250 252;
  --text-muted: 148 163 184;
  --text-inverse: 15 23 42;
}
```

```typescript
// src/composables/useTheme.ts
import { useColorMode } from '@vueuse/core'

export type ThemeMode = 'light' | 'dark' | 'auto'

export function useTheme() {
  const mode = useColorMode({
    selector: 'html',
    attribute: 'class',
    storageKey: 'dashboard-theme',
    initialValue: 'auto',
    modes: {
      light: '',
      dark: 'dark',
      auto: 'auto',
    },
  })

  const setTheme = (value: ThemeMode) => {
    mode.value = value
  }

  const toggleTheme = () => {
    mode.value = mode.value === 'dark' ? 'light' : 'dark'
  }

  return { mode, setTheme, toggleTheme }
}
```

```vue
<!-- src/components/base/ThemeToggle.vue -->
<script setup lang="ts">
import { useTheme } from '@/composables/useTheme'

const { mode, toggleTheme, setTheme } = useTheme()
</script>

<template>
  <div class="flex items-center gap-2">
    <button
      type="button"
      class="rounded-lg border border-surface-border bg-surface px-3 py-2 text-text hover:bg-surface-secondary focus:outline-none focus:ring-2 focus:ring-primary"
      @click="toggleTheme"
    >
      {{ mode === 'dark' ? 'Dark' : mode === 'light' ? 'Light' : 'Auto' }}
    </button>

    <select
      :value="mode"
      class="rounded-lg border border-surface-border bg-surface px-2 py-2 text-text"
      @change="setTheme(($event.target as HTMLSelectElement).value as 'light' | 'dark' | 'auto')"
    >
      <option value="light">Light</option>
      <option value="dark">Dark</option>
      <option value="auto">System</option>
    </select>
  </div>
</template>
```

Dashboard components must use semantic tokens (`bg-surface`, `bg-surface-secondary`, `border-surface-border`, `text-text`, `text-text-muted`) instead of hard-coded `white/slate-*` whenever the color must respond to dark/light mode. Tenant branding remains separate: `--color-primary` is still set by `useTenant.applyBranding()`.

---

## Base Components

### Button
```vue
<!-- src/components/base/BaseButton.vue -->
<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="[
      'inline-flex items-center justify-center gap-2 font-medium rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed',
      sizeClasses,
      variantClasses,
    ]"
    v-bind="$attrs"
  >
    <span v-if="loading" class="size-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
    <slot />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  type?: 'button' | 'submit' | 'reset'
  loading?: boolean
  disabled?: boolean
}>(), {
  variant: 'primary', size: 'md', type: 'button'
})

const sizeClasses = computed(() => ({
  sm: 'px-3 py-1.5 text-sm',
  md: 'px-4 py-2 text-sm',
  lg: 'px-6 py-3 text-base',
}[props.size]))

const variantClasses = computed(() => ({
  primary:   'bg-primary text-white hover:bg-primary-hover',
  secondary: 'bg-surface border border-surface-border text-text hover:bg-surface-secondary',
  ghost:     'text-text-muted hover:bg-surface-secondary hover:text-text',
  danger:    'bg-red-600 text-white hover:bg-red-700',
}[props.variant]))
</script>
```

### Input
```vue
<!-- src/components/base/BaseInput.vue -->
<template>
  <div class="flex flex-col gap-1.5">
    <label v-if="label" :for="inputId" class="text-sm font-medium text-text">
      {{ label }}
      <span v-if="required" class="text-red-500 ml-0.5">*</span>
    </label>
    <div class="relative">
      <div v-if="$slots.prefix" class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted">
        <slot name="prefix" />
      </div>
      <input
        :id="inputId"
        v-bind="$attrs"
        :value="modelValue"
        :type="type"
        :disabled="disabled"
        :placeholder="placeholder"
        :class="[
          'w-full rounded-lg border bg-surface text-text placeholder:text-text-muted transition-colors',
          'focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
          'disabled:bg-surface-secondary disabled:cursor-not-allowed',
          error ? 'border-red-400 focus:ring-red-400' : 'border-surface-border',
          $slots.prefix ? 'pl-9' : 'pl-3',
          $slots.suffix ? 'pr-9' : 'pr-3',
          'py-2 text-sm',
        ]"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
      <div v-if="$slots.suffix" class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted">
        <slot name="suffix" />
      </div>
    </div>
    <p v-if="error" class="text-xs text-red-500">{{ error }}</p>
    <p v-else-if="hint" class="text-xs text-text-muted">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  modelValue?: string | number
  label?: string
  type?: string
  placeholder?: string
  error?: string
  hint?: string
  disabled?: boolean
  required?: boolean
  id?: string
}>(), { type: 'text' })

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const inputId = computed(() => props.id ?? `input-${Math.random().toString(36).slice(2)}`)
</script>
```

> **Wiring server validation to inputs.** The `error` prop is what binds backend validation to the field. On a `422 validation_failed`, the API wrapper (`frontend-vue-cloudflare.md`) surfaces `ApiError.fields` as `{ email: "...", password: "..." }`; bind each to the matching input:
> ```vue
> <BaseInput v-model="form.email" label="Email" :error="fieldErrors.email" />
> <BaseInput v-model="form.password" type="password" label="Password" :error="fieldErrors.password" />
> ```
> Client-side validation (instant feedback) and these server-returned messages use the same rules — the backend is authoritative (`backend-golang.md`), the client mirror is convenience.

### Modal
```vue
<!-- src/components/base/BaseModal.vue -->
<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @click.self="closeOnBackdrop && emit('update:modelValue', false)"
      >
        <!-- Backdrop -->
        <div class="absolute inset-0 bg-black/50" />

        <!-- Panel -->
        <div
          :class="['relative bg-surface rounded-xl shadow-xl w-full', sizeClass]"
          role="dialog"
          :aria-label="title"
        >
          <!-- Header -->
          <div v-if="title || $slots.header" class="flex items-center justify-between px-6 py-4 border-b border-surface-border">
            <slot name="header">
              <h2 class="text-base font-semibold text-text">{{ title }}</h2>
            </slot>
            <button
              class="p-1 rounded-md text-text-muted hover:bg-surface-secondary transition-colors"
              @click="emit('update:modelValue', false)"
            >
              <!-- X icon (inline SVG, no heroicons) -->
              <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 6L6 18M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- Body -->
          <div class="px-6 py-4">
            <slot />
          </div>

          <!-- Footer -->
          <div v-if="$slots.footer" class="flex items-center justify-end gap-3 px-6 py-4 border-t border-surface-border">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  closeOnBackdrop?: boolean
}>(), { size: 'md', closeOnBackdrop: true })

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const sizeClass = computed(() => ({
  sm: 'max-w-sm', md: 'max-w-md', lg: 'max-w-lg', xl: 'max-w-xl'
}[props.size]))

// Close on Escape
const onKeydown = (e: KeyboardEvent) => { if (e.key === 'Escape') emit('update:modelValue', false) }
onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: opacity 150ms ease; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
</style>
```

### Toast / Notification
```typescript
// src/composables/useToast.ts
import { ref, readonly } from 'vue'

interface Toast {
  id: string
  message: string
  type: 'success' | 'error' | 'warning' | 'info'
  duration?: number
}

const toasts = ref<Toast[]>([])

export function useToast() {
  const add = (message: string, type: Toast['type'] = 'info', duration = 4000) => {
    const id = Math.random().toString(36).slice(2)
    toasts.value.push({ id, message, type, duration })
    if (duration > 0) setTimeout(() => remove(id), duration)
  }

  const remove = (id: string) => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  return {
    toasts: readonly(toasts),
    success: (msg: string) => add(msg, 'success'),
    error:   (msg: string) => add(msg, 'error'),
    warning: (msg: string) => add(msg, 'warning'),
    info:    (msg: string) => add(msg, 'info'),
    remove,
  }
}
```

```vue
<!-- src/components/base/ToastContainer.vue — place in App.vue -->
<template>
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 min-w-[300px]">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="['flex items-start gap-3 px-4 py-3 rounded-lg shadow-lg text-sm', typeClasses[toast.type]]"
        >
          <span class="flex-1">{{ toast.message }}</span>
          <button class="opacity-70 hover:opacity-100" @click="remove(toast.id)">✕</button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useToast } from '@/composables/useToast'
const { toasts, remove } = useToast()
const typeClasses = {
  success: 'bg-green-600 text-white',
  error:   'bg-red-600 text-white',
  warning: 'bg-amber-500 text-white',
  info:    'bg-slate-800 text-white',
}
</script>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all 200ms ease; }
.toast-enter-from { opacity: 0; transform: translateX(100%); }
.toast-leave-to   { opacity: 0; transform: translateX(100%); }
</style>
```

### Data Table
```vue
<!-- src/components/base/BaseTable.vue -->
<template>
  <div class="overflow-x-auto rounded-lg border border-surface-border">
    <table class="w-full text-sm">
      <thead class="bg-surface-secondary border-b border-surface-border">
        <tr>
          <th
            v-for="col in columns"
            :key="col.key"
            class="px-4 py-3 text-left font-medium text-text-muted"
          >
            {{ col.label }}
          </th>
        </tr>
      </thead>
      <tbody class="divide-y divide-surface-border">
        <tr v-if="loading">
          <td :colspan="columns.length" class="px-4 py-8 text-center text-text-muted">
            Loading...
          </td>
        </tr>
        <tr v-else-if="!rows.length">
          <td :colspan="columns.length" class="px-4 py-8 text-center text-text-muted">
            <slot name="empty">No data</slot>
          </td>
        </tr>
        <tr
          v-else
          v-for="(row, i) in rows"
          :key="i"
          class="hover:bg-surface-secondary transition-colors"
        >
          <td v-for="col in columns" :key="col.key" class="px-4 py-3 text-text">
            <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
              {{ row[col.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  columns: { key: string; label: string }[]
  rows: Record<string, unknown>[]
  loading?: boolean
}>()
</script>
```

---

## Inline SVG Icons (no Heroicons/Lucide)

```typescript
// src/components/icons/index.ts
// Use minimal inline SVG, not a library import
// Format: 24x24 viewBox, stroke-based
```

```vue
<!-- Usage example — create your own icon component -->
<template>
  <svg
    :class="['shrink-0', sizeClass]"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <slot />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = withDefaults(defineProps<{
  size?: 'xs' | 'sm' | 'md' | 'lg'
  strokeWidth?: number
}>(), { size: 'md', strokeWidth: 2 })

const sizeClass = computed(() => ({
  xs: 'size-3', sm: 'size-4', md: 'size-5', lg: 'size-6'
}[props.size]))
</script>
```

---

## Components Folder Structure

```
src/components/
  base/
    BaseButton.vue
    BaseInput.vue
    BaseModal.vue
    BaseTable.vue
    BaseSelect.vue
    BaseBadge.vue
    BaseAvatar.vue
    BaseCard.vue
    ToastContainer.vue
  icons/
    IconBase.vue
    IconChevron.vue
    IconUser.vue
    IconSettings.vue
    (etc — create as needed, SVG path is inlined)
  layout/
    AppSidebar.vue
    AppHeader.vue
    AppLayout.vue
  feature/
    content/
    users/
    settings/
    billing/        ← plan picker, current plan, invoices (tenant); catalog editor (platform)
```

---

## Billing UI (see `billing.md`)

Two distinct surfaces, built from the same base components — no new primitives needed:

- **Tenant billing page** (`billing:read` / `billing:manage`): current plan + status badge, a **plan picker** with a monthly/annual toggle. The annual option must surface the saving explicitly — *"Annual: pay 10 months, get 12 — save 2 months"* — because the price is `monthly × 10` (`billing.md`). A pay button calls the backend `CreateCharge` → gateway redirect/snap (Midtrans/Duitku) or shows VA / manual-transfer instructions. Invoice history in a `BaseTable`.
- **Platform catalog editor** (`RequirePlatformRole`, platform dashboard only): edit each plan's `price_monthly`, `features`, `limits`, `is_active` via `BaseInput`/`BaseSelect`. This is the "harga & fitur diatur di dashboard" surface — platform-level, writes through the platform pool (no RLS), never exposed to tenants.

```vue
<!-- plan picker: annual toggle makes the 2-months-free saving explicit -->
<template>
  <label class="flex items-center gap-2">
    <input type="checkbox" v-model="annual" />
    <span>Annual billing <BaseBadge v-if="annual" variant="success">save 2 months</BaseBadge></span>
  </label>
  <!-- price comes from the server-computed catalog; the client never sends an amount -->
  <p class="text-2xl font-semibold">{{ formatIDR(annual ? plan.price_monthly * 10 : plan.price_monthly) }}
    <span class="text-sm text-text-muted">/ {{ annual ? 'year' : 'month' }}</span></p>
</template>
```

> The displayed price is for show; the charged amount is always recomputed server-side from the catalog (`billing.md` → `PriceFor`). Never trust a price submitted by the client.

---

## Tips

- Use `v-bind="$attrs"` in base components so HTML attributes (class, id, data-*) can still be passed from outside
- `defineOptions({ inheritAttrs: false })` if you need full control over attribute forwarding
- All colors use Tailwind classes that reference a CSS variable (`bg-primary`, `text-text`) — automatically follow tenant branding
- For animation, just use Tailwind `transition-*` + Vue `<Transition>` — no animation library needed