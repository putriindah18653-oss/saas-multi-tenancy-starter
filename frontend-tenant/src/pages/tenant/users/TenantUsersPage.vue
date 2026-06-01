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

    <UserInviteForm v-if="canInvite" :allowed-roles="assignableRoles" @invited="onInvited" />

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
                  :disabled="!canChangeMemberRole(m)"
                  class="tenant-input max-w-[180px]"
                  @change="changeRole(m.id, ($event.target as HTMLSelectElement).value)"
                >
                  <option v-for="role in memberRoleOptions(m)" :key="role.value" :value="role.value">{{ role.label }}</option>
                </select>
                <p v-if="isSelf(m)" class="mt-1 text-xs text-[var(--tenant-text-muted)]">Own role protected.</p>
                <p v-else-if="isLastActiveOwner(m)" class="mt-1 text-xs text-[var(--tenant-text-muted)]">Last owner protected.</p>
              </td>
              <td>
                <span class="rounded-full border border-[var(--tenant-border)] px-2 py-1 text-xs" :class="m.is_active ? 'text-[var(--tenant-success)]' : 'text-[var(--tenant-text-muted)]'">
                  {{ m.is_active ? 'active' : 'inactive' }}
                </span>
              </td>
              <td>
                <button
                  v-if="canRemove"
                  :disabled="!canRemoveMember(m)"
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
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'
import { tenantUsersService, type TenantMember } from '@/services/tenantUsers'
import { canTenant, isTenantOwnerRole, TENANT_ROLE_OPTIONS, type TenantRole } from '@/services/rbac'
import UserInviteForm from '@/components/tenant-users/UserInviteForm.vue'

const auth = useAuthStore()
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
const isCurrentUserOwner = computed(() => isTenantOwnerRole(tenant.selectedMembership?.role))
const activeOwnerCount = computed(() => members.value.filter((member) => member.is_active && isTenantOwnerRole(member.role)).length)
const assignableRoles = computed<TenantRole[]>(() => {
  const roles = TENANT_ROLE_OPTIONS.map((role) => role.value)
  return isCurrentUserOwner.value ? roles : roles.filter((role) => !isTenantOwnerRole(role))
})

function isSelf(member: TenantMember) {
  return member.user_id === auth.user?.id
}

function isLastActiveOwner(member: TenantMember) {
  return member.is_active && isTenantOwnerRole(member.role) && activeOwnerCount.value <= 1
}

function canAssignRole(role: string) {
  return assignableRoles.value.includes(role as TenantRole)
}

function canChangeMemberRole(member: TenantMember) {
  return canUpdate.value && !loading.value && member.is_active && !isSelf(member) && !isLastActiveOwner(member)
}

function canRemoveMember(member: TenantMember) {
  return canRemove.value && !loading.value && member.is_active && !isSelf(member) && !isLastActiveOwner(member)
}

function memberRoleOptions(member: TenantMember) {
  const values = new Set<TenantRole>(assignableRoles.value)
  if (isTenantOwnerRole(member.role) && !isCurrentUserOwner.value) values.add(member.role as TenantRole)
  return TENANT_ROLE_OPTIONS.filter((role) => values.has(role.value))
}

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
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal memuat members'
    console.error('[tenant-users] load failed', e)
  } finally {
    loading.value = false
  }
}

async function changeRole(id: string, role: string) {
  if (!canUpdate.value) return
  const member = members.value.find((item) => item.id === id)
  if (!member) return
  if (member.role === role) return
  if (!canChangeMemberRole(member)) {
    error.value = 'This member role is protected.'
    return
  }
  if (!canAssignRole(role)) {
    error.value = 'Role is not allowed for your access level.'
    return
  }
  if (isTenantOwnerRole(member.role) && !isTenantOwnerRole(role) && isLastActiveOwner(member)) {
    error.value = 'At least one active tenant owner is required.'
    return
  }
  if (!confirm(`Change ${member.email} role from ${member.role} to ${role}?`)) return
  loading.value = true
  error.value = ''
  try {
    await tenantUsersService.changeRole(id, role)
    await load()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal mengubah role'
    console.error('[tenant-users] changeRole failed', e)
  } finally {
    loading.value = false
  }
}

async function removeMember(id: string) {
  if (!canRemove.value) return
  const member = members.value.find((item) => item.id === id)
  if (!member) return
  if (!canRemoveMember(member)) {
    error.value = isSelf(member) ? 'You cannot remove your own tenant access.' : 'This member is protected.'
    return
  }
  if (!confirm(`Remove ${member.email} from tenant?`)) return
  loading.value = true
  error.value = ''
  try {
    await tenantUsersService.remove(id)
    await load()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal menghapus member'
    console.error('[tenant-users] removeMember failed', e)
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
