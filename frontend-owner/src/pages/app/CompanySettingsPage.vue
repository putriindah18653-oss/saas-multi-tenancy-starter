<template>
  <div class="space-y-5">
    <div>
      <h2 class="owner-page-title">Company Settings</h2>
      <p class="owner-page-subtitle">Kelola identitas dan konfigurasi platform.</p>
    </div>

    <UiAlert v-if="error" title="Error" tone="danger">{{ error }}</UiAlert>
    <UiAlert v-if="saved" title="Tersimpan" tone="success">Settings berhasil disimpan.</UiAlert>

    <div class="settings-card">
      <nav class="settings-nav">
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
        <!-- Branding -->
        <div v-show="activeTab === 'branding'" class="settings-fields">
          <div class="settings-fields__row">
            <label class="settings-field">
              <span class="owner-label">Display name</span>
              <input v-model.trim="form.display_name" class="owner-input" maxlength="120" placeholder="PortalOnline" />
            </label>
            <label class="settings-field">
              <span class="owner-label">Legal name</span>
              <input v-model.trim="form.legal_name" class="owner-input" maxlength="160" placeholder="PT Portal Online Indonesia" />
            </label>
          </div>

          <div class="settings-field">
            <span class="owner-label">Logo</span>
            <ImageUpload
              ref="logoUpload"
              v-model="form.logo_url"
              title="Company logo"
              hint="Drag & drop atau klik. Auto resize 200×200."
              meta="PNG, JPG, WebP, GIF, AVIF. Max 8 MB."
              :initials="logoInitials"
              :resolve-url="resolveAssetUrl"
            />
          </div>

          <label class="settings-field">
            <span class="owner-label">Website</span>
            <input v-model.trim="form.website_url" class="owner-input" inputmode="url" placeholder="https://company.com" />
          </label>
        </div>

        <!-- Contact -->
        <div v-show="activeTab === 'contact'" class="settings-fields">
          <div class="settings-fields__row">
            <label class="settings-field">
              <span class="owner-label">Support email</span>
              <input v-model.trim="form.support_email" class="owner-input" type="email" placeholder="support@company.com" />
            </label>
            <label class="settings-field">
              <span class="owner-label">Support phone</span>
              <input v-model.trim="form.support_phone" class="owner-input" maxlength="40" placeholder="+62 21 xxxx xxxx" />
            </label>
          </div>

          <label class="settings-field">
            <span class="owner-label">Address</span>
            <textarea v-model.trim="form.address" class="owner-input min-h-24 resize-y" maxlength="500" placeholder="Alamat lengkap company" />
          </label>
        </div>

        <!-- Regional -->
        <div v-show="activeTab === 'regional'" class="settings-fields">
          <div class="settings-fields__row settings-fields__row--3">
            <label class="settings-field">
              <span class="owner-label">Timezone</span>
              <select v-model="form.timezone" class="owner-input">
                <option v-for="tz in timezoneOptions" :key="tz" :value="tz">{{ tz }}</option>
              </select>
            </label>
            <label class="settings-field">
              <span class="owner-label">Locale</span>
              <select v-model="form.locale" class="owner-input">
                <option v-for="loc in localeOptions" :key="loc" :value="loc">{{ loc }}</option>
              </select>
            </label>
            <label class="settings-field">
              <span class="owner-label">Currency</span>
              <select v-model="form.currency" class="owner-input">
                <option v-for="cur in currencyOptions" :key="cur" :value="cur">{{ cur }}</option>
              </select>
            </label>
          </div>
        </div>

        <div class="settings-actions">
          <AppButton variant="secondary" type="button" :disabled="loading" @click="load">Reset</AppButton>
          <AppButton type="submit" :disabled="loading">{{ loading ? 'Saving...' : 'Save' }}</AppButton>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppButton from '@/components/common/AppButton.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import { appEnv } from '@/app/env'
import { appSettingsService, defaultAppCompanySettings, type AppCompanySettings } from '@/services/appSettings'

const tabs = [
  { id: 'branding', label: 'Branding', icon: '◎' },
  { id: 'contact', label: 'Contact', icon: '✉' },
  { id: 'regional', label: 'Regional', icon: '◷' },
] as const

type TabId = (typeof tabs)[number]['id']

const timezoneOptions = ['Asia/Jakarta', 'Asia/Makassar', 'Asia/Jayapura', 'UTC']
const localeOptions = ['id-ID', 'en-US']
const currencyOptions = ['IDR', 'USD']

const activeTab = ref<TabId>('branding')
const loading = ref(false)
const error = ref('')
const saved = ref(false)
const logoUpload = ref<InstanceType<typeof ImageUpload> | null>(null)
const form = reactive<AppCompanySettings>({ ...defaultAppCompanySettings })

const logoInitials = computed(() => {
  const base = form.display_name || 'PO'
  return base.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]?.toUpperCase()).join('') || 'PO'
})

function resolveAssetUrl(url: string): string {
  if (!url) return ''
  if (url.startsWith('http')) return url
  const base = appEnv.apiBaseUrl || `${window.location.protocol}//${window.location.hostname}:8080/api/v1`
  return base.replace(/\/$/, '') + url
}

async function load() {
  loading.value = true
  error.value = ''
  saved.value = false
  try {
    const { data } = await appSettingsService.get()
    Object.assign(form, defaultAppCompanySettings, data.data)
  } catch (e: any) {
    Object.assign(form, defaultAppCompanySettings)
    error.value = e?.response?.data?.error?.message || 'Gagal memuat settings.'
  } finally {
    loading.value = false
  }
}

async function save() {
  loading.value = true
  error.value = ''
  saved.value = false
  try {
    const pendingFile = logoUpload.value?.pendingFile
    if (pendingFile) {
      const { data: uploadData } = await appSettingsService.uploadLogo(pendingFile)
      form.logo_url = uploadData.data.url_path
    }
    const { data } = await appSettingsService.update(form)
    Object.assign(form, defaultAppCompanySettings, data.data)
    saved.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Gagal menyimpan settings.'
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
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  background: var(--surface);
  overflow: hidden;
  min-height: 420px;
}

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1rem 0.75rem;
  border-right: 1px solid var(--border);
  background: var(--surface-elevated);
}

.settings-nav__item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 0.85rem;
  border-radius: var(--radius-button);
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-muted);
  transition: background 0.15s, color 0.15s;
  cursor: pointer;
  border: none;
  background: none;
  text-align: left;
  width: 100%;
}

.settings-nav__item:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.04);
}

.settings-nav__item--active {
  color: var(--text-primary);
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
  border-top: 1px solid var(--border);
}

@media (max-width: 640px) {
  .settings-card {
    grid-template-columns: 1fr;
  }

  .settings-nav {
    flex-direction: row;
    border-right: none;
    border-bottom: 1px solid var(--border);
    padding: 0.75rem;
    overflow-x: auto;
  }

  .settings-fields__row,
  .settings-fields__row--3 {
    grid-template-columns: 1fr;
  }
}
</style>
