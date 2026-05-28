import axios, { AxiosError, type AxiosInstance } from 'axios'
import { appEnv } from '@/app/env'
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'

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

function attachTenantContext(instance: AxiosInstance) {
  instance.interceptors.request.use((config) => {
    const tenant = useTenantStore()
    config.headers = config.headers || {}

    if (tenant.selectedTenantId) {
      config.headers[appEnv.tenantHeader] = tenant.selectedTenantId
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
          refreshToken: data.refresh_token ?? auth.refreshToken,
          user: data.user ?? auth.user,
        })
        return data.access_token as string
      })
      .catch(() => {
        const tenant = useTenantStore()
        auth.clearSession()
        tenant.clearTenant()
        return null
      })
      .finally(() => {
        refreshPromise = null
      })
  }

  return refreshPromise
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

      if (status === 401 && !original?._retry && !original?.url?.includes('/auth/refresh')) {
        const auth = useAuthStore()
        if (auth.accessToken && auth.refreshToken) {
          original._retry = true
          const token = await refreshAccessToken()
          if (token) {
            original.headers = original.headers || {}
            original.headers.Authorization = `Bearer ${token}`
            return instance(original)
          }
        }
      }

      if (status === 401) {
        const auth = useAuthStore()
        const tenant = useTenantStore()
        auth.clearSession()
        tenant.clearTenant()
      }
      return Promise.reject(error)
    },
  )
}

function createApiClient({ includeTenant = false }: { includeTenant?: boolean } = {}) {
  const instance = axios.create({
    baseURL: appEnv.apiBaseUrl,
    timeout: 15000,
  })

  attachAuth(instance)
  if (includeTenant) attachTenantContext(instance)
  attachUnauthorizedHandler(instance)

  return instance
}

export const authApi = createApiClient()
export const tenantApi = createApiClient({ includeTenant: true })

// Backward compatibility while services migrate. Prefer authApi or tenantApi.
export const api = authApi
