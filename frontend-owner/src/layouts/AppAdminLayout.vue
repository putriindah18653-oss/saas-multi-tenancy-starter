<template>
  <div class="min-h-screen bg-slate-100">
    <TopNav title="App Admin" />
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

const items = [
  { to: '/app', label: 'Dashboard' },
  { to: '/app/tenants', label: 'Tenants' },
  { to: '/app/audit', label: 'Audit Log' },
]
</script>
