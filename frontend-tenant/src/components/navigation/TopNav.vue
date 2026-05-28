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

      <button type="button" class="topbar-action" title="Layar Penuh" aria-label="Toggle fullscreen" @click="toggleFullscreen">
        ⛶
      </button>
      <button type="button" class="topbar-action" title="Tema" aria-label="Toggle theme" @click="ui.toggleTheme">
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
          🔔
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

      <div class="topbar-profile">
        <button type="button" class="topbar-avatar" aria-label="Profile">
          {{ initials }}
        </button>
        <div class="topbar-profile-copy">
          <p>{{ auth.user?.full_name || 'Tenant User' }}</p>
          <span>{{ auth.user?.email || 'guest' }}</span>
        </div>
      </div>

      <LogoutButton />
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import LogoutButton from '@/components/navigation/LogoutButton.vue'

defineProps<{ title: string }>()
const auth = useAuthStore()
const ui = useUiStore()
const now = ref(new Date())
const isNotificationOpen = ref(false)
const notificationRef = ref<HTMLElement | null>(null)
let timer: number | undefined

const formattedDate = computed(() =>
  new Intl.DateTimeFormat('id-ID', { weekday: 'short', day: '2-digit', month: 'short', year: 'numeric' }).format(now.value),
)
const formattedTime = computed(() =>
  new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(now.value),
)
const initials = computed(() => {
  const source = auth.user?.full_name || auth.user?.email || 'Tenant User'
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
}

function handleDocumentClick(event: MouseEvent) {
  if (!notificationRef.value?.contains(event.target as Node)) {
    isNotificationOpen.value = false
  }
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
