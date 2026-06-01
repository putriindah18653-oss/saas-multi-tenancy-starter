<template>
  <div class="max-w-3xl space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h2 class="owner-page-title">Tenant detail</h2>
        <p class="owner-page-subtitle">Update nama dan status tenant sesuai izin owner.</p>
      </div>
      <RouterLink to="/app/tenants" class="text-sm font-medium text-[var(--accent)] hover:text-[var(--accent-hover)]">Back to tenants</RouterLink>
    </div>

    <UiAlert v-if="error" title="Tenant request failed" tone="danger">{{ error }}</UiAlert>
    <UiAlert v-if="saved" title="Tenant updated" tone="success">Perubahan sudah tersimpan.</UiAlert>

    <AppCard v-if="loading && !tenant">
      <div class="space-y-3">
        <div class="h-10 animate-pulse rounded-[var(--radius-button)] bg-white/[0.05]" />
        <div class="h-10 animate-pulse rounded-[var(--radius-button)] bg-white/[0.05]" />
        <div class="h-10 animate-pulse rounded-[var(--radius-button)] bg-white/[0.05]" />
      </div>
    </AppCard>

    <UiEmptyState v-else-if="!tenant" title="Tenant not available" description="Tenant tidak ditemukan atau belum bisa dimuat." />

    <AppCard v-else>
      <form class="space-y-5" @submit.prevent="save">
        <div>
          <label for="tenant-name" class="owner-label">Name</label>
          <input id="tenant-name" v-model="form.name" :disabled="!canManage" class="owner-input" />
        </div>

        <div>
          <label for="tenant-slug" class="owner-label">Slug</label>
          <input id="tenant-slug" :value="tenant.slug" disabled class="owner-input" />
        </div>

        <div>
          <label for="tenant-status" class="owner-label">Status</label>
          <select id="tenant-status" v-model="form.status" :disabled="!canManage" class="owner-input">
            <option value="active">active</option>
            <option value="suspended">suspended</option>
            <option value="deleted">deleted</option>
          </select>
        </div>

        <div v-if="!canManage" class="rounded-[var(--radius-card)] border border-[var(--border)] bg-white/[0.03] p-3 text-sm text-[var(--text-muted)]">
          Akun Anda hanya punya akses baca untuk tenant ini.
        </div>

        <div class="flex flex-wrap gap-2">
          <AppButton type="submit" :disabled="loading || !canManage">
            {{ loading ? 'Saving...' : 'Save changes' }}
          </AppButton>
          <AppButton variant="danger" :disabled="loading || !canDelete" @click="confirmDelete = true">
            Delete tenant
          </AppButton>
          <RouterLink to="/app/tenants" class="inline-flex min-h-10 items-center rounded-[var(--radius-button)] border border-[var(--border-strong)] px-4 py-2 text-sm text-[var(--text-secondary)] hover:bg-white/5">
            Back
          </RouterLink>
        </div>
      </form>
    </AppCard>

    <!-- Delete confirmation modal -->
    <Teleport to="body">
      <div v-if="confirmDelete" class="fixed inset-0 z-[100] flex items-center justify-center" role="dialog" aria-modal="true" aria-labelledby="delete-dialog-title">
        <div class="fixed inset-0 bg-black/50 backdrop-blur-sm" @click="confirmDelete = false" />
        <div class="relative z-10 mx-4 w-full max-w-md rounded-[var(--radius-card)] border border-[var(--border)] bg-[var(--surface-elevated)] p-6 shadow-2xl">
          <h3 id="delete-dialog-title" class="text-lg font-semibold text-[var(--text-primary)]">Hapus tenant?</h3>
          <p class="mt-2 text-sm text-[var(--text-muted)]">
            Tenant <strong class="text-[var(--text-primary)]">{{ tenant?.name }}</strong> akan dihapus. Data tenant dapat dipulihkan oleh administrator.
          </p>
          <div class="mt-6 flex justify-end gap-2">
            <AppButton variant="secondary" :disabled="loading" @click="confirmDelete = false">Batal</AppButton>
            <AppButton variant="danger" :disabled="loading" @click="removeTenant">
              {{ loading ? 'Menghapus...' : 'Hapus tenant' }}
            </AppButton>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppButton from '@/components/common/AppButton.vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
import { useAuthStore } from '@/stores/auth'
import { tenantsService, type Tenant } from '@/services/tenants'
import { canOwner } from '@/services/rbac'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const id = computed(() => String(route.params.id || ''))
const tenant = ref<Tenant | null>(null)
const loading = ref(false)
const error = ref('')
const saved = ref(false)
const form = reactive({ name: '', status: 'active' })

const canManage = computed(() => canOwner(auth.user, 'app.tenants.update'))
const canDelete = computed(() => canOwner(auth.user, 'app.tenants.delete'))
const confirmDelete = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await tenantsService.get(id.value)
    tenant.value = res.data.data
    form.name = tenant.value?.name || ''
    form.status = tenant.value?.status || 'active'
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal memuat tenant'
    console.error('[tenant-detail] load failed', e)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!canManage.value) return
  loading.value = true
  error.value = ''
  saved.value = false
  try {
    const res = await tenantsService.update(id.value, { name: form.name, status: form.status })
    tenant.value = res.data.data
    form.name = tenant.value.name
    form.status = tenant.value.status
    saved.value = true
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal menyimpan tenant'
    console.error('[tenant-detail] save failed', e)
  } finally {
    loading.value = false
  }
}

async function removeTenant() {
  if (!canDelete.value) return
  confirmDelete.value = false
  loading.value = true
  error.value = ''
  try {
    await tenantsService.remove(id.value)
    router.push('/app/tenants')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal menghapus tenant'
    console.error('[tenant-detail] delete failed', e)
  } finally {
    loading.value = false
  }
}

onMounted(load)

// Close delete dialog on Escape
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && confirmDelete.value) confirmDelete.value = false
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>
