<template>
  <div class="space-y-4">
    <div>
      <h2 class="owner-page-title">Profile</h2>
      <p class="owner-page-subtitle">Kelola informasi akun user owner. Data company ada di menu Settings.</p>
    </div>

    <section class="overflow-hidden rounded-[var(--radius-card)] border border-[var(--border)] bg-[var(--surface)]">
      <div class="grid min-h-[620px] lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside class="border-b border-[var(--border)] p-5 lg:border-b-0 lg:border-r">
          <h3 class="text-xl font-semibold text-[var(--text-primary)]">Profile</h3>
          <p class="mt-6 text-xs font-semibold uppercase tracking-[0.18em] text-[var(--text-muted)]">Menu</p>

          <nav class="mt-3 flex gap-2 overflow-x-auto pb-1 lg:flex-col lg:overflow-visible lg:pb-0" aria-label="Profile tabs">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              class="flex shrink-0 items-center gap-3 rounded-[var(--radius-button)] px-3 py-2.5 text-left text-sm font-medium transition lg:w-full"
              :class="activeTab === tab.id
                ? 'bg-[var(--accent)] text-[var(--bg-app)]'
                : 'text-[var(--text-muted)] hover:bg-white/5 hover:text-[var(--text-primary)]'"
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
          <div class="border-b border-dashed border-[var(--border)] px-5 py-5 sm:px-8">
            <h3 class="text-xl font-semibold text-[var(--text-primary)]">{{ currentTab?.label }}</h3>
          </div>

          <div v-if="activeTab === 'basic'" class="divide-y divide-dashed divide-[var(--border)]">
            <section class="grid gap-4 px-5 py-6 sm:px-8 md:grid-cols-[240px_minmax(0,1fr)]">
              <div>
                <h4 class="font-medium text-[var(--text-primary)]">Profile Photo</h4>
                <p class="mt-1 text-sm text-[var(--text-muted)]">Upload your profile image</p>
              </div>
              <div>
                <ImageUpload
                  ref="avatarUpload"
                  v-model="form.avatar_url"
                  title="Profile avatar"
                  hint="Drop or select a cover image"
                  meta="PNG, JPG, WebP. Recommended 200×200."
                  shape="circle"
                  alt="Profile photo of {{ displayName }}"
                  :initials="initials"
                  :resolve-url="resolveAvatarUrl"
                />
              </div>
            </section>

            <ProfileField title="Name and Surname" helper="Your full legal name and last name">
              <input id="profile-name" v-model="form.name" required maxlength="120" class="owner-input" placeholder="Nama lengkap" />
            </ProfileField>

            <ProfileField title="Email Address" helper="Provide a valid email">
              <input :value="auth.user?.email || ''" disabled class="owner-input" placeholder="email@example.com" />
            </ProfileField>

            <ProfileField title="Phone Number" helper="Include country code">
              <input id="profile-phone" v-model="form.phone" maxlength="40" class="owner-input" placeholder="+62 8xx xxxx xxxx" />
            </ProfileField>

            <ProfileField title="Biography" helper="Short personal description">
              <textarea id="profile-bio" v-model="form.bio" maxlength="500" rows="5" class="owner-input resize-y" placeholder="Bio singkat" />
            </ProfileField>
          </div>

          <div v-else class="divide-y divide-dashed divide-[var(--border)]">
            <ProfileField title="Current Password" helper="Masukkan password lama akun.">
              <div class="relative">
                <input id="current-password" v-model="passwordForm.current_password" :type="passwordVisible.current ? 'text' : 'password'" autocomplete="current-password" required class="owner-input pr-11" placeholder="Password lama" />
                <button type="button" class="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-[var(--text-muted)] transition hover:text-[var(--text-primary)]" :aria-label="passwordVisible.current ? 'Hide current password' : 'Show current password'" @click="passwordVisible.current = !passwordVisible.current">
                  <svg v-if="passwordVisible.current" class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M3 3l18 18" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M10.58 10.58a2 2 0 0 0 2.83 2.83" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M9.88 5.1A9.7 9.7 0 0 1 12 5c5 0 8.5 4.5 9.5 7a12.4 12.4 0 0 1-2.2 3.3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M6.6 6.6A12.3 12.3 0 0 0 2.5 12c1 2.5 4.5 7 9.5 7a9.8 9.8 0 0 0 4.5-1.1" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                  <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M2.5 12S6 5 12 5s9.5 7 9.5 7-3.5 7-9.5 7-9.5-7-9.5-7Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
              </div>
            </ProfileField>

            <ProfileField title="New Password" helper="Minimal 12 karakter, gunakan kombinasi aman.">
              <div class="relative">
                <input id="new-password" v-model="passwordForm.new_password" :type="passwordVisible.new ? 'text' : 'password'" autocomplete="new-password" required minlength="12" class="owner-input pr-11" placeholder="Password baru" />
                <button type="button" class="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-[var(--text-muted)] transition hover:text-[var(--text-primary)]" :aria-label="passwordVisible.new ? 'Hide new password' : 'Show new password'" @click="passwordVisible.new = !passwordVisible.new">
                  <svg v-if="passwordVisible.new" class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M3 3l18 18" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M10.58 10.58a2 2 0 0 0 2.83 2.83" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M9.88 5.1A9.7 9.7 0 0 1 12 5c5 0 8.5 4.5 9.5 7a12.4 12.4 0 0 1-2.2 3.3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M6.6 6.6A12.3 12.3 0 0 0 2.5 12c1 2.5 4.5 7 9.5 7a9.8 9.8 0 0 0 4.5-1.1" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                  <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M2.5 12S6 5 12 5s9.5 7 9.5 7-3.5 7-9.5 7-9.5-7-9.5-7Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
              </div>
            </ProfileField>

            <ProfileField title="Confirm Password" helper="Ulangi password baru.">
              <div class="relative">
                <input id="confirm-password" v-model="passwordForm.confirm_password" :type="passwordVisible.confirm ? 'text' : 'password'" autocomplete="new-password" required minlength="12" class="owner-input pr-11" placeholder="Konfirmasi password baru" />
                <button type="button" class="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-[var(--text-muted)] transition hover:text-[var(--text-primary)]" :aria-label="passwordVisible.confirm ? 'Hide confirm password' : 'Show confirm password'" @click="passwordVisible.confirm = !passwordVisible.confirm">
                  <svg v-if="passwordVisible.confirm" class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M3 3l18 18" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M10.58 10.58a2 2 0 0 0 2.83 2.83" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M9.88 5.1A9.7 9.7 0 0 1 12 5c5 0 8.5 4.5 9.5 7a12.4 12.4 0 0 1-2.2 3.3" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M6.6 6.6A12.3 12.3 0 0 0 2.5 12c1 2.5 4.5 7 9.5 7a9.8 9.8 0 0 0 4.5-1.1" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                  <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M2.5 12S6 5 12 5s9.5 7 9.5 7-3.5 7-9.5 7-9.5-7-9.5-7Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
              </div>
            </ProfileField>
          </div>

          <div class="space-y-3 border-t border-dashed border-[var(--border)] px-5 py-5 sm:px-8">
            <UiAlert v-if="success" :title="successTitle" tone="success">{{ successMessage }}</UiAlert>
            <UiAlert v-if="error" title="Gagal menyimpan" tone="danger">{{ error }}</UiAlert>

            <div class="flex flex-wrap justify-end gap-2">
              <AppButton variant="secondary" :disabled="loading" @click="resetActiveTab">Cancel</AppButton>
              <AppButton type="submit" :disabled="loading">
                {{ loading ? 'Saving...' : 'Continue' }}
              </AppButton>
            </div>
          </div>
        </form>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, reactive, ref, watch } from 'vue'
import AppButton from '@/components/common/AppButton.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import { appEnv } from '@/app/env'
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
        h('h4', { class: 'font-medium text-[var(--text-primary)]' }, props.title),
        h('p', { class: 'mt-1 text-sm text-[var(--text-muted)]' }, props.helper),
      ]),
      h('div', { class: 'min-w-0' }, slots.default?.()),
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
const displayName = computed(() => auth.user?.name || auth.user?.full_name || auth.user?.email || 'Owner')
const initials = computed(() => displayName.value.split(/\s+|@/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join('') || 'OW')

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
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: { message?: string } } } }
    error.value = e?.response?.data?.error?.message || 'Profile gagal disimpan.'
    console.error('[profile] save failed', err)
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
  } catch (err: unknown) {
    const e = err as { response?: { data?: { error?: { message?: string } } } }
    error.value = e?.response?.data?.error?.message || 'Password gagal disimpan.'
    console.error('[profile] password change failed', err)
  } finally {
    loading.value = false
  }
}

watch(() => auth.user, resetProfileForm, { immediate: true })
</script>
