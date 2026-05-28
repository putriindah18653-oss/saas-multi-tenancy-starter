import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export type UserProfile = {
  id: string
  email: string
  name?: string
  full_name?: string
  phone?: string
  address?: string
  avatar_url?: string
  bio?: string
  app_role?: string
  must_change_password?: boolean
}

const REFRESH_TOKEN_KEY = 'refresh_token'

function readStoredRefreshToken() {
  const token = localStorage.getItem(REFRESH_TOKEN_KEY) || sessionStorage.getItem(REFRESH_TOKEN_KEY)
  if (token) localStorage.setItem(REFRESH_TOKEN_KEY, token)
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  return token
}

function persistRefreshToken(token: string | null) {
  if (token) localStorage.setItem(REFRESH_TOKEN_KEY, token)
  else localStorage.removeItem(REFRESH_TOKEN_KEY)
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(readStoredRefreshToken())
  const user = ref<UserProfile | null>(null)
  const hydrated = ref(false)
  const hydrating = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value)
  const defaultHomeRoute = computed(() => 'tenant-home')

  function setSession(payload: { accessToken: string; refreshToken?: string | null; user?: UserProfile | null }) {
    accessToken.value = payload.accessToken
    if (payload.refreshToken !== undefined) {
      refreshToken.value = payload.refreshToken
      persistRefreshToken(payload.refreshToken)
    }
    if (payload.user !== undefined) user.value = payload.user
  }

  function setHydrationState(payload: { hydrated?: boolean; hydrating?: boolean }) {
    if (payload.hydrated !== undefined) hydrated.value = payload.hydrated
    if (payload.hydrating !== undefined) hydrating.value = payload.hydrating
  }

  function clearSession() {
    accessToken.value = null
    refreshToken.value = null
    persistRefreshToken(null)
    user.value = null
    hydrated.value = true
    hydrating.value = false
  }

  return {
    accessToken,
    refreshToken,
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
