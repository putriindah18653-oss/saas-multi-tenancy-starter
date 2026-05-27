<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-semibold text-slate-900">Tenants</h2>
      <RouterLink
        v-if="canCreate"
        to="/app/tenants/create"
        class="rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white"
      >
        Create tenant
      </RouterLink>
    </div>

    <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

    <div class="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <table class="min-w-full text-sm">
        <thead class="bg-slate-50 text-left text-slate-600">
          <tr>
            <th class="px-4 py-3">Name</th>
            <th class="px-4 py-3">Slug</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in tenants" :key="t.id" class="border-t border-slate-100">
            <td class="px-4 py-3">{{ t.name }}</td>
            <td class="px-4 py-3">{{ t.slug }}</td>
            <td class="px-4 py-3">{{ t.status }}</td>
            <td class="px-4 py-3">
              <RouterLink class="text-slate-900 underline" :to="`/app/tenants/${t.id}`">Detail</RouterLink>
            </td>
          </tr>
          <tr v-if="!loading && tenants.length === 0">
            <td colspan="4" class="px-4 py-6 text-center text-slate-500">No tenants found.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { tenantsService, type Tenant } from '@/services/tenants'
import { useAuthStore } from '@/stores/auth'
import { canOwner } from '@/services/rbac'

const auth = useAuthStore()
const tenants = ref<Tenant[]>([])
const loading = ref(false)
const error = ref('')

const canCreate = computed(() => canOwner(auth.user, 'app.tenants.create'))

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
