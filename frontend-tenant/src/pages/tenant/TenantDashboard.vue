<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h2 class="tenant-page-title">Tenant Dashboard</h2>
        <p class="tenant-page-subtitle">Workspace aktif: {{ tenantName }}.</p>
      </div>
      <span class="w-fit rounded-full border border-[var(--tenant-border)] bg-white/[0.04] px-3 py-1 text-sm text-[var(--tenant-text-secondary)]">
        Role: {{ tenant.selectedMembership?.role || '-' }}
      </span>
    </div>

    <div class="grid gap-3 sm:grid-cols-3">
      <KpiCard label="Tenant aktif" :value="tenant.selectedTenantId ? 1 : 0" :hint="tenant.selectedTenantId ? 'selected' : 'none'" />
      <KpiCard label="Memberships" :value="tenant.memberships.length" hint="available" />
      <KpiCard label="Permissions" :value="permissions.length" hint="role grants" />
    </div>

    <div class="grid gap-5 xl:grid-cols-[1fr_320px]">
      <AppCard>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-semibold text-[var(--tenant-text-primary)]">Workspace summary</h3>
            <p class="mt-1 text-sm text-[var(--tenant-text-muted)]">Ringkasan tenant terpilih dari membership user.</p>
          </div>
        </div>

        <dl class="mt-5 grid gap-3 sm:grid-cols-2">
          <div class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] bg-white/[0.03] p-4">
            <dt class="text-xs uppercase tracking-wide text-[var(--tenant-text-muted)]">Tenant</dt>
            <dd class="mt-2 text-sm font-medium text-[var(--tenant-text-primary)]">{{ tenantName }}</dd>
          </div>
          <div class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] bg-white/[0.03] p-4">
            <dt class="text-xs uppercase tracking-wide text-[var(--tenant-text-muted)]">Tenant ID</dt>
            <dd class="mt-2 break-all text-sm font-medium text-[var(--tenant-text-primary)]">{{ tenant.selectedTenantId || '-' }}</dd>
          </div>
        </dl>
      </AppCard>

      <AppCard>
        <h3 class="text-base font-semibold text-[var(--tenant-text-primary)]">Quick actions</h3>
        <div class="mt-4 grid gap-2">
          <RouterLink
            v-if="canUsers"
            class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] bg-white/[0.03] px-3 py-2 text-sm text-[var(--tenant-text-secondary)] hover:bg-white/[0.06]"
            to="/tenant/users"
          >
            Manage users
          </RouterLink>
          <RouterLink
            v-if="canSettings"
            class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] bg-white/[0.03] px-3 py-2 text-sm text-[var(--tenant-text-secondary)] hover:bg-white/[0.06]"
            to="/tenant/settings"
          >
            Open settings
          </RouterLink>
          <RouterLink
            v-if="canAudit"
            class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] bg-white/[0.03] px-3 py-2 text-sm text-[var(--tenant-text-secondary)] hover:bg-white/[0.06]"
            to="/tenant/audit"
          >
            View audit log
          </RouterLink>
          <UiEmptyState
            v-if="!canUsers && !canSettings && !canAudit"
            title="No quick actions"
            description="Role saat ini hanya punya akses dashboard."
          />
        </div>
      </AppCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AppCard from '@/components/common/AppCard.vue'
import KpiCard from '@/components/common/KpiCard.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
import { canTenant, getTenantPermissions } from '@/services/rbac'
import { useTenantStore } from '@/stores/tenant'

const tenant = useTenantStore()
const tenantName = computed(() => tenant.selectedMembership?.tenant_name || tenant.selectedTenantId || 'belum dipilih')
const permissions = computed(() => getTenantPermissions(tenant.selectedMembership))
const canUsers = computed(() => canTenant(tenant.selectedMembership, 'tenant.users.read'))
const canSettings = computed(() => canTenant(tenant.selectedMembership, 'tenant.settings.read'))
const canAudit = computed(() => canTenant(tenant.selectedMembership, 'tenant.audit.read'))
</script>
