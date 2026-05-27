import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export type UserProfile = {
  id: string
  email: string
  full_name?: string
  app_role?: string
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const user = ref<UserProfile | null>(null)

  const isAuthenticated = computed(() => !!accessToken.value)
  const defaultHomeRoute = computed(() => 'app-home')

  function setSession(payload: { accessToken: string; refreshToken?: string; user?: UserProfile | null }) {
    accessToken.value = payload.accessToken
    if (payload.refreshToken) refreshToken.value = payload.refreshToken
    if (payload.user !== undefined) user.value = payload.user
  }

  function clearSession() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
  }

  return { accessToken, refreshToken, user, isAuthenticated, defaultHomeRoute, setSession, clearSession }
})
