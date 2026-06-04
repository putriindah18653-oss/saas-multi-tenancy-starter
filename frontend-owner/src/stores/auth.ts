import axios from 'axios'
import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { appEnv } from '@/app/env'
import type { UserProfile } from '@/contracts/api'
import { devWarn, toLogContext } from '@/utils/devLogger'

const REFRESH_TOKEN_KEY='***'

function clearStoredRefreshToken() {
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

export type { UserProfile } from '@/contracts/api'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const hasRefreshCookie = ref(true)
  const user = ref<UserProfile | null>(null)
  const restoring = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value)
  const defaultHomeRoute = computed(() => 'app-home')

  function setSession(payload: { accessToken: string; refreshToken?: string | null; user?: UserProfile | null }) {
    accessToken.value = payload.accessToken
    hasRefreshCookie.value = true
    clearStoredRefreshToken()
    if (payload.user !== undefined) user.value = payload.user
  }

  function clearSession() {
    accessToken.value = null
    hasRefreshCookie.value = false
    clearStoredRefreshToken()
    user.value = null
  }

  async function tryRestoreSession(): Promise<boolean> {
    if (accessToken.value) return true
    restoring.value = true
    try {
      const res = await fetch(`${appEnv.apiBaseUrl}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
      })
      if (!res.ok) throw new Error(`refresh failed: ${res.status}`)
      const json = await res.json()
      const data = json.data
      setSession({
        accessToken: data.access_token,
        user: data.user ?? null,
      })
      await fetchPermissions()
      return true
    } catch (err) {
      devWarn('auth.refresh_failed', toLogContext(err))
      clearSession()
      return false
    } finally {
      restoring.value = false
    }
  }

  async function fetchPermissions() {
    if (!accessToken.value) return
    try {
      const res = await axios.get(`${appEnv.apiBaseUrl}/me/permissions`, {
        headers: { Authorization: `Bearer ${accessToken.value}` },
        timeout: 10000,
      })
      const perms = res.data?.data?.app_permissions
      if (Array.isArray(perms) && user.value) {
        // Mutate in place to avoid triggering watchers that spread would cause
        ;(user.value as Record<string, unknown>).permissions = perms
      }
    } catch (err) {
      devWarn('auth.permissions_failed_fallback_used', toLogContext(err))
    }
  }

  return { accessToken, refreshToken: hasRefreshCookie, user, restoring, isAuthenticated, defaultHomeRoute, setSession, clearSession, tryRestoreSession, fetchPermissions }
})
