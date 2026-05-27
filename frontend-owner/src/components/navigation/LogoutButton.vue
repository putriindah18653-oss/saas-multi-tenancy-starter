<template>
  <button
    class="rounded-md border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-100"
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
    await authService.logout()
  } catch {
    // ignore network errors on logout UI
  }
  auth.clearSession()
  router.push('/auth/login')
}
</script>
