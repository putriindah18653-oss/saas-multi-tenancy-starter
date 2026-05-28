<template>
  <div class="space-y-4">
    <div>
      <h2 class="tenant-page-title">Profile</h2>
      <p class="tenant-page-subtitle">Kelola informasi akun dan password user tenant.</p>
    </div>

    <UiAlert v-if="error" title="Error" tone="danger">{{ error }}</UiAlert>
    <UiAlert v-if="success" :title="successTitle" tone="success">{{ successMessage }}</UiAlert>

    <section class="overflow-hidden rounded-[var(--tenant-radius-card)] border border-[var(--tenant-border)] bg-[var(--tenant-surface)]">
      <div class="grid min-h-[620px] lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside class="border-b border-[var(--tenant-border)] p-5 lg:border-b-0 lg:border-r">
          <h3 class="text-xl font-semibold text-[var(--tenant-text-primary)]">Profile</h3>
          <p class="mt-6 text-xs font-semibold uppercase tracking-[0.18em] text-[var(--tenant-text-muted)]">Menu</p>

          <nav class="mt-3 flex gap-2 overflow-x-auto pb-1 lg:flex-col lg:overflow-visible lg:pb-0" aria-label="Profile tabs">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              class="flex shrink-0 items-center gap-3 rounded-[var(--tenant-radius-button)] px-3 py-2.5 text-left text-sm font-medium transition lg:w-full"
              :class="activeTab === tab.id
                ? 'bg-[var(--tenant-accent)] text-slate-950'
                : 'text-[var(--tenant-text-muted)] hover:bg-white/5 hover:text-[var(--tenant-text-primary)]'"
              @click="setActiveTab(tab.id)"
            >
              <svg v-if="tab.id === 'basic'" class="h-5 w-5 shrink-0" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                <path d="M4 20a8 8 0 0 1 16 0" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              <svg v-else class="h-5 w-5 shrink-0" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <path d="M7 11V8a5 5 0 0 1 10 0v3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                <path d="M6 11h12a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-8a1 1 0 0 1 1-1Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                <path d="M12 15v2" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              <span class="whitespace-nowrap">{{ tab.label }}</span>
            </button>
          </nav>
        </aside>

        <form class="min-w-0" @submit.prevent="submitActiveTab">
          <div class="border-b border-dashed border-[var(--tenant-border)] px-5 py-5 sm:px-8">
            <h3 class="text-xl font-semibold text-[var(--tenant-text-primary)]">{{ currentTab?.label }}</h3>
          </div>

          <div v-if="activeTab === 'basic'" class="divide-y divide-dashed divide-[var(--tenant-border)]">
            <ProfileField title="Profile Photo" helper="Upload foto profil user tenant.">
              <ImageUpload
                ref="avatarUpload"
                v-model="form.avatar_url"
                title="Profile avatar"
                hint="Drop atau pilih gambar avatar"
                meta="PNG, JPG, WebP, GIF, AVIF. Recommended 200×200."
                shape="circle"
                :initials="initials"
                :resolve-url="resolveAvatarUrl"
              />
            </ProfileField>

            <ProfileField title="Name" helper="Nama lengkap user tenant.">
              <input id="profile-name" v-model="form.name" required maxlength="120" class="tenant-input" placeholder="Nama lengkap" />
            </ProfileField>

            <ProfileField title="Email Address" helper="Email akun tidak dapat diubah dari halaman ini.">
              <input :value="auth.user?.email || ''" disabled class="tenant-input" placeholder="email@example.com" />
            </ProfileField>

            <ProfileField title="Phone Number" helper="Opsional. Sertakan kode negara bila perlu.">
              <input id="profile-phone" v-model="form.phone" maxlength="40" class="tenant-input" placeholder="+62 8xx xxxx xxxx" />
            </ProfileField>

            <ProfileField title="Address" helper="Alamat kontak user.">
              <textarea id="profile-address" v-model="form.address" maxlength="500" rows="3" class="tenant-input resize-y" placeholder="Alamat" />
            </ProfileField>

            <ProfileField title="Biography" helper="Deskripsi singkat user.">
              <textarea id="profile-bio" v-model="form.bio" maxlength="500" rows="5" class="tenant-input resize-y" placeholder="Bio singkat" />
            </ProfileField>
          </div>

          <div v-else class="divide-y divide-dashed divide-[var(--tenant-border)]">
            <ProfileField title="Current Password" helper="Masukkan password lama akun.">
              <PasswordInput
                id="current-password"
                v-model="passwordForm.current_password"
                :visible="passwordVisible.current"
                autocomplete="current-password"
                placeholder="Password lama"
                @toggle="passwordVisible.current = !passwordVisible.current"
              />
            </ProfileField>

            <ProfileField title="New Password" helper="Minimal 8 karakter.">
              <PasswordInput
                id="new-password"
                v-model="passwordForm.new_password"
                :visible="passwordVisible.new"
                autocomplete="new-password"
                placeholder="Password baru"
                @toggle="passwordVisible.new = !passwordVisible.new"
              />
            </ProfileField>

            <ProfileField title="Confirm Password" helper="Ulangi password baru.">
              <PasswordInput
                id="confirm-password"
                v-model="passwordForm.confirm_password"
                :visible="passwordVisible.confirm"
                autocomplete="new-password"
                placeholder="Konfirmasi password baru"
                @toggle="passwordVisible.confirm = !passwordVisible.confirm"
              />
            </ProfileField>
          </div>

          <div class="flex flex-wrap justify-end gap-3 border-t border-[var(--tenant-border)] px-5 py-5 sm:px-8">
            <button type="button" class="rounded-[var(--tenant-radius-button)] border border-[var(--tenant-border)] px-4 py-2 text-sm font-medium text-[var(--tenant-text-secondary)] transition hover:bg-[var(--tenant-surface-elevated)] hover:text-[var(--tenant-text-primary)] disabled:opacity-60" :disabled="loading" @click="resetActiveTab">
              Reset
            </button>
            <button type="submit" class="rounded-[var(--tenant-radius-button)] bg-[var(--tenant-accent)] px-4 py-2 text-sm font-medium text-slate-950 transition hover:bg-[var(--tenant-accent-hover)] disabled:opacity-60" :disabled="loading">
              {{ loading ? 'Menyimpan...' : activeTab === 'basic' ? 'Simpan profile' : 'Ganti password' }}
            </button>
          </div>
        </form>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, reactive, ref, watch } from 'vue'
