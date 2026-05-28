<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h2 class="owner-page-title">Tenants</h2>
        <p class="owner-page-subtitle">Kelola tenant platform dan status operasionalnya.</p>
      </div>
      <RouterLink
        v-if="canCreate"
        to="/app/tenants/create"
        class="inline-flex min-h-10 items-center justify-center rounded-[var(--radius-button)] bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--accent-hover)]"
      >
        Create tenant
      </RouterLink>
    </div>

    <UiAlert v-if="error" title="Failed to load tenants" tone="danger">{{ error }}</UiAlert>

    <AppCard>
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 5" :key="i" class="h-12 animate-pulse rounded-[var(--radius-button)] bg-white/[0.05]" />
      </div>

      <UiEmptyState
        v-else-if="tenants.length === 0"
        title="No tenants found"
        description="Create a tenant when your first customer workspace is ready."
      >
        <template #actions>
          <RouterLink v-if="canCreate" to="/app/tenants/create" class="rounded-[var(--radius-button)] bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white">
            Create tenant
          </RouterLink>
        </template>
      </UiEmptyState>

      <div v-else class="overflow-x-auto">
        <table class="owner-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Slug</th>
              <th>Status</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in tenants" :key="t.id">
              <td class="font-medium text-[var(--text-primary)]">{{ t.name }}</td>
              <td>{{ t.slug }}</td>
              <td>
                <span class="rounded-full border px-2 py-1 text-xs" :class="statusClass(t.status)">{{ t.status }}</span>
              </td>
              <td>
                <RouterLink class="font-medium text-[var(--accent)] hover:text-[var(--accent-hover)]" :to="`/app/tenants/${t.id}`">Detail</RouterLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </AppCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
import { tenantsService, type Tenant } from '@/services/tenants'
import { useAuthStore } from '@/stores/auth'
import { canOwner } from '@/services/rbac'

const auth = useAuthStore()
const tenants = ref<Tenant[]>([])
const loading = ref(false)
const error = ref('')

const canCreate = computed(() => canOwner(auth.user, 'app.tenants.create'))

function statusClass(status: string) {
  if (status === 'active') return 'border-emerald-400/25 bg-emerald-400/10 text-[var(--success)]'
  if (status === 'deleted') return 'border-red-400/25 bg-red-400/10 text-[var(--danger)]'
  return 'border-amber-400/25 bg-amber-400/10 text-[var(--warning)]'
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await tenantsService.list()
    tenants.value = res.data.data || []
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to load tenants'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
