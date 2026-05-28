<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h2 class="tenant-page-title">Tenant Users</h2>
        <p class="tenant-page-subtitle">Kelola member untuk {{ tenant.selectedMembership?.tenant_name || tenant.selectedTenantId || 'tenant aktif' }}.</p>
      </div>
      <span class="w-fit rounded-full border border-[var(--tenant-border)] bg-white/[0.04] px-3 py-1 text-sm text-[var(--tenant-text-secondary)]">
        {{ members.length }} members
      </span>
    </div>

    <UiAlert v-if="flashMessage" title="User invited" tone="success">
      <p>{{ flashMessage }}</p>
      <div v-if="tempPassword" class="mt-3 rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] bg-black/20 px-3 py-2">
        <p class="text-xs text-[var(--tenant-text-muted)]">Temporary password (show once):</p>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <code class="rounded bg-white/[0.06] px-2 py-1 text-xs text-[var(--tenant-text-primary)]">{{ tempPassword }}</code>
          <button class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] px-2 py-1 text-xs text-[var(--tenant-text-secondary)]" @click="copyTempPassword">
            {{ copied ? 'Copied' : 'Copy' }}
          </button>
          <button class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] px-2 py-1 text-xs text-[var(--tenant-text-secondary)]" @click="dismissTempPassword">
            Hide
          </button>
        </div>
      </div>
    </UiAlert>
    <UiAlert v-if="error" title="Failed to load tenant users" tone="danger">{{ error }}</UiAlert>
    <UiAlert v-if="!tenant.selectedTenantId" title="Tenant belum dipilih" tone="warning">Pilih tenant dulu dari switcher kanan atas untuk mengelola user.</UiAlert>

    <UserInviteForm v-if="canInvite" @invited="onInvited" />

    <AppCard v-if="tenant.selectedTenantId">
      <div class="overflow-x-auto">
        <table class="tenant-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Email</th>
              <th>Role</th>
              <th>Status</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading" v-for="i in 4" :key="`skeleton-${i}`">
              <td colspan="5"><div class="h-10 animate-pulse rounded-[var(--tenant-radius-button)] bg-white/[0.05]" /></td>
            </tr>
            <tr v-for="m in members" v-else :key="m.id">
              <td class="font-medium text-[var(--tenant-text-primary)]">{{ m.name }}</td>
              <td>{{ m.email }}</td>
              <td>
                <select
                  :value="m.role"
                  :disabled="!canUpdate || loading"
                  class="tenant-input max-w-[180px]"
                  @change="changeRole(m.id, ($event.target as HTMLSelectElement).value)"
                >
                  <option value="owner-tenant">owner-tenant</option>
                  <option value="admin">admin</option>
                  <option value="finance">finance</option>
                  <option value="support">support</option>
                </select>
              </td>
              <td>
                <span class="rounded-full border border-[var(--tenant-border)] px-2 py-1 text-xs" :class="m.is_active ? 'text-[var(--tenant-success)]' : 'text-[var(--tenant-text-muted)]'">
                  {{ m.is_active ? 'active' : 'inactive' }}
                </span>
              </td>
              <td>
                <button
                  v-if="canRemove"
                  :disabled="loading || !m.is_active"
                  class="rounded-[var(--tenant-radius-button)] border border-red-400/30 px-3 py-1.5 text-sm text-[var(--tenant-danger)] disabled:cursor-not-allowed disabled:opacity-50"
                  @click="removeMember(m.id)"
                >
                  Remove
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <UiEmptyState
        v-if="!loading && members.length === 0"
        class="mt-4"
        title="No members found"
        description="Invite first member or switch tenant."
      />
    </AppCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
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

const canInvite = computed(() => canTenant(tenant.selectedMembership, 'tenant.users.invite'))
const canUpdate = computed(() => canTenant(tenant.selectedMembership, 'tenant.users.update'))
const canRemove = computed(() => canTenant(tenant.selectedMembership, 'tenant.users.remove'))

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
