<template>
  <div class="min-h-screen bg-[var(--tenant-bg-app)] text-[var(--tenant-text-primary)]">
    <TopNav title="Tenant dashboard">
      <template #actions>
        <TenantSwitcher />
      </template>
    </TopNav>
    <div class="mx-auto grid w-full max-w-7xl gap-5 px-4 py-5 sm:px-6 lg:grid-cols-[240px_1fr]">
      <AppSidebar :items="items" />
      <main class="min-w-0">
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
