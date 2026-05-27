<template>
  <div class="max-w-2xl space-y-4">
    <h2 class="text-xl font-semibold text-slate-900">Create Tenant</h2>

    <form class="space-y-4 rounded-xl border border-slate-200 bg-white p-4" @submit.prevent="submit">
      <div>
        <label class="mb-1 block text-sm text-slate-700">Name</label>
        <input v-model="form.name" required class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>

      <div>
        <label class="mb-1 block text-sm text-slate-700">Slug (optional)</label>
        <input v-model="form.slug" class="w-full rounded-md border border-slate-300 px-3 py-2" />
      </div>

      <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

      <div class="flex gap-2">
        <button :disabled="loading" class="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60">
          {{ loading ? 'Saving...' : 'Save' }}
        </button>
        <RouterLink to="/app/tenants" class="rounded-md border border-slate-300 px-4 py-2 text-sm">Cancel</RouterLink>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { tenantsService } from '@/services/tenants'

const router = useRouter()
const loading = ref(false)
const error = ref('')
const form = reactive({ name: '', slug: '' })

async function submit() {
  loading.value = true
  error.value = ''
  try {
    const res = await tenantsService.create({ name: form.name, slug: form.slug || undefined })
    router.push(`/app/tenants/${res.data.data.id}`)
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Failed to create tenant'
  } finally {
    loading.value = false
  }
}
</script>
