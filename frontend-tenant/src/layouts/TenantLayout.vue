<template>
  <div class="app-shell">
    <AppSidebar :items="visibleItems" />
    <div class="app-main" :class="{ 'app-main--collapsed': ui.sidebarCollapsed }">
      <TopNav title="Dashboard Tenant">
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
import { computed } from 'vue'
import TopNav from '@/components/navigation/TopNav.vue'
import AppSidebar from '@/components/navigation/AppSidebar.vue'
import TenantSwitcher from '@/components/navigation/TenantSwitcher.vue'
import { canTenant, type TenantPermission } from '@/services/rbac'
import { useTenantStore } from '@/stores/tenant'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const tenant = useTenantStore()

type SidebarItem = {
  to: string
  label: string
  icon: string
  permission: TenantPermission
}

const items: SidebarItem[] = [
  { to: '/tenant', label: 'Dashboard', icon: '▦', permission: 'tenant.dashboard.read' },
  { to: '/tenant/users', label: 'Users', icon: '👥', permission: 'tenant.users.read' },
  { to: '/tenant/settings', label: 'Settings', icon: '⚙', permission: 'tenant.settings.read' },
  { to: '/tenant/audit', label: 'Audit Log', icon: '◇', permission: 'tenant.audit.read' },
]

const visibleItems = computed(() => items.filter((item) => canTenant(tenant.selectedMembership, item.permission)))
</script>
