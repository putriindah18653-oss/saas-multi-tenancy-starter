import { api } from '@/services/api'

export type Tenant = {
  id: string
  name: string
  slug: string
  status: string
}

type Envelope<T> = { success: boolean; data: T }

export const tenantsService = {
  list() {
    return api.get<Envelope<Tenant[]>>('/app/tenants/')
  },
  get(id: string) {
    return api.get<Envelope<Tenant>>(`/app/tenants/${id}`)
  },
  create(payload: { name: string; slug?: string }) {
    return api.post<Envelope<Tenant>>('/app/tenants/', payload)
  },
  update(id: string, payload: { name?: string; status?: string }) {
    return api.patch<Envelope<Tenant>>(`/app/tenants/${id}`, payload)
  },
  remove(id: string) {
    return api.delete<Envelope<{ deleted: boolean }>>(`/app/tenants/${id}`)
  },
}
