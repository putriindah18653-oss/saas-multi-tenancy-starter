<template>
  <header class="topbar">
    <div class="topbar-left">
      <button type="button" class="topbar-action hamburger" aria-label="Open sidebar" @click="ui.openMobileSidebar">
        ☰
      </button>
      <div>
        <p class="topbar-eyebrow">Tenant Control Center</p>
        <h1 class="topbar-title">{{ title }}</h1>
      </div>
    </div>

    <div class="topbar-right">
      <slot name="actions" />

      <div class="topbar-clock">
        <span>{{ formattedDate }}</span>
        <strong>{{ formattedTime }}</strong>
      </div>

      <button type="button" class="topbar-action" title="Fullscreen" aria-label="Toggle fullscreen" @click="toggleFullscreen">
        ⛶
      </button>
      <button type="button" class="topbar-action" title="Theme" aria-label="Toggle theme" @click="ui.toggleTheme">
        {{ ui.theme === 'dark' ? '☀' : '☾' }}
      </button>
      <div ref="notificationRef" class="topbar-notification">
        <button
          type="button"
          class="topbar-action topbar-action--notify"
          :class="{ active: isNotificationOpen }"
          title="Notifications"
          aria-label="Notifications"
          :aria-expanded="isNotificationOpen"
          aria-haspopup="menu"
          @click="toggleNotifications"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="M15 17H9m9-6a6 6 0 1 0-12 0c0 7-3 7-3 8h18c0-1-3-1-3-8Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
            <path d="M13.73 21a2 2 0 0 1-3.46 0" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span />
        </button>

        <div v-if="isNotificationOpen" class="notification-dropdown" role="menu">
          <div class="notification-dropdown__header">
            <div>
              <p>Notifikasi</p>
              <span>Tidak ada notifikasi baru</span>
            </div>
          </div>

          <div class="notification-empty" role="menuitem">
            <span class="notification-empty__icon">🔕</span>
            <strong>Belum ada notifikasi</strong>
            <span>Notifikasi tenant akan muncul di sini saat fitur backend tersedia.</span>
          </div>
        </div>
      </div>

      <div ref="profileRef" class="topbar-profile">
        <button
          type="button"
          class="topbar-profile-trigger"
          aria-label="Profile menu"
          :aria-expanded="profileMenuOpen"
          aria-haspopup="menu"
          @click="profileMenuOpen = !profileMenuOpen"
        >
          <span class="topbar-avatar">{{ initials }}</span>
        </button>

        <div v-if="profileMenuOpen" class="topbar-profile-menu" role="menu">
          <div class="topbar-profile-menu-user">
            <span class="topbar-profile-menu-avatar">{{ initials }}</span>
            <div class="min-w-0">
              <p>{{ auth.user?.full_name || auth.user?.name || 'Tenant User' }}</p>
              <span>{{ auth.user?.email || 'guest' }}</span>
            </div>
          </div>
          <RouterLink class="topbar-profile-menu-item" to="/tenant/profile" role="menuitem" @click="profileMenuOpen = false">
            Profile
          </RouterLink>
          <RouterLink class="topbar-profile-menu-item" to="/tenant/settings" role="menuitem" @click="profileMenuOpen = false">
            Settings
          </RouterLink>
          <button type="button" class="topbar-profile-menu-item topbar-profile-menu-item--danger" role="menuitem" @click="onLogout">
            Logout
          </button>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

defineProps<{ title: string }>()
const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()
const now = ref(new Date())
const isNotificationOpen = ref(false)
const profileMenuOpen = ref(false)
const notificationRef = ref<HTMLElement | null>(null)
const profileRef = ref<HTMLElement | null>(null)
let timer: number | undefined

const formattedDate = computed(() =>
  new Intl.DateTimeFormat('id-ID', { weekday: 'short', day: '2-digit', month: 'short', year: 'numeric' }).format(now.value),
)
const formattedTime = computed(() =>
  new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(now.value),
)
const initials = computed(() => {
  const source = auth.user?.full_name || auth.user?.name || auth.user?.email || 'Tenant User'
  return source
    .split(/\s+|@/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('') || 'TU'
})

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen?.()
    return
  }
  document.exitFullscreen?.()
}

function toggleNotifications() {
  isNotificationOpen.value = !isNotificationOpen.value
  profileMenuOpen.value = false
}

function handleDocumentClick(event: MouseEvent) {
  const target = event.target as Node
  if (!notificationRef.value?.contains(target)) {
    isNotificationOpen.value = false
  }
  if (!profileRef.value?.contains(target)) {
    profileMenuOpen.value = false
  }
}

async function onLogout() {
  profileMenuOpen.value = false
  let serverLogoutFailed = false
  try {
    if (auth.refreshToken) {
      await authService.logout(auth.refreshToken)
    }
  } catch {
    serverLogoutFailed = true
  }
  auth.clearSession()
  if (serverLogoutFailed) {
    alert('Warning: Server logout gagal. Session mungkin masih aktif di device lain.')
  }
  router.push('/auth/login')
}

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
  document.addEventListener('click', handleDocumentClick)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
  document.removeEventListener('click', handleDocumentClick)
})
</script>
