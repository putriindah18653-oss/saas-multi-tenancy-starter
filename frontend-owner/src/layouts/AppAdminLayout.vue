<template>
  <div class="min-h-screen bg-[var(--bg-app)] text-[var(--text-primary)]">
    <TopNav title="Owner dashboard" />
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

const items = [
  { to: '/app', label: 'Dashboard' },
  { to: '/app/tenants', label: 'Tenants' },
  { to: '/app/audit', label: 'Audit Log' },
]
</script>
