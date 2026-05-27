<template>
  <header class="border-b border-slate-200 bg-white">
    <div class="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-6">
      <div>
        <p class="text-sm text-slate-500">Control Panel</p>
        <h1 class="text-base font-semibold text-slate-900">{{ title }}</h1>
      </div>

      <div class="flex items-center gap-3">
        <TenantSwitcher />
        <span class="hidden text-sm text-slate-600 md:block">{{ auth.user?.email || 'guest' }}</span>
        <LogoutButton />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import TenantSwitcher from '@/components/navigation/TenantSwitcher.vue'
import LogoutButton from '@/components/navigation/LogoutButton.vue'

defineProps<{ title: string }>()
const auth = useAuthStore()
</script>
