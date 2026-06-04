import { authApi } from '@/services/api'
import type { UserProfile } from '@/stores/auth'

type LoginPayload = { email: string; password: string }
export type UpdateProfilePayload = {
  name: string
  phone?: string
  address?: string
  avatar_url?: string
  bio?: string
}

type AuthResponse = {
  data: {
    user: UserProfile
    access_token: string
    refresh_token?: string
  }
}

export const authService = {
  login(payload: LoginPayload) {
    return authApi.post<AuthResponse>('/auth/login', payload)
  },
  refreshToken() {
    return authApi.post<{ data: { user?: UserProfile; access_token: string; refresh_token?: string } }>('/auth/refresh', {})
  },
  logout(_refresh_token?: string | null) {
    return authApi.post('/auth/logout', {})
  },
  changePassword(payload: { current_password: string; new_password: string }) {
    return authApi.post<{ data: { changed: boolean } }>('/me/password', payload)
  },
  me() {
    return authApi.get<{ data: UserProfile }>('/me')
  },
  updateProfile(payload: UpdateProfilePayload) {
    return authApi.patch<{ data: UserProfile }>('/me/profile', payload)
  },
  uploadAvatar(file: File) {
    const form = new FormData()
    form.append('image', file)
    return authApi.post<{ data: { filename: string; url_path: string } }>('/me/uploads/avatar', form, {
      timeout: 60000,
    })
  },
  permissions() {
    return authApi.get<{ data: { app_permissions: string[] } }>('/me/permissions')
  },
}
