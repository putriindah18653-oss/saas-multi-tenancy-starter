import { tenantApi } from '@/services/api'
import type { Envelope, TenantMember } from '@/contracts/api'

export type { TenantMember }

export const tenantUsersService = {
  me() {
    return tenantApi.get<Envelope<TenantMember>>('/tenant/me')
  },
  list() {
    return tenantApi.get<Envelope<TenantMember[]>>('/tenant/users')
  },
  invite(payload: { name: string; email: string; role: string }) {
    return tenantApi.post<Envelope<{ member: TenantMember; temporary_password: string }>>('/tenant/users/invite', payload)
  },
  changeRole(id: string, role: string) {
    return tenantApi.patch<Envelope<TenantMember>>(`/tenant/users/${id}/role`, { role })
  },
  remove(id: string) {
    return tenantApi.delete<Envelope<{ removed: boolean }>>(`/tenant/users/${id}`)
  },
}
