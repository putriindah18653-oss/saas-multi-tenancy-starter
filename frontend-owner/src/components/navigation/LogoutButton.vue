<template>
  <button
    type="button"
    class="rounded-[var(--radius-button)] border border-[var(--border-strong)] bg-white/[0.03] px-3 py-1.5 text-sm text-[var(--text-secondary)] transition hover:bg-white/10 hover:text-[var(--text-primary)]"
    @click="onLogout"
  >
    Logout
  </button>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/auth'
const router = useRouter()
const auth = useAuthStore()

async function onLogout() {
  try {
    if (auth.refreshToken) {
      await authService.logout(auth.refreshToken)
    }
  } catch (err) {
    console.warn('[logout] server revocation failed', err)
  }
  auth.clearSession()
  router.push('/auth/login')
}
</script>
