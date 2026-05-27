<template>
  <form class="space-y-3 rounded-xl border border-slate-200 bg-white p-4" @submit.prevent="submit">
    <h3 class="font-semibold text-slate-900">Invite member</h3>

    <div class="grid gap-3 md:grid-cols-2">
      <input v-model="form.name" required placeholder="Full name" class="rounded-md border border-slate-300 px-3 py-2 text-sm" />
      <input v-model="form.email" required type="email" placeholder="Email" class="rounded-md border border-slate-300 px-3 py-2 text-sm" />
    </div>

    <select v-model="form.role" class="rounded-md border border-slate-300 px-3 py-2 text-sm">
      <option value="admin">admin</option>
      <option value="finance">finance</option>
      <option value="support">support</option>
      <option value="owner-tenant">owner-tenant</option>
    </select>

    <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

    <button :disabled="loading" class="rounded-md bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-60">
      {{ loading ? 'Inviting...' : 'Invite' }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { tenantUsersService } from '@/services/tenantUsers'

const emit = defineEmits<{ invited: [msg: string] }>()
const loading = ref(false)
const error = ref('')
const form = reactive({ name: '', email: '', role: 'support' })

async function submit() {
  loading.value = true
  error.value = ''
  try {
    const res = await tenantUsersService.invite({ ...form })
    const pwd = res.data.data.temporary_password
    emit('invited', `User invited. Temporary password: ${pwd}`)
    form.name = ''
    form.email = ''
    form.role = 'support'
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Invite failed'
  } finally {
    loading.value = false
  }
}
</script>
