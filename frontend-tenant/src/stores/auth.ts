import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export type UserProfile = {
  id: string
  email: string
  full_name?: string
  app_role?: string
  must_change_password?: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(sessionStorage.getItem('refresh_token'))
  const user = ref<UserProfile | null>(null)

  const isAuthenticated = computed(() => !!accessToken.value)
  const defaultHomeRoute = computed(() => 'tenant-home')

  function setSession(payload: { accessToken: string; refreshToken?: string; user?: UserProfile | null }) {
    accessToken.value = payload.accessToken
    if (payload.refreshToken) {
      refreshToken.value = payload.refreshToken
      sessionStorage.setItem('refresh_token', payload.refreshToken)
    }
    if (payload.user !== undefined) user.value = payload.user
  }

  function clearSession() {
    accessToken.value = null
    refreshToken.value = null
    sessionStorage.removeItem('refresh_token')
    user.value = null
  }

  return { accessToken, refreshToken, user, isAuthenticated, defaultHomeRoute, setSession, clearSession }
})
