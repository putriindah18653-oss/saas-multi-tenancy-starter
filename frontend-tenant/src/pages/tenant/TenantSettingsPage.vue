<template>
  <div class="space-y-5">
    <div>
      <h2 class="tenant-page-title">Tenant Settings</h2>
      <p class="tenant-page-subtitle">Kelola branding dan preferensi tenant aktif.</p>
    </div>

    <UiAlert v-if="error" title="Error" tone="danger">{{ error }}</UiAlert>
    <UiAlert v-if="saved" title="Tersimpan" tone="success">Settings tenant berhasil disimpan.</UiAlert>

    <div class="settings-card">
      <nav class="settings-nav" aria-label="Tenant settings tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="settings-nav__item"
          :class="{ 'settings-nav__item--active': activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          <span class="settings-nav__icon">{{ tab.icon }}</span>
          <span>{{ tab.label }}</span>
        </button>
      </nav>

      <form class="settings-content" @submit.prevent="save">
        <div v-show="activeTab === 'branding'" class="settings-fields">
          <label class="settings-field">
            <span class="tenant-label">Display name</span>
            <input v-model.trim="form.display_name" class="tenant-input" maxlength="120" placeholder="Nama tenant" />
          </label>

          <label class="settings-field">
            <span class="tenant-label">Logo URL</span>
            <input v-model.trim="form.logo_url" class="tenant-input" placeholder="https://..." inputmode="url" />
            <span class="tenant-helper">Opsional. Harus URL http/https valid.</span>
          </label>
        </div>

        <div v-show="activeTab === 'regional'" class="settings-fields">
          <div class="settings-fields__row settings-fields__row--3">
            <label class="settings-field">
              <span class="tenant-label">Timezone</span>
              <select v-model="form.timezone" class="tenant-input">
                <option v-for="timezone in timezoneOptions" :key="timezone" :value="timezone">{{ timezone }}</option>
              </select>
            </label>
            <label class="settings-field">
              <span class="tenant-label">Locale</span>
              <select v-model="form.locale" class="tenant-input">
                <option v-for="locale in localeOptions" :key="locale" :value="locale">{{ locale }}</option>
              </select>
            </label>
            <label class="settings-field">
              <span class="tenant-label">Currency</span>
              <select v-model="form.currency" class="tenant-input">
                <option v-for="currency in currencyOptions" :key="currency" :value="currency">{{ currency }}</option>
              </select>
            </label>
          </div>
        </div>

        <div v-show="activeTab === 'advanced'" class="settings-fields">
          <div class="rounded-[var(--tenant-radius-card)] border border-dashed border-[var(--tenant-border)] bg-[var(--tenant-surface-elevated)] p-4">
            <p class="text-sm font-semibold text-[var(--tenant-text-primary)]">Metadata</p>
            <p class="mt-1 text-sm text-[var(--tenant-text-muted)]">Konfigurasi tambahan tenant akan ditampilkan di sini saat backend metadata siap.</p>
          </div>
        </div>

        <div class="settings-actions">
          <button type="button" class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] px-4 py-2 text-sm font-medium text-[var(--tenant-text-secondary)] transition hover:bg-[var(--tenant-surface-elevated)] hover:text-[var(--tenant-text-primary)] disabled:opacity-60" :disabled="loading" @click="load">
            Reset
          </button>
          <button type="submit" class="rounded-[var(--tenant-radius-button)] bg-[var(--tenant-accent)] px-4 py-2 text-sm font-medium text-slate-950 transition hover:bg-[var(--tenant-accent-hover)] disabled:opacity-60" :disabled="loading">
            {{ loading ? 'Saving...' : 'Save' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import UiAlert from '@/components/common/UiAlert.vue'
import { tenantSettingsService, type TenantSettings } from '@/services/tenantSettings'

const tabs = [
  { id: 'branding', label: 'Branding', icon: '◎' },
  { id: 'regional', label: 'Regional', icon: '◷' },
  { id: 'advanced', label: 'Advanced', icon: '◇' },
] as const

type TabId = (typeof tabs)[number]['id']

const timezoneOptions = ['Asia/Jakarta', 'Asia/Makassar', 'Asia/Jayapura', 'UTC']
const localeOptions = ['id-ID', 'en-US']
const currencyOptions = ['IDR', 'USD']

const activeTab = ref<TabId>('branding')
const loading = ref(false)
const error = ref('')
const saved = ref(false)
const form = reactive<Partial<TenantSettings>>({ display_name: '', logo_url: '', timezone: 'Asia/Jakarta', locale: 'id-ID', currency: 'IDR', metadata: {} })

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
  saved.value = false
  try {
    const { data } = await tenantSettingsService.get()
    Object.assign(form, data.data)
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Gagal memuat settings tenant.'
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
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Gagal menyimpan settings tenant.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.settings-card {
  display: grid;
  grid-template-columns: 200px 1fr;
  border: 1px solid var(--tenant-border);
  border-radius: var(--tenant-radius-card);
  background: var(--tenant-surface);
  overflow: hidden;
  min-height: 420px;
}

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1rem 0.75rem;
  border-right: 1px solid var(--tenant-border);
  background: var(--tenant-surface-elevated);
}

.settings-nav__item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 0.85rem;
  border-radius: var(--tenant-radius-button);
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--tenant-text-muted);
  transition: background 0.15s, color 0.15s;
  cursor: pointer;
  border: none;
  background: none;
  text-align: left;
  width: 100%;
}

.settings-nav__item:hover {
  color: var(--tenant-text-primary);
  background: rgba(255, 255, 255, 0.04);
}

.settings-nav__item--active {
  color: var(--tenant-text-primary);
  background: rgba(53, 211, 153, 0.1);
}

.settings-nav__icon {
  font-size: 1rem;
  width: 20px;
  text-align: center;
}

.settings-content {
  display: flex;
  flex-direction: column;
  padding: 1.5rem;
}

.settings-fields {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.settings-fields__row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.settings-fields__row--3 {
  grid-template-columns: repeat(3, 1fr);
}

.settings-field {
  display: block;
}

.settings-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--tenant-border);
}

@media (max-width: 640px) {
  .settings-card {
    grid-template-columns: 1fr;
  }

  .settings-nav {
    flex-direction: row;
    border-right: none;
    border-bottom: 1px solid var(--tenant-border);
    padding: 0.75rem;
    overflow-x: auto;
  }

  .settings-fields__row,
  .settings-fields__row--3 {
    grid-template-columns: 1fr;
  }
}
</style>
