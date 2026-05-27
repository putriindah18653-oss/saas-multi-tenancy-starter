<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-semibold text-slate-900">Tenant Users</h2>
      <span class="text-xs text-slate-500">Tenant: {{ tenant.selectedMembership?.tenant_name || tenant.selectedTenantId || '-' }}</span>
    </div>

    <div v-if="flashMessage" class="space-y-2 rounded-md bg-emerald-50 px-3 py-3 text-sm text-emerald-800">
      <p>{{ flashMessage }}</p>
      <div v-if="tempPassword" class="rounded border border-emerald-200 bg-white px-3 py-2">
        <p class="text-xs text-slate-600">Temporary password (show once):</p>
        <div class="mt-1 flex flex-wrap items-center gap-2">
          <code class="rounded bg-slate-100 px-2 py-1 text-xs text-slate-900">{{ tempPassword }}</code>
          <button class="rounded border border-slate-300 px-2 py-1 text-xs text-slate-700" @click="copyTempPassword">
            {{ copied ? 'Copied' : 'Copy' }}
          </button>
          <button class="rounded border border-slate-300 px-2 py-1 text-xs text-slate-700" @click="dismissTempPassword">
            Hide
          </button>
        </div>
      </div>
    </div>
    <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

    <UserInviteForm v-if="canInvite" @invited="onInvited" />

    <div v-if="!tenant.selectedTenantId" class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
      Pilih tenant dulu dari switcher di kanan atas untuk mengelola user tenant.
    </div>

    <div v-if="tenant.selectedTenantId" class="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <table class="min-w-full text-sm">
        <thead class="bg-slate-50 text-left text-slate-600">
          <tr>
            <th class="px-4 py-3">Name</th>
            <th class="px-4 py-3">Email</th>
            <th class="px-4 py-3">Role</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in members" :key="m.id" class="border-t border-slate-100">
            <td class="px-4 py-3">{{ m.name }}</td>
            <td class="px-4 py-3">{{ m.email }}</td>
            <td class="px-4 py-3">
              <select
                :value="m.role"
                :disabled="!canUpdate || loading"
                class="rounded border border-slate-300 px-2 py-1"
                @change="changeRole(m.id, ($event.target as HTMLSelectElement).value)"
              >
                <option value="owner-tenant">owner-tenant</option>
                <option value="admin">admin</option>
                <option value="finance">finance</option>
                <option value="support">support</option>
              </select>
            </td>
            <td class="px-4 py-3">{{ m.is_active ? 'active' : 'inactive' }}</td>
            <td class="px-4 py-3">
              <button
                v-if="canRemove"
                :disabled="loading || !m.is_active"
                class="rounded border border-red-300 px-2 py-1 text-red-700 disabled:opacity-50"
                @click="removeMember(m.id)"
              >
                Remove
              </button>
            </td>
          </tr>
          <tr v-if="!loading && members.length === 0">
            <td colspan="5" class="px-4 py-6 text-center text-slate-500">No members found.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useTenantStore } from '@/stores/tenant'
import { tenantUsersService, type TenantMember } from '@/services/tenantUsers'
import { canTenant } from '@/services/rbac'
import UserInviteForm from '@/components/tenant-users/UserInviteForm.vue'

const tenant = useTenantStore()
const members = ref<TenantMember[]>([])
const loading = ref(false)
const error = ref('')
const flashMessage = ref('')
const tempPassword = ref('')
const copied = ref(false)

const canInvite = computed(() => tenant.selectedMembership?.role === 'owner-tenant' && canTenant(tenant.selectedMembership, 'tenant.users.invite'))
const canUpdate = computed(() => canTenant(tenant.selectedMembership, 'tenant.users.update'))
const canRemove = computed(() => canTenant(tenant.selectedMembership, 'tenant.users.remove'))
const canManage = computed(() => canInvite.value || canUpdate.value || canRemove.value)

async function load() {
  if (!tenant.selectedTenantId) {
    members.value = []
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await tenantUsersService.list()
    members.value = res.data.data || []
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to load members'
  } finally {
    loading.value = false
  }
}

async function changeRole(id: string, role: string) {
  if (!canUpdate.value) return
  loading.value = true
  error.value = ''
  try {
    await tenantUsersService.changeRole(id, role)
    await load()
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to change role'
  } finally {
    loading.value = false
  }
}

async function removeMember(id: string) {
  if (!canRemove.value) return
  if (!confirm('Remove this member from tenant?')) return
  loading.value = true
  error.value = ''
  try {
    await tenantUsersService.remove(id)
    await load()
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to remove member'
  } finally {
    loading.value = false
  }
}

function onInvited(payload: { message: string; temporaryPassword: string }) {
  flashMessage.value = payload.message
  tempPassword.value = payload.temporaryPassword
  copied.value = false
  load()
}

async function copyTempPassword() {
  if (!tempPassword.value) return
  try {
    await navigator.clipboard.writeText(tempPassword.value)
    copied.value = true
  } catch {
    copied.value = false
  }
}

function dismissTempPassword() {
  tempPassword.value = ''
  copied.value = false
}

watch(
  () => tenant.selectedTenantId,
  () => {
    members.value = []
    error.value = ''
    flashMessage.value = ''
    tempPassword.value = ''
    copied.value = false
    load()
  },
)

onMounted(load)
</script>
