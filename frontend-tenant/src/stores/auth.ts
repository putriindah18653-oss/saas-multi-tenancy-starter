import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { UserProfile } from '@/contracts/api'

export type { UserProfile } from '@/contracts/api'

const REFRESH_TOKEN_KEY = 'refresh_token'

function clearStoredRefreshToken() {
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const hasRefreshCookie = ref(true)
  const user = ref<UserProfile | null>(null)
  const hydrated = ref(false)
  const hydrating = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value)
  const defaultHomeRoute = computed(() => 'tenant-home')

  function setSession(payload: { accessToken: string; refreshToken?: string | null; user?: UserProfile | null }) {
    accessToken.value = payload.accessToken
    hasRefreshCookie.value = true
    clearStoredRefreshToken()
    if (payload.user !== undefined) user.value = payload.user
  }

  function setHydrationState(payload: { hydrated?: boolean; hydrating?: boolean }) {
    if (payload.hydrated !== undefined) hydrated.value = payload.hydrated
    if (payload.hydrating !== undefined) hydrating.value = payload.hydrating
  }

  function clearSession() {
    accessToken.value = null
    hasRefreshCookie.value = false
    clearStoredRefreshToken()
    user.value = null
    hydrated.value = true
    hydrating.value = false
  }

  return {
    accessToken,
    refreshToken: hasRefreshCookie,
    user,
    hydrated,
    hydrating,
    isAuthenticated,
    defaultHomeRoute,
    setSession,
    setHydrationState,
    clearSession,
  }
})
