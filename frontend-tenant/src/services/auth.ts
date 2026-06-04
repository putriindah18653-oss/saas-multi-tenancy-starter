import { authApi } from '@/services/api'
import type { UserProfile, TenantMembership, Envelope } from '@/contracts/api'

type LoginPayload = { email: string; password: string }
export type UpdateProfilePayload = {
  name: string
  phone?: string
  address?: string
  avatar_url?: string
  bio?: string
}

export const authService = {
  login(payload: LoginPayload) {
    return authApi.post<Envelope<{ user: UserProfile; access_token: string; refresh_token: string; tenant_memberships: TenantMembership[] }>>('/auth/login', payload)
  },
  refreshToken(_refresh_token?: string | null) {
    return authApi.post<{ data: { user?: UserProfile; access_token: string; refresh_token?: string } }>('/auth/refresh', {})
  },
  logout(_refresh_token?: string | null) {
    return authApi.post('/auth/logout', {})
  },
  changePassword(payload: { current_password: string; new_password: string }) {
    return authApi.post<Envelope<{ changed: boolean }>>('/me/password', payload)
  },
  me() {
    return authApi.get<Envelope<UserProfile & { tenant_memberships?: TenantMembership[] }>>('/me')
  },
  updateProfile(payload: UpdateProfilePayload) {
    return authApi.patch<Envelope<UserProfile>>('/me/profile', payload)
  },
  uploadAvatar(file: File) {
    const form = new FormData()
    form.append('image', file)
    return authApi.post<{ data: { filename: string; url_path: string } }>('/me/uploads/avatar', form, {
      timeout: 60000,
    })
  },
}
