import { api } from '@/services/api'

export type TenantMember = {
  id: string
  user_id: string
  name: string
  email: string
  tenant_id: string
  role: string
  is_active: boolean
}

type Envelope<T> = { success: boolean; data: T }

export const tenantUsersService = {
  me() {
    return api.get<Envelope<TenantMember>>('/tenant/me')
  },
  list() {
    return api.get<Envelope<TenantMember[]>>('/tenant/users')
  },
  invite(payload: { name: string; email: string; role: string }) {
    return api.post<Envelope<{ member: TenantMember; temporary_password: string }>>('/tenant/users/invite', payload)
  },
  changeRole(id: string, role: string) {
    return api.patch<Envelope<TenantMember>>(`/tenant/users/${id}/role`, { role })
  },
  remove(id: string) {
    return api.delete<Envelope<{ removed: boolean }>>(`/tenant/users/${id}`)
  },
}
