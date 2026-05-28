<template>
  <select
    :value="tenant.selectedTenantId || ''"
    :disabled="!tenant.memberships.length"
    aria-label="Switch tenant"
    class="min-h-10 rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] bg-[var(--tenant-surface)] px-3 py-2 text-sm text-[var(--tenant-text-primary)] outline-none transition hover:bg-[var(--tenant-surface-elevated)] focus:border-[var(--tenant-accent)] focus:ring-2 focus:ring-[var(--tenant-focus-ring)]"
    @change="onChange"
  >
    <option value="" disabled>{{ tenant.memberships.length ? 'Select tenant' : 'No tenant' }}</option>
    <option v-for="m in tenant.memberships" :key="m.tenant_id" :value="m.tenant_id">
      {{ m.tenant_name || m.tenant_id }} ({{ m.role }})
    </option>
  </select>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { canTenant, type TenantPermission } from '@/services/rbac'
import { useTenantStore } from '@/stores/tenant'

const tenant = useTenantStore()
const route = useRoute()
const router = useRouter()

const firstAllowedTenantRoute = computed(() => {
  const membership = tenant.selectedMembership
  if (canTenant(membership, 'tenant.dashboard.read')) return { name: 'tenant-home' }
  if (canTenant(membership, 'tenant.users.read')) return { name: 'tenant-users' }
  if (canTenant(membership, 'tenant.settings.read')) return { name: 'tenant-settings' }
  if (canTenant(membership, 'tenant.audit.read')) return { name: 'tenant-audit' }
  return { name: 'forbidden' }
})

function currentRoutePermission(): TenantPermission | null {
  return ([...route.matched].reverse().find((record) => record.meta.permission)?.meta.permission as TenantPermission | undefined) ?? null
}

function onChange(event: Event) {
  const select = event.target as HTMLSelectElement
  const value = select.value

  if (!value || !tenant.selectTenant(value)) {
    select.value = tenant.selectedTenantId || ''
    return
  }

  const permission = currentRoutePermission()
  if (route.meta.scope === 'tenant' && permission && !canTenant(tenant.selectedMembership, permission)) {
    router.replace(firstAllowedTenantRoute.value)
  }
}
</script>
