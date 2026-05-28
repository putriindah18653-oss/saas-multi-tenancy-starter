import axios, { AxiosError, type AxiosInstance } from 'axios'
import { appEnv } from '@/app/env'
import { useAuthStore } from '@/stores/auth'

function attachAuth(instance: AxiosInstance) {
  instance.interceptors.request.use((config) => {
    const auth = useAuthStore()
    config.headers = config.headers || {}

    if (auth.accessToken) {
      config.headers.Authorization = `Bearer ${auth.accessToken}`
    }

    return config
  })
}

let refreshPromise: Promise<string | null> | null = null

function refreshAccessToken() {
  const auth = useAuthStore()
  if (!auth.refreshToken) return Promise.resolve(null)

  if (!refreshPromise) {
    refreshPromise = authApi
      .post('/auth/refresh', { refresh_token: auth.refreshToken })
      .then((response) => {
        const data = response.data.data
        auth.setSession({
          accessToken: data.access_token,
          refreshToken: data.refresh_token,
          user: data.user ?? auth.user,
        })
        return data.access_token as string
      })
      .catch(() => {
        auth.clearSession()
        return null
      })
      .finally(() => {
        refreshPromise = null
      })
  }

  return refreshPromise
}

function isAuthRefreshUrl(url: string | undefined) {
  return !!url && url.includes('/auth/refresh')
}

function shouldRefreshUnauthorized(error: AxiosError, original: any) {
  if (error.response?.status !== 401) return false
  if (original?._retry || isAuthRefreshUrl(original?.url)) return false

  const auth = useAuthStore()
  return !!auth.accessToken && !!auth.refreshToken
}

function attachUnauthorizedHandler(instance: AxiosInstance) {
  instance.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      const status = error.response?.status
      const original = error.config as any

      if (status === 403 && (error.response?.data as any)?.error?.code === 'password_change_required') {
        if (window.location.pathname !== '/auth/change-password') window.location.assign('/auth/change-password')
        return Promise.reject(error)
      }

      if (shouldRefreshUnauthorized(error, original)) {
        original._retry = true
        const token = await refreshAccessToken()
        if (token) {
          original.headers = original.headers || {}
          original.headers.Authorization = `Bearer ${token}`
          return instance(original)
        }
      }

      if (status === 401) {
        const auth = useAuthStore()
        auth.clearSession()
      }
      return Promise.reject(error)
    },
  )
}

function createApiClient() {
  const instance = axios.create({
    baseURL: appEnv.apiBaseUrl,
    timeout: 15000,
  })

  attachAuth(instance)
  attachUnauthorizedHandler(instance)

  return instance
}

export const authApi = createApiClient()
export const ownerApi = createApiClient()

// Backward compatibility while services migrate. Prefer authApi or ownerApi.
export const api = authApi
