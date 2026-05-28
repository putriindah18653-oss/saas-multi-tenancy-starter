<template>
  <div class="app-shell">
    <AppSidebar :items="items" />
    <div class="app-main" :class="{ 'app-main--collapsed': ui.sidebarCollapsed }">
      <TopNav title="Dashboard Owner" />
      <main class="content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import TopNav from '@/components/navigation/TopNav.vue'
import AppSidebar from '@/components/navigation/AppSidebar.vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

const items = [
  { to: '/app', label: 'Dashboard', icon: '▦' },
  { to: '/app/tenants', label: 'Tenants', icon: '◉', children: true },
  { to: '/app/audit', label: 'Audit Log', icon: '◇' },
]
</script>
