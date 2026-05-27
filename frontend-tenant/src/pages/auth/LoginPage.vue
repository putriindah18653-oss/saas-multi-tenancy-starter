<template>
  <div>
    <h2 class="text-2xl font-semibold text-slate-900">Login</h2>
    <p class="mt-1 text-sm text-slate-600">Masuk untuk akses dashboard sesuai role Anda.</p>

    <form class="mt-6 space-y-4" @submit.prevent="submit">
      <div>
        <label class="mb-1 block text-sm text-slate-700">Email</label>
        <input v-model="form.email" type="email" autocomplete="email" placeholder="you@company.com" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>
      <div>
        <label class="mb-1 block text-sm text-slate-700">Password</label>
        <input v-model="form.password" type="password" autocomplete="current-password" placeholder="••••••••" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>

      <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

      <button :disabled="loading" class="w-full rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60">
        {{ loading ? 'Loading...' : 'Login' }}
      </button>
    </form>

    <p class="mt-5 text-sm text-slate-600">Gunakan akun yang sudah disediakan admin untuk masuk.</p>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'

const router = useRouter()
const auth = useAuthStore()
const tenant = useTenantStore()
const loading = ref(false)
const error = ref('')
const form = reactive({ email: '', password: '' })

async function submit() {
  loading.value = true
  error.value = ''
  try {
    const res = await authService.login({ email: form.email, password: form.password })
    const payload = res.data.data
    auth.setSession({ accessToken: payload.access_token, refreshToken: payload.refresh_token, user: payload.user })
    tenant.setMemberships(payload.tenant_memberships || [])
    router.push({ name: auth.defaultHomeRoute })
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Login gagal'
  } finally {
    loading.value = false
  }
}
</script>
