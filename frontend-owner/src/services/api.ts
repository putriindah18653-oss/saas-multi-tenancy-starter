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

function attachUnauthorizedHandler(instance: AxiosInstance) {
  instance.interceptors.response.use(
    (response) => response,
    (error: AxiosError) => {
      const status = error.response?.status
      if (status === 401) {
        const auth = useAuthStore()
        auth.clearSession()
      }
      if (status === 403 && (error.response?.data as any)?.error?.code === 'password_change_required') {
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
