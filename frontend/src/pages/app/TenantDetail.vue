<template>
  <div class="max-w-3xl space-y-4">
    <h2 class="text-xl font-semibold text-slate-900">Tenant Detail</h2>

    <div v-if="error" class="text-sm text-red-600">{{ error }}</div>

    <form v-if="tenant" class="space-y-4 rounded-xl border border-slate-200 bg-white p-4" @submit.prevent="save">
      <div>
        <label class="mb-1 block text-sm text-slate-700">Name</label>
        <input v-model="form.name" class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>

      <div>
        <label class="mb-1 block text-sm text-slate-700">Slug</label>
        <input :value="tenant.slug" disabled class="w-full rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-slate-500" />
      </div>

      <div>
        <label class="mb-1 block text-sm text-slate-700">Status</label>
        <select v-model="form.status" class="w-full rounded-md border border-slate-300 px-3 py-2">
          <option value="active">active</option>
          <option value="inactive">inactive</option>
          <option value="deleted">deleted</option>
        </select>
      </div>

      <div class="flex flex-wrap gap-2">
        <button :disabled="loading || !canManage" class="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60">
          {{ loading ? 'Saving...' : 'Save changes' }}
        </button>
        <button type="button" :disabled="loading || !canManage" class="rounded-md border border-red-300 px-4 py-2 text-sm text-red-700 disabled:opacity-60" @click="removeTenant">
          Soft delete
        </button>
        <RouterLink to="/app/tenants" class="rounded-md border border-slate-300 px-4 py-2 text-sm">Back</RouterLink>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { tenantsService, type Tenant } from '@/services/tenants'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const id = computed(() => String(route.params.id || ''))
const tenant = ref<Tenant | null>(null)
const loading = ref(false)
const error = ref('')
const form = reactive({ name: '', status: 'active' })

const canManage = computed(() => auth.user?.app_role === 'owner-app' || auth.user?.app_role === 'admin')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await tenantsService.get(id.value)
    tenant.value = res.data.data
    form.name = tenant.value?.name || ''
    form.status = tenant.value?.status || 'active'
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to load tenant'
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!canManage.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await tenantsService.update(id.value, { name: form.name, status: form.status })
    tenant.value = res.data.data
    form.name = tenant.value.name
    form.status = tenant.value.status
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to update tenant'
  } finally {
    loading.value = false
  }
}

async function removeTenant() {
  if (!canManage.value) return
  if (!confirm('Delete this tenant?')) return
  loading.value = true
  error.value = ''
  try {
    await tenantsService.remove(id.value)
    router.push('/app/tenants')
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to delete tenant'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
