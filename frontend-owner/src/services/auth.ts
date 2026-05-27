import { authApi } from '@/services/api'
import type { UserProfile } from '@/stores/auth'

type LoginPayload = { email: string; password: string }

type AuthResponse = {
  data: {
    user: UserProfile
    access_token: string
    refresh_token: string
  }
}

export const authService = {
  login(payload: LoginPayload) {
    return authApi.post<AuthResponse>('/auth/login', payload)
  },
  refreshToken(refresh_token: string) {
    return authApi.post<{ data: { access_token: string; refresh_token?: string } }>('/auth/refresh', { refresh_token })
  },
  logout() {
    return authApi.post('/auth/logout')
  },
  me() {
    return authApi.get<{ data: UserProfile }>('/me')
  },
}
