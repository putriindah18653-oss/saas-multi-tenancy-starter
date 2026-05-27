import { api } from '@/services/api'
import type { UserProfile } from '@/stores/auth'
import type { TenantMembership } from '@/stores/tenant'

type LoginPayload = { email: string; password: string }
type RegisterOwnerPayload = { full_name: string; email: string; password: string }

type AuthResponse = {
  data: {
    user: UserProfile
    access_token: string
    refresh_token: string
    tenant_memberships: TenantMembership[]
  }
}

export const authService = {
  registerOwner(payload: RegisterOwnerPayload) {
    return api.post<AuthResponse>('/auth/register-owner', payload)
  },
  login(payload: LoginPayload) {
    return api.post<AuthResponse>('/auth/login', payload)
  },
  refreshToken(refresh_token: string) {
    return api.post<{ data: { access_token: string; refresh_token?: string } }>('/auth/refresh', { refresh_token })
  },
  logout() {
    return api.post('/auth/logout')
  },
  me() {
    return api.get<{ data: UserProfile & { tenant_memberships?: TenantMembership[] } }>('/me')
  },
}
