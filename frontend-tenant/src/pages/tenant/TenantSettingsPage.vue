<template>
  <div class="space-y-6">
    <div>
      <h2 class="tenant-page-title">Tenant Settings</h2>
      <p class="tenant-page-subtitle">Branding dan preferensi tenant aktif.</p>
    </div>

    <UiAlert v-if="error" title="Gagal memuat settings" tone="danger">{{ error }}</UiAlert>
    <UiAlert v-if="saved" title="Settings tersimpan" tone="success">Perubahan tenant berhasil disimpan.</UiAlert>

    <AppCard>
      <form class="grid gap-4" @submit.prevent="save">
        <label>
          <span class="tenant-label">Display name</span>
          <input v-model="form.display_name" class="tenant-input" />
        </label>
        <label>
          <span class="tenant-label">Logo URL</span>
          <input v-model="form.logo_url" class="tenant-input" placeholder="https://..." />
        </label>
        <div class="grid gap-4 md:grid-cols-3">
          <label>
            <span class="tenant-label">Timezone</span>
            <input v-model="form.timezone" class="tenant-input" />
          </label>
          <label>
            <span class="tenant-label">Locale</span>
            <input v-model="form.locale" class="tenant-input" />
          </label>
          <label>
            <span class="tenant-label">Currency</span>
            <input v-model="form.currency" class="tenant-input" />
          </label>
        </div>
        <button class="w-fit rounded-[var(--tenant-radius-button)] bg-[var(--tenant-accent)] px-4 py-2 text-sm font-medium text-slate-950 hover:bg-[var(--tenant-accent-hover)] disabled:opacity-60" :disabled="loading">
          {{ loading ? 'Menyimpan...' : 'Simpan settings' }}
        </button>
      </form>
    </AppCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import { tenantSettingsService, type TenantSettings } from '@/services/tenantSettings'

const loading = ref(false)
const error = ref('')
const saved = ref(false)
const form = reactive<Partial<TenantSettings>>({ display_name: '', logo_url: '', timezone: 'Asia/Jakarta', locale: 'id-ID', currency: 'IDR' })

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await tenantSettingsService.get()
    Object.assign(form, data.data)
  } catch {
    error.value = 'Gagal memuat settings tenant.'
  } finally {
    loading.value = false
  }
}
async function save() {
  loading.value = true
  error.value = ''
  saved.value = false
  try {
    const { data } = await tenantSettingsService.update(form)
    Object.assign(form, data.data)
    saved.value = true
  } catch {
    error.value = 'Gagal menyimpan settings tenant.'
  } finally {
    loading.value = false
  }
}
onMounted(load)
</script>
