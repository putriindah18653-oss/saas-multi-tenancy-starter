<template>
  <div class="app-shell">
    <AppSidebar :items="visibleItems" />
    <div class="app-main" :class="{ 'app-main--collapsed': ui.sidebarCollapsed }">
      <TopNav :title="pageTitle">
        <template #actions>
          <TenantSwitcher />
        </template>
      </TopNav>
      <main class="content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from '@/components/navigation/TopNav.vue'
import AppSidebar from '@/components/navigation/AppSidebar.vue'
import TenantSwitcher from '@/components/navigation/TenantSwitcher.vue'
import { canTenant, type TenantPermission } from '@/services/rbac'
import { useTenantStore } from '@/stores/tenant'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const tenant = useTenantStore()
const route = useRoute()
const pageTitle = computed(() => (route.meta.title as string | undefined) || 'Dashboard Tenant')

const icons = {
  dashboard: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>`,
  users: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
  audit: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>`,
  settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1.08-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1.08 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1.08z"/></svg>`,
}

type SidebarChildItem = {
  to: string
  label: string
  permission?: TenantPermission
  badge?: string
  mock?: boolean
}

type SidebarLinkItem = {
  type?: 'link'
  to?: string
  label: string
  icon?: string
  permission?: TenantPermission
  children?: SidebarChildItem[]
  badge?: string
  mock?: boolean
}

type SidebarGroupItem = {
  type: 'group'
  label: string
}

type SidebarItem = SidebarLinkItem | SidebarGroupItem

const items: SidebarItem[] = [
  { type: 'group', label: 'Overview' },
  { to: '/tenant', label: 'Dashboard', icon: icons.dashboard, permission: 'tenant.dashboard.read' },

  { type: 'group', label: 'Operations' },
  { to: '/tenant/users', label: 'Users', icon: icons.users, permission: 'tenant.users.read' },

  { type: 'group', label: 'System' },
  { to: '/tenant/settings', label: 'Settings', icon: icons.settings, permission: 'tenant.settings.read' },
  { to: '/tenant/audit', label: 'Audit Log', icon: icons.audit, permission: 'tenant.audit.read' },
]

function canSee(permission?: TenantPermission) {
  return !permission || canTenant(tenant.selectedMembership, permission)
}

const visibleItems = computed<SidebarItem[]>(() => {
  const filtered: SidebarItem[] = []

  for (const item of items) {
    if (item.type === 'group') {
      filtered.push(item)
      continue
    }

    const children = item.children?.filter((child) => canSee(child.permission))
    if (item.children?.length && !children?.length) continue
    if (!item.children?.length && !canSee(item.permission)) continue

    filtered.push({ ...item, children })
  }

  return filtered.filter((item, index, all) => {
    if (item.type !== 'group') return true
    return all.slice(index + 1).some((next) => next.type !== 'group')
  })
})

onMounted(() => {
  ui.syncDesktopSidebar()
  window.addEventListener('resize', ui.syncDesktopSidebar)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', ui.syncDesktopSidebar)
})
</script>
