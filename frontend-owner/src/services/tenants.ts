import { ownerApi } from '@/services/api'

export type Tenant = {
  id: string
  name: string
  slug: string
  status: string
}

type Envelope<T> = { success: boolean; data: T }

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
