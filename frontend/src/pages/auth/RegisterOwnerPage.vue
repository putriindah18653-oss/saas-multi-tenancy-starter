<template>
  <div>
    <h2 class="text-2xl font-semibold text-slate-900">Register Owner</h2>
    <p class="mt-1 text-sm text-slate-600">Buat akun owner pertama untuk mulai setup aplikasi.</p>

    <form class="mt-6 space-y-4" @submit.prevent="submit">
      <div>
        <label class="mb-1 block text-sm text-slate-700">Full name</label>
        <input v-model="form.full_name" autocomplete="name" placeholder="Jane Doe" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>
      <div>
        <label class="mb-1 block text-sm text-slate-700">Email</label>
        <input v-model="form.email" type="email" autocomplete="email" placeholder="owner@company.com" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>
      <div>
        <label class="mb-1 block text-sm text-slate-700">Password</label>
        <input v-model="form.password" type="password" autocomplete="new-password" placeholder="Min. 8 karakter" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>

      <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

      <button :disabled="loading" class="w-full rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60">
        {{ loading ? 'Loading...' : 'Register owner' }}
      </button>
    </form>

    <p class="mt-5 text-sm text-slate-600">
      Sudah punya akun? <RouterLink to="/auth/login" class="text-slate-900 underline">Login</RouterLink>
    </p>
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
const form = reactive({ full_name: '', email: '', password: '' })

async function submit() {
  loading.value = true
  error.value = ''
  try {
    const res = await authService.registerOwner({ full_name: form.full_name, email: form.email, password: form.password })
    const payload = res.data.data
    auth.setSession({ accessToken: payload.access_token, refreshToken: payload.refresh_token, user: payload.user })
    tenant.setMemberships(payload.tenant_memberships || [])
    router.push({ name: 'app-home' })
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Register owner gagal'
  } finally {
    loading.value = false
  }
}
</script>
