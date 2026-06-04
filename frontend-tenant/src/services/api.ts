import axios, { AxiosError, type AxiosInstance } from 'axios'
import { appEnv } from '@/app/env'
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'
import type { ErrorEnvelope } from '@/contracts/api'
import { devWarn, toLogContext } from '@/utils/devLogger'

function attachAuth(instance: AxiosInstance) {
  instance.interceptors.request.use((config) => {
    const auth = useAuthStore()
    config.headers = config.headers || {}
    config.headers['X-Request-Start'] = String(Date.now())

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

  if (!refreshPromise) {
    refreshPromise = authApi
      .post('/auth/refresh', {})
      .then((response) => {
        const data = response.data?.data
        if (!data?.access_token || typeof data.access_token !== 'string') {
          throw new Error('invalid auth response')
        }
        auth.setSession({
          accessToken: data.access_token,
          user: data.user ?? auth.user,
        })
        return data.access_token as string
      })
      .catch((err) => {
        console.error('[api] token refresh failed', err)
        devWarn('auth.refresh_failed', toLogContext(err))
        const tenant = useTenantStore()
        auth.clearSession()
        tenant.clearTenant()
        refreshPromise = null
        return null
      })
      .finally(() => {
        if (refreshPromise) refreshPromise = null
      })
  }

  return refreshPromise
}

function attachUnauthorizedHandler(instance: AxiosInstance) {
  instance.interceptors.response.use(
    (response) => {
      // Log response timing in dev
      const start = Number(response.config.headers['X-Request-Start'])
      const duration = start ? Date.now() - start : -1
      if (import.meta.env.DEV) {
        console.debug(`[api] ${response.config.method?.toUpperCase()} ${response.config.url} ${response.status} ${duration}ms`)
      }
      return response
    },
    async (error: AxiosError<ErrorEnvelope>) => {
      const start = Number(error.config?.headers?.['X-Request-Start'])
      const duration = start ? Date.now() - start : -1
      console.error(
        `[api] ${error.config?.method?.toUpperCase()} ${error.config?.url} ${error.response?.status ?? 'NETWORK'} ${duration}ms`,
        error.response?.data ?? error.message,
      )

      const status = error.response?.status
      const original = error.config as Record<string, unknown> & { _retry?: boolean }
      const errData = error.response?.data as ErrorEnvelope | undefined

      if (status === 403 && errData?.error?.code === 'password_change_required') {
        if (window.location.pathname !== '/auth/change-password') window.location.assign('/auth/change-password')
        return Promise.reject(error)
      }

      if (status === 401 && !original._retry && !(original.url && String(original.url).includes('/auth/refresh'))) {
        const auth = useAuthStore()
        if (auth.accessToken) {
          original._retry = true
          const token = await refreshAccessToken()
          if (token) {
            original.headers = original.headers || {}
            ;(original.headers as Record<string, string>).Authorization = `Bearer ${token}`
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
    withCredentials: true,
  })

  attachAuth(instance)
  if (includeTenant) attachTenantContext(instance)
  attachUnauthorizedHandler(instance)

  return instance
}

export const authApi = createApiClient()
export const tenantApi = createApiClient({ includeTenant: true })

/** @deprecated Use {@link authApi} or {@link tenantApi} directly. */
export const api = authApi
