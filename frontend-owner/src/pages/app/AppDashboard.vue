<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h2 class="owner-page-title">App Dashboard</h2>
        <p class="owner-page-subtitle">Selamat datang, {{ auth.user?.name || auth.user?.full_name || auth.user?.email || 'user' }}.</p>
      </div>
      <RouterLink
        v-if="canCreate"
        to="/app/tenants/create"
        class="inline-flex min-h-10 items-center justify-center rounded-[var(--radius-button)] bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--accent-hover)]"
      >
        Create tenant
      </RouterLink>
    </div>

    <UiAlert v-if="error" title="Failed to load dashboard" tone="danger">{{ error }}</UiAlert>

    <div class="grid gap-3 sm:grid-cols-3">
      <KpiCard label="Total tenants" :value="loading ? '—' : tenants.length" hint="All records" />
      <KpiCard label="Active tenants" :value="loading ? '—' : activeTenants" hint="status active" />
      <KpiCard label="Inactive tenants" :value="loading ? '—' : inactiveTenants" hint="needs review" />
    </div>

    <div class="grid gap-5 xl:grid-cols-[1fr_320px]">
      <AppCard>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-semibold text-[var(--text-primary)]">Recent tenants</h3>
            <p class="mt-1 text-sm text-[var(--text-muted)]">Snapshot tenant terbaru dari owner API.</p>
          </div>
          <RouterLink to="/app/tenants" class="text-sm font-medium text-[var(--accent)] hover:text-[var(--accent-hover)]">View all</RouterLink>
        </div>

        <div v-if="loading" class="mt-5 space-y-3">
          <div v-for="i in 4" :key="i" class="h-12 animate-pulse rounded-[var(--radius-button)] bg-white/[0.05]" />
        </div>

        <UiEmptyState
          v-else-if="tenants.length === 0"
          class="mt-5"
          title="No tenants yet"
          description="Create first tenant to start operating platform workspace."
        >
          <template #actions>
            <RouterLink v-if="canCreate" to="/app/tenants/create" class="rounded-[var(--radius-button)] bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white">
              Create tenant
            </RouterLink>
          </template>
        </UiEmptyState>

        <div v-else class="mt-5 overflow-x-auto">
          <table class="owner-table">
            <thead><tr><th>Name</th><th>Slug</th><th>Status</th></tr></thead>
            <tbody>
              <tr v-for="tenant in tenants.slice(0, 5)" :key="tenant.id">
                <td class="font-medium text-[var(--text-primary)]">{{ tenant.name }}</td>
                <td>{{ tenant.slug }}</td>
                <td><span class="rounded-full border border-[var(--border)] px-2 py-1 text-xs">{{ tenant.status }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </AppCard>

      <AppCard>
        <h3 class="text-base font-semibold text-[var(--text-primary)]">Quick actions</h3>
        <div class="mt-4 grid gap-2">
          <RouterLink class="rounded-[var(--radius-button)] border border-[var(--border)] bg-white/[0.03] px-3 py-2 text-sm text-[var(--text-secondary)] hover:bg-white/[0.06]" to="/app/tenants">
            Manage tenants
          </RouterLink>
          <RouterLink class="rounded-[var(--radius-button)] border border-[var(--border)] bg-white/[0.03] px-3 py-2 text-sm text-[var(--text-secondary)] hover:bg-white/[0.06]" to="/app/audit">
            Open audit log
          </RouterLink>
        </div>
      </AppCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppCard from '@/components/common/AppCard.vue'
import KpiCard from '@/components/common/KpiCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
import { tenantsService, type Tenant } from '@/services/tenants'
import { canOwner } from '@/services/rbac'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const tenants = ref<Tenant[]>([])
const loading = ref(false)
const error = ref('')

const canCreate = computed(() => canOwner(auth.user, 'app.tenants.create'))
const activeTenants = computed(() => tenants.value.filter((tenant) => tenant.status === 'active').length)
const inactiveTenants = computed(() => tenants.value.filter((tenant) => tenant.status !== 'active').length)

onMounted(async () => {
  loading.value = true
  error.value = ''
  try {
    tenants.value = (await tenantsService.list()).data.data || []
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to load tenants'
  } finally {
    loading.value = false
  }
})
</script>
