<template>
  <div class="min-h-screen bg-slate-100">
    <TopNav title="Tenant">
      <template #actions>
        <TenantSwitcher />
      </template>
    </TopNav>
    <div class="mx-auto grid max-w-7xl gap-6 p-6 lg:grid-cols-[240px_1fr]">
      <AppSidebar :items="items" />
      <main>
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import TopNav from '@/components/navigation/TopNav.vue'
import AppSidebar from '@/components/navigation/AppSidebar.vue'
import TenantSwitcher from '@/components/navigation/TenantSwitcher.vue'

const items = [
  { to: '/tenant', label: 'Dashboard' },
  { to: '/tenant/users', label: 'Users' },
  { to: '/tenant/settings', label: 'Settings' },
  { to: '/tenant/audit', label: 'Audit Log' },
]
</script>
