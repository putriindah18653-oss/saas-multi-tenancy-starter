<template>
  <div class="flex min-h-[calc(100vh-8rem)] items-center justify-center p-4">
    <section class="w-full max-w-xl rounded-[24px] border border-[var(--border)] bg-[var(--surface)] p-8 text-center text-[var(--text-primary)]">
      <p class="text-sm font-semibold uppercase tracking-wide text-[var(--warning)]">Permission needed</p>
      <h1 class="mt-3 text-3xl font-semibold tracking-tight">Access unavailable</h1>
      <p class="mt-3 text-sm leading-6 text-[var(--text-muted)]">
        Akun Anda belum punya izin untuk membuka halaman atau aksi ini. Hubungi admin jika akses diperlukan.
      </p>
      <RouterLink
        :to="homeRoute"
        class="mt-6 inline-flex min-h-10 items-center rounded-[var(--radius-button)] bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--accent-hover)]"
      >
        Back to dashboard
      </RouterLink>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const homeRoute = computed(() => ({ name: auth.defaultHomeRoute }))
</script>