import { appEnv } from '@/app/env'
import ImageUpload from '@/components/common/ImageUpload.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/auth'

const ProfileField = defineComponent({
  props: {
    title: { type: String, required: true },
    helper: { type: String, required: true },
  },
  setup(props, { slots }) {
    return () => h('section', { class: 'grid gap-4 px-5 py-6 sm:px-8 md:grid-cols-[240px_minmax(0,1fr)]' }, [
      h('div', [
        h('h4', { class: 'font-medium text-[var(--tenant-text-primary)]' }, props.title),
        h('p', { class: 'mt-1 text-sm text-[var(--tenant-text-muted)]' }, props.helper),
      ]),
      h('div', { class: 'min-w-0' }, slots.default?.()),
    ])
  },
})

const PasswordInput = defineComponent({
  props: {
    id: { type: String, required: true },
    modelValue: { type: String, required: true },
    visible: { type: Boolean, required: true },
    autocomplete: { type: String, required: true },
    placeholder: { type: String, required: true },
  },
  emits: ['update:modelValue', 'toggle'],
  setup(props, { emit }) {
    return () => h('div', { class: 'relative' }, [
      h('input', {
        id: props.id,
        value: props.modelValue,
        type: props.visible ? 'text' : 'password',
        autocomplete: props.autocomplete,
        required: true,
        class: 'tenant-input pr-11',
        placeholder: props.placeholder,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      }),
      h('button', {
        type: 'button',
        class: 'absolute inset-y-0 right-0 flex w-11 items-center justify-center text-[var(--tenant-text-muted)] transition hover:text-[var(--tenant-text-primary)]',
        'aria-label': props.visible ? 'Hide password' : 'Show password',
        onClick: () => emit('toggle'),
      }, props.visible ? '🙈' : '👁'),
    ])
  },
})

