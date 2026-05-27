<template>
  <div class="space-y-4">
    <div>
      <h2 class="text-xl font-semibold text-slate-900">Tenant Settings</h2>
      <p class="text-sm text-slate-500">Branding dan preferensi tenant aktif.</p>
    </div>

    <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>
    <div v-if="saved" class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">Settings tersimpan.</div>

    <form class="grid gap-4 rounded-xl border border-slate-200 bg-white p-4" @submit.prevent="save">
      <label class="space-y-1 text-sm">
        <span class="font-medium text-slate-700">Display name</span>
        <input v-model="form.display_name" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
      </label>
      <label class="space-y-1 text-sm">
        <span class="font-medium text-slate-700">Logo URL</span>
        <input v-model="form.logo_url" class="w-full rounded-lg border border-slate-300 px-3 py-2" placeholder="https://..." />
      </label>
      <div class="grid gap-4 md:grid-cols-3">
        <label class="space-y-1 text-sm">
          <span class="font-medium text-slate-700">Timezone</span>
          <input v-model="form.timezone" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </label>
        <label class="space-y-1 text-sm">
          <span class="font-medium text-slate-700">Locale</span>
          <input v-model="form.locale" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </label>
        <label class="space-y-1 text-sm">
          <span class="font-medium text-slate-700">Currency</span>
          <input v-model="form.currency" class="w-full rounded-lg border border-slate-300 px-3 py-2" />
        </label>
      </div>
      <button class="w-fit rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60" :disabled="loading">
        {{ loading ? 'Menyimpan...' : 'Simpan settings' }}
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
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
