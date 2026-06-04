<template>
  <div>
    <h2 class="text-2xl font-semibold text-slate-900">Change password</h2>
    <p class="mt-1 text-sm text-slate-600">Password harus diganti sebelum lanjut memakai dashboard tenant.</p>

    <form class="mt-6 space-y-4" @submit.prevent="submit">
      <div>
        <label class="mb-1 block text-sm text-slate-700">Current password</label>
        <input v-model="form.currentPassword" type="password" autocomplete="current-password" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>
      <div>
        <label class="mb-1 block text-sm text-slate-700">New password</label>
        <input v-model="form.newPassword" type="password" autocomplete="new-password" required minlength="12" maxlength="72" class="w-full rounded-md border border-slate-300 px-3 py-2" />
        <p class="mt-1 text-xs text-slate-500">Min 12 karakter, ada huruf besar, huruf kecil, angka, simbol. Maks 72 byte.</p>
      </div>
      <div>
        <label class="mb-1 block text-sm text-slate-700">Confirm new password</label>
        <input v-model="form.confirmPassword" type="password" autocomplete="new-password" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>

      <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
      <p v-if="success" class="text-sm text-emerald-700">Password updated.</p>

      <button :disabled="loading" class="w-full rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60">
        {{ loading ? 'Saving...' : 'Change password' }}
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
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
      auth.setSession({ accessToken: auth.accessToken, user: { ...auth.user, must_change_password: false } })
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
