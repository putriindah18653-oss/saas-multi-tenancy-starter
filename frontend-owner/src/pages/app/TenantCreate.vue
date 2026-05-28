<template>
  <div class="max-w-2xl space-y-6">
    <div>
      <h2 class="owner-page-title">Create tenant</h2>
      <p class="owner-page-subtitle">Buat workspace tenant baru tanpa mengubah kontrak API.</p>
    </div>

    <AppCard>
      <form class="space-y-5" @submit.prevent="submit">
        <div>
          <label for="tenant-name" class="owner-label">Name</label>
          <input id="tenant-name" v-model="form.name" required class="owner-input" placeholder="Acme Corp" />
        </div>

        <div>
          <label for="tenant-slug" class="owner-label">Slug (optional)</label>
          <input id="tenant-slug" v-model="form.slug" class="owner-input" placeholder="acme-corp" />
          <p class="owner-helper">Kosongkan untuk mengikuti aturan backend.</p>
        </div>

        <UiAlert v-if="error" title="Failed to create tenant" tone="danger">{{ error }}</UiAlert>

        <div class="flex flex-wrap gap-2">
          <AppButton type="submit" :disabled="loading">
            {{ loading ? 'Saving...' : 'Save tenant' }}
          </AppButton>
          <RouterLink to="/app/tenants" class="inline-flex min-h-10 items-center rounded-[var(--radius-button)] border border-[var(--border-strong)] px-4 py-2 text-sm text-[var(--text-secondary)] hover:bg-white/5">
            Cancel
          </RouterLink>
        </div>
      </form>
    </AppCard>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppButton from '@/components/common/AppButton.vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
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
