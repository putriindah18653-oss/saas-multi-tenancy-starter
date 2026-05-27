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

function attachUnauthorizedHandler(instance: AxiosInstance) {
  instance.interceptors.response.use(
    (response) => response,
    (error: AxiosError) => {
      const status = error.response?.status
      if (status === 401) {
        const auth = useAuthStore()
        const tenant = useTenantStore()
        auth.clearSession()
        tenant.clearTenant()
      }
      if (status === 403 && (error.response?.data as any)?.error?.code === 'password_change_required') {
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
