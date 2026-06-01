import { ownerApi } from '@/services/api'
import type { Envelope, Tenant } from '@/contracts/api'

export const tenantsService = {
  list() {
    return ownerApi.get<Envelope<Tenant[]>>('/app/tenants/')
  },
  get(id: string) {
    return ownerApi.get<Envelope<Tenant>>(`/app/tenants/${id}`)
  },
  create(payload: { name: string; slug?: string }) {
    return ownerApi.post<Envelope<Tenant>>('/app/tenants/', payload)
  },
  update(id: string, payload: { name?: string; status?: string }) {
    return ownerApi.patch<Envelope<Tenant>>(`/app/tenants/${id}`, payload)
  },
  remove(id: string) {
    return ownerApi.delete<Envelope<{ deleted: boolean }>>(`/app/tenants/${id}`)
  },
}
