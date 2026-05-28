<template>
  <header class="topbar">
    <div class="topbar-left">
      <button type="button" class="topbar-action hamburger" aria-label="Open sidebar" @click="ui.openMobileSidebar">
        ☰
      </button>
      <div>
        <p class="topbar-eyebrow">Owner Control Center</p>
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
      <button type="button" class="topbar-action topbar-action--notify" title="Notifications" aria-label="Notifications">
        🔔
        <span />
      </button>

      <div class="topbar-profile">
        <button type="button" class="topbar-avatar" aria-label="Profile">
          {{ initials }}
        </button>
        <div class="topbar-profile-copy">
          <p>{{ auth.user?.full_name || 'Owner' }}</p>
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
let timer: number | undefined

const formattedDate = computed(() =>
  new Intl.DateTimeFormat('id-ID', { weekday: 'short', day: '2-digit', month: 'short', year: 'numeric' }).format(now.value),
)
const formattedTime = computed(() =>
  new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(now.value),
)
const initials = computed(() => {
  const source = auth.user?.full_name || auth.user?.email || 'Owner'
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

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = new Date()
  }, 1000)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
})
</script>