const tabs = [
  { id: 'basic', label: 'Basic Information' },
  { id: 'password', label: 'Change password' },
] as const

type TabId = (typeof tabs)[number]['id']

const auth = useAuthStore()
const activeTab = ref<TabId>('basic')
const loading = ref(false)
const error = ref('')
const success = ref(false)
const successTitle = ref('Profile tersimpan')
const successMessage = ref('Data profile berhasil diperbarui.')
const avatarUpload = ref<InstanceType<typeof ImageUpload> | null>(null)

const form = reactive({
  name: '',
  phone: '',
  address: '',
  avatar_url: '',
  bio: '',
})

const passwordForm = reactive({
  current_password: '',
  new_password: '',
  confirm_password: '',
})

const passwordVisible = reactive({
  current: false,
  new: false,
  confirm: false,
})

const currentTab = computed(() => tabs.find((tab) => tab.id === activeTab.value))
const displayName = computed(() => auth.user?.name || auth.user?.full_name || auth.user?.email || 'Tenant User')
const initials = computed(() => displayName.value.split(/\s+|@/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join('') || 'TU')

function resolveAvatarUrl(url: string): string {
  if (!url) return ''
  if (url.startsWith('http')) return url
  const base = appEnv.apiBaseUrl || `${window.location.protocol}//${window.location.hostname}:8080/api/v1`
  return base.replace(/\/$/, '') + url
}

function setActiveTab(tab: TabId) {
  activeTab.value = tab
  error.value = ''
  success.value = false
}

function resetProfileForm() {
  form.name = auth.user?.name || auth.user?.full_name || ''
  form.phone = auth.user?.phone || ''
  form.address = auth.user?.address || ''
  form.avatar_url = auth.user?.avatar_url || ''
  form.bio = auth.user?.bio || ''
}

function resetPasswordForm() {
  passwordForm.current_password = ''
  passwordForm.new_password = ''
  passwordForm.confirm_password = ''
  passwordVisible.current = false
  passwordVisible.new = false
  passwordVisible.confirm = false
}

function resetActiveTab() {
  if (activeTab.value === 'basic') resetProfileForm()
  else resetPasswordForm()
  error.value = ''
  success.value = false
}

async function submitActiveTab() {
  if (activeTab.value === 'basic') {
    await saveProfile()
    return
  }
  await changePassword()
}

async function saveProfile() {
  loading.value = true
  error.value = ''
  success.value = false
  try {
    const pendingFile = avatarUpload.value?.pendingFile
    if (pendingFile) {
      const { data: uploadData } = await authService.uploadAvatar(pendingFile)
      form.avatar_url = uploadData.data.url_path
    }
    const res = await authService.updateProfile({ ...form })
    auth.setSession({ accessToken: auth.accessToken || '', user: res.data.data })
    resetProfileForm()
    successTitle.value = 'Profile tersimpan'
    successMessage.value = 'Data profile berhasil diperbarui.'
    success.value = true
  } catch (err: any) {
    error.value = err?.response?.data?.error?.message || 'Profile gagal disimpan.'
  } finally {
    loading.value = false
  }
}

async function changePassword() {
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    error.value = 'Konfirmasi password tidak sama.'
    success.value = false
    return
  }
  if (passwordForm.new_password.length < 8) {
    error.value = 'Password baru minimal 8 karakter.'
    success.value = false
    return
  }

  loading.value = true
  error.value = ''
  success.value = false
  try {
    await authService.changePassword({
      current_password: passwordForm.current_password,
      new_password: passwordForm.new_password,
    })
    resetPasswordForm()
    successTitle.value = 'Password tersimpan'
    successMessage.value = 'Password berhasil diperbarui.'
    success.value = true
  } catch (err: any) {
    error.value = err?.response?.data?.error?.message || 'Password gagal disimpan.'
  } finally {
    loading.value = false
  }
}

watch(() => auth.user, resetProfileForm, { immediate: true })
</script>
