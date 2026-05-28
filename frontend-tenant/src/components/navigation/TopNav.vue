<template>
  <header class="border-b border-[var(--tenant-border)] bg-[var(--tenant-bg-sidebar)]/95 backdrop-blur">
    <div class="mx-auto flex min-h-16 w-full max-w-7xl flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6">
      <div>
        <p class="text-sm text-[var(--tenant-text-muted)]">Tenant Control Panel</p>
        <h1 class="text-base font-semibold text-[var(--tenant-text-primary)]">{{ title }}</h1>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <slot name="actions" />
        <span class="hidden text-sm text-[var(--tenant-text-secondary)] md:block">{{ auth.user?.email || 'guest' }}</span>
        <LogoutButton />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import LogoutButton from '@/components/navigation/LogoutButton.vue'

defineProps<{ title: string }>()
const auth = useAuthStore()
</script>
