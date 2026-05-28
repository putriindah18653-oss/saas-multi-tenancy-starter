<template>
  <header class="sticky top-0 z-20 border-b border-[var(--border)] bg-[var(--bg-app)]/85 backdrop-blur-xl">
    <div class="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-4 sm:px-6">
      <div>
        <p class="text-xs font-medium uppercase tracking-wide text-[var(--text-muted)]">Control Panel</p>
        <h1 class="text-base font-semibold text-[var(--text-primary)]">{{ title }}</h1>
      </div>

      <div class="flex items-center gap-3">
        <slot name="actions" />
        <div class="hidden rounded-full border border-[var(--border)] bg-white/[0.03] px-3 py-1.5 text-sm text-[var(--text-secondary)] md:block">
          {{ auth.user?.email || 'guest' }}
        </div>
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
