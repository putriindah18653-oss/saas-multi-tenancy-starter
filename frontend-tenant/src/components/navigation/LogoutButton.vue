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
import { useTenantStore } from '@/stores/tenant'

const router = useRouter()
const auth = useAuthStore()
const tenant = useTenantStore()

async function onLogout() {
  try {
    if (auth.refreshToken) {
      await authService.logout(auth.refreshToken)
    }
  } catch {
    // ignore network errors on logout UI
  }
  auth.clearSession()
  tenant.clearTenant()
  router.push('/auth/login')
}
</script>
