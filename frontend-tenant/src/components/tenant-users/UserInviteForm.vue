<template>
  <form class="space-y-4 rounded-[var(--tenant-radius-card)] border border-[var(--tenant-border)] bg-[var(--tenant-surface)] p-5" @submit.prevent="submit">
    <div>
      <h3 class="font-semibold text-[var(--tenant-text-primary)]">Invite member</h3>
      <p class="tenant-helper">Temporary password ditampilkan sekali. Simpan aman, lalu minta user ganti password setelah login.</p>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <label>
        <span class="tenant-label">Full name</span>
        <input v-model="form.name" required autocomplete="name" placeholder="Full name" class="tenant-input" />
      </label>
      <label>
        <span class="tenant-label">Email</span>
        <input v-model="form.email" required type="email" autocomplete="email" placeholder="work@email.com" class="tenant-input" />
      </label>
    </div>

    <label class="block max-w-xs">
      <span class="tenant-label">Role</span>
      <select v-model="form.role" class="tenant-input">
        <option v-for="role in roleOptions" :key="role.value" :value="role.value">{{ role.label }}</option>
      </select>
    </label>

    <UiAlert v-if="error" title="Invite failed" tone="danger">{{ error }}</UiAlert>

    <button :disabled="loading" class="rounded-[var(--tenant-radius-button)] bg-[var(--tenant-accent)] px-4 py-2 text-sm font-medium text-slate-950 hover:bg-[var(--tenant-accent-hover)] disabled:opacity-60">
      {{ loading ? 'Inviting...' : 'Invite' }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import UiAlert from '@/components/common/UiAlert.vue'
import { tenantUsersService } from '@/services/tenantUsers'
import { TENANT_ROLE_OPTIONS, type TenantRole } from '@/services/rbac'

const emit = defineEmits<{ invited: [payload: { message: string; temporaryPassword: string }] }>()
const props = withDefaults(defineProps<{ allowedRoles?: TenantRole[] }>(), {
  allowedRoles: () => ['admin', 'finance', 'support', 'manager', 'staff', 'viewer'],
})
const loading = ref(false)
const error = ref('')
const form = reactive({ name: '', email: '', role: 'support' })

const roleOptions = computed(() => TENANT_ROLE_OPTIONS.filter((role) => props.allowedRoles.includes(role.value)))

watch(
  roleOptions,
  (roles) => {
    if (!roles.some((role) => role.value === form.role)) form.role = roles[0]?.value || 'support'
  },
  { immediate: true },
)

async function submit() {
  loading.value = true
  error.value = ''
  try {
    const res = await tenantUsersService.invite({ ...form })
    const pwd = res.data.data.temporary_password
    emit('invited', {
      message: 'User invited successfully.',
      temporaryPassword: pwd,
    })
    form.name = ''
    form.email = ''
    form.role = 'support'
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal mengundang user'
    console.error('[user-invite] failed', e)
  } finally {
    loading.value = false
  }
}
</script>
