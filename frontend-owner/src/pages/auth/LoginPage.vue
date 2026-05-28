<template>
  <div>
    <p class="text-sm font-medium text-[var(--accent)]">Welcome back</p>
    <h2 class="mt-2 text-2xl font-semibold tracking-tight text-[var(--text-primary)]">Login</h2>
    <p class="mt-2 text-sm text-[var(--text-muted)]">Masuk untuk akses dashboard sesuai role Anda.</p>

    <form class="mt-7 space-y-5" @submit.prevent="submit">
      <div>
        <label for="email" class="owner-label">Email</label>
        <input
          id="email"
          v-model="form.email"
          type="email"
          autocomplete="email"
          placeholder="you@company.com"
          required
          class="owner-input"
        />
      </div>
      <div>
        <label for="password" class="owner-label">Password</label>
        <input
          id="password"
          v-model="form.password"
          type="password"
          autocomplete="current-password"
          placeholder="••••••••"
          required
          class="owner-input"
        />
      </div>

      <UiAlert v-if="error" title="Login gagal" tone="danger">{{ error }}</UiAlert>

      <AppButton type="submit" :disabled="loading" class="w-full">
        {{ loading ? 'Loading...' : 'Login' }}
      </AppButton>
    </form>

    <p class="mt-6 rounded-[var(--radius-card)] border border-[var(--border)] bg-white/[0.03] p-3 text-xs leading-5 text-[var(--text-muted)]">
      Gunakan akun yang sudah disediakan admin untuk masuk.
    </p>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppButton from '@/components/common/AppButton.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const error = ref('')
const form = reactive({ email: '', password: '' })

async function submit() {
  loading.value = true
  error.value = ''
  try {
    const res = await authService.login({ email: form.email, password: form.password })
    const payload = res.data.data
    auth.setSession({
      accessToken: payload.access_token,
      refreshToken: payload.refresh_token,
      user: payload.user,
    })
    router.push({ name: auth.defaultHomeRoute })
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Login gagal'
  } finally {
    loading.value = false
  }
}
</script>
