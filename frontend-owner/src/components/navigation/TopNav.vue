<template>
  <header class="topbar">
    <div class="topbar-left">
      <button type="button" class="topbar-action hamburger" aria-label="Open sidebar" @click="ui.openMobileSidebar">
        ☰
      </button>
      <h1 class="topbar-title">{{ title }}</h1>
    </div>

    <div class="topbar-right">
      <slot name="actions" />

      <button type="button" class="topbar-action" title="Fullscreen" aria-label="Toggle fullscreen" @click="toggleFullscreen">
        ⛶
      </button>
      <button type="button" class="topbar-action" title="Theme" aria-label="Toggle theme" @click="ui.toggleTheme">
        {{ ui.theme === 'dark' ? '☀' : '☾' }}
      </button>
      <button type="button" class="topbar-action topbar-action--notify" title="Notifications" aria-label="Notifications">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path d="M15 17H9m9-6a6 6 0 1 0-12 0c0 7-3 7-3 8h18c0-1-3-1-3-8Z" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <span />
      </button>

      <div class="topbar-profile">
        <button
          type="button"
          class="topbar-profile-trigger"
          aria-label="Profile menu"
          :aria-expanded="profileMenuOpen"
          aria-controls="owner-profile-menu"
          aria-haspopup="menu"
          @click="profileMenuOpen = !profileMenuOpen"
        >
          <span class="topbar-avatar">{{ initials }}</span>
        </button>

        <div v-if="profileMenuOpen" id="owner-profile-menu" class="topbar-profile-menu" role="menu">
          <div class="topbar-profile-menu-user">
            <span class="topbar-profile-menu-avatar">{{ initials }}</span>
            <div class="min-w-0">
              <p>{{ auth.user?.name || auth.user?.full_name || 'Owner' }}</p>
              <span>{{ auth.user?.email || 'guest' }}</span>
            </div>
          </div>
          <RouterLink class="topbar-profile-menu-item" to="/app/profile" role="menuitem" @click="profileMenuOpen = false">
            Profile
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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

defineProps<{ title: string }>()
const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()
const profileMenuOpen = ref(false)

const initials = computed(() => {
  const source = auth.user?.name || auth.user?.full_name || auth.user?.email || 'Owner'
  return source
    .split(/\s+|@/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('') || 'OW'
})

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen?.()
    return
  }
  document.exitFullscreen?.()
}

async function onLogout() {
  profileMenuOpen.value = false
  let serverLogoutFailed = false
  try {
    if (auth.refreshToken) {
      await authService.logout(auth.refreshToken)
    }
  } catch (err) {
    serverLogoutFailed = true
    console.warn('[logout] server revocation failed', err)
  }
  auth.clearSession()
  if (serverLogoutFailed) {
    alert('Warning: Server logout gagal. Session mungkin masih aktif di device lain.')
  }
  router.push('/auth/login')
}

// Close profile menu on Escape and click-outside
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') profileMenuOpen.value = false
}
function onClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement | null
  if (!target) return
  if (profileMenuOpen.value && !target.closest('.topbar-profile')) profileMenuOpen.value = false
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('click', onClickOutside)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('click', onClickOutside)
})
</script>
