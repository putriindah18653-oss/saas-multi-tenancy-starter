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
          <input v-model.trim="form.display_name" class="tenant-input" maxlength="120" />
        </label>
        <label>
          <span class="tenant-label">Logo URL</span>
          <input v-model.trim="form.logo_url" class="tenant-input" placeholder="https://..." inputmode="url" />
          <span class="tenant-helper">Opsional. Harus URL http/https valid.</span>
        </label>
        <div class="grid gap-4 md:grid-cols-3">
          <label>
            <span class="tenant-label">Timezone</span>
            <select v-model="form.timezone" class="tenant-input">
              <option v-for="timezone in timezoneOptions" :key="timezone" :value="timezone">{{ timezone }}</option>
            </select>
          </label>
          <label>
            <span class="tenant-label">Locale</span>
            <select v-model="form.locale" class="tenant-input">
              <option v-for="locale in localeOptions" :key="locale" :value="locale">{{ locale }}</option>
            </select>
          </label>
          <label>
            <span class="tenant-label">Currency</span>
            <select v-model="form.currency" class="tenant-input">
              <option v-for="currency in currencyOptions" :key="currency" :value="currency">{{ currency }}</option>
            </select>
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

const timezoneOptions = ['Asia/Jakarta', 'Asia/Makassar', 'Asia/Jayapura', 'UTC']
const localeOptions = ['id-ID', 'en-US']
const currencyOptions = ['IDR', 'USD']

const loading = ref(false)
const error = ref('')
const saved = ref(false)
const form = reactive<Partial<TenantSettings>>({ display_name: '', logo_url: '', timezone: 'Asia/Jakarta', locale: 'id-ID', currency: 'IDR' })

function validateSettings() {
  if (form.logo_url) {
    try {
      const url = new URL(form.logo_url)
      if (!['http:', 'https:'].includes(url.protocol)) return 'Logo URL harus memakai http atau https.'
    } catch {
      return 'Logo URL tidak valid.'
    }
  }

  if (!timezoneOptions.includes(form.timezone || '')) return 'Timezone tidak valid.'
  if (!localeOptions.includes(form.locale || '')) return 'Locale tidak valid.'
  if (!currencyOptions.includes(form.currency || '')) return 'Currency tidak valid.'

  return ''
}

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
  const validationError = validateSettings()
  if (validationError) {
    error.value = validationError
    saved.value = false
    return
  }

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
