<template>
  <div>
    <p class="text-sm font-medium text-[var(--warning)]">Action required</p>
    <h2 class="mt-2 text-2xl font-semibold tracking-tight text-[var(--text-primary)]">Change password</h2>
    <p class="mt-2 text-sm text-[var(--text-muted)]">Password harus diganti sebelum lanjut memakai dashboard.</p>

    <form class="mt-7 space-y-5" @submit.prevent="submit">
      <div>
        <label for="current-password" class="owner-label">Current password</label>
        <input
          id="current-password"
          v-model="form.currentPassword"
          type="password"
          autocomplete="current-password"
          required
          class="owner-input"
        />
      </div>
      <div>
        <label for="new-password" class="owner-label">New password</label>
        <input
          id="new-password"
          v-model="form.newPassword"
          type="password"
          autocomplete="new-password"
          required
          minlength="12"
          maxlength="72"
          class="owner-input"
          aria-describedby="password-help"
        />
        <p id="password-help" class="owner-helper">Min 12 karakter, ada huruf besar, huruf kecil, angka, simbol. Maks 72 byte.</p>
      </div>
      <div>
        <label for="confirm-password" class="owner-label">Confirm new password</label>
        <input
          id="confirm-password"
          v-model="form.confirmPassword"
          type="password"
          autocomplete="new-password"
          required
          class="owner-input"
        />
      </div>

      <UiAlert v-if="error" title="Gagal mengganti password" tone="danger">{{ error }}</UiAlert>
      <UiAlert v-if="success" title="Password updated" tone="success">Redirecting to dashboard.</UiAlert>

      <AppButton type="submit" :disabled="loading" class="w-full">
        {{ loading ? 'Saving...' : 'Change password' }}
      </AppButton>
    </form>
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
const success = ref(false)
const form = reactive({ currentPassword: '', newPassword: '', confirmPassword: '' })

async function submit() {
  error.value = ''
  success.value = false
  if (form.newPassword !== form.confirmPassword) {
    error.value = 'Konfirmasi password tidak sama'
    return
  }
  loading.value = true
  try {
    await authService.changePassword({ current_password: form.currentPassword, new_password: form.newPassword })
    if (auth.accessToken && auth.user) {
      auth.setSession({ accessToken: auth.accessToken, refreshToken: auth.refreshToken || undefined, user: { ...auth.user, must_change_password: false } })
    }
    success.value = true
    router.push({ name: auth.defaultHomeRoute })
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal mengganti password'
    console.error('[change-password] failed', e)
  } finally {
    loading.value = false
  }
}
</script>
