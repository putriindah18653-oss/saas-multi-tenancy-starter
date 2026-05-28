import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export type UserProfile = {
  id: string
  email: string
  full_name?: string
  app_role?: string
  permissions?: string[]
  must_change_password?: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(sessionStorage.getItem('refresh_token'))
  const user = ref<UserProfile | null>(null)
  const restoring = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value)
  const defaultHomeRoute = computed(() => 'app-home')

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

  async function tryRestoreSession(): Promise<boolean> {
    if (accessToken.value || !refreshToken.value) return !!accessToken.value
    restoring.value = true
    try {
      const { authApi } = await import('@/services/api')
      const res = await authApi.post('/auth/refresh', { refresh_token: refreshToken.value })
      const data = res.data.data
      setSession({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        user: data.user ?? null,
      })
      return true
    } catch {
      clearSession()
      return false
    } finally {
      restoring.value = false
    }
  }

  return { accessToken, refreshToken, user, restoring, isAuthenticated, defaultHomeRoute, setSession, clearSession, tryRestoreSession }
})
