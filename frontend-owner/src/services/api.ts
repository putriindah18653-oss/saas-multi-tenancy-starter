import axios, { AxiosError, type AxiosInstance } from 'axios'
import { appEnv } from '@/app/env'
import { useAuthStore } from '@/stores/auth'
import type { ErrorEnvelope } from '@/contracts/api'

/**
 * Creates an Axios instance with auth interceptors, token refresh, and
 * response logging. Uses lazy `useAuthStore()` calls inside interceptors
 * (the standard Pinia pattern) so the store is always available.
 */
function createApiClient(): AxiosInstance {
  const instance = axios.create({
    baseURL: appEnv.apiBaseUrl,
    timeout: 15000,
  })

  // Attach access token to every request
  instance.interceptors.request.use((config) => {
    const auth = useAuthStore()
    config.headers = config.headers || {}
    if (auth.accessToken) {
      config.headers.Authorization = `Bearer ${auth.accessToken}`
    }
    config.headers['X-Request-Start'] = String(Date.now())
    return config
  })

  // Log all responses for observability
  instance.interceptors.response.use(
    (response) => {
      const start = Number(response.config.headers['X-Request-Start'])
      const duration = start ? Date.now() - start : -1
      if (import.meta.env.DEV) {
        console.debug(`[api] ${response.config.method?.toUpperCase()} ${response.config.url} ${response.status} ${duration}ms`)
      }
      return response
    },
    (error: AxiosError<ErrorEnvelope>) => {
      const start = Number(error.config?.headers?.['X-Request-Start'])
      const duration = start ? Date.now() - start : -1
      console.error(
        `[api] ${error.config?.method?.toUpperCase()} ${error.config?.url} ${error.response?.status ?? 'NETWORK'} ${duration}ms`,
        error.response?.data ?? error.message,
      )
      return Promise.reject(error)
    },
  )

  // Handle 401 token refresh and 403 password-change redirect
  let refreshPromise: Promise<string | null> | null = null

  function refreshAccessToken(): Promise<string | null> {
    const auth = useAuthStore()
    if (!auth.refreshToken) return Promise.resolve(null)

    if (!refreshPromise) {
      refreshPromise = instance
        .post('/auth/refresh', { refresh_token: auth.refreshToken })
        .then((response) => {
          const data = response.data?.data
          if (!data?.access_token || typeof data.access_token !== 'string') {
            throw new Error('invalid auth response')
          }
          auth.setSession({
            accessToken: data.access_token,
            refreshToken: data.refresh_token,
            user: data.user ?? auth.user,
          })
          return data.access_token as string
        })
        .catch((err) => {
          console.error('[api] token refresh failed, clearing session', err)
          auth.clearSession()
          return null
        })
        .finally(() => {
          refreshPromise = null
        })
    }

    return refreshPromise
  }

  instance.interceptors.response.use(
    (response) => response,
    async (error: AxiosError<ErrorEnvelope>) => {
      const status = error.response?.status
      const original = error.config as Record<string, unknown> & { _retry?: boolean }
      const errData = error.response?.data as ErrorEnvelope | undefined

      // 403 with password_change_required → force redirect
      if (status === 403 && errData?.error?.code === 'password_change_required') {
        if (window.location.pathname !== '/auth/change-password') {
          window.location.assign('/auth/change-password')
        }
        return Promise.reject(error)
      }

      // 401 with refresh token → attempt silent refresh
      const isAuthRefreshUrl = original.url ? String(original.url).includes('/auth/refresh') : false
      if (status === 401 && !original._retry && !isAuthRefreshUrl) {
        const auth = useAuthStore()
        if (auth.accessToken && auth.refreshToken) {
          original._retry = true
          const newToken = await refreshAccessToken()
          if (newToken) {
            original.headers = original.headers || {}
            ;(original.headers as Record<string, string>).Authorization = `Bearer ${newToken}`
            return instance(original)
          }
        }
      }

      // Clear session on final 401
      if (status === 401) {
        const auth = useAuthStore()
        auth.clearSession()
      }

      return Promise.reject(error)
    },
  )

  return instance
}

export const authApi = createApiClient()
export const ownerApi = createApiClient()

/**
 * Backward compatibility while services migrate.
 * @deprecated Prefer `authApi` or `ownerApi`.
 */
export const api = authApi
