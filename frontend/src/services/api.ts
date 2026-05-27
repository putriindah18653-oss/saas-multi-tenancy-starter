import axios, { AxiosError } from 'axios'
import { appEnv } from '@/app/env'
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'

export const api = axios.create({
  baseURL: appEnv.apiBaseUrl,
  timeout: 15000,
})

api.interceptors.request.use((config) => {
  const auth = useAuthStore()
  const tenant = useTenantStore()

  config.headers = config.headers || {}

  if (auth.accessToken) {
    config.headers.Authorization = `Bearer ${auth.accessToken}`
  }

  if (tenant.selectedTenantId) {
    config.headers[appEnv.tenantHeader] = tenant.selectedTenantId
  }

  return config
})

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    const status = error.response?.status
    if (status === 401) {
      const auth = useAuthStore()
      const tenant = useTenantStore()
      auth.clearSession()
      tenant.clearTenant()
    }
    return Promise.reject(error)
  },
)
