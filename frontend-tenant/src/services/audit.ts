import { authApi, tenantApi } from '@/services/api'

export type AuditEntry = {
  id: string
  actor_user_id?: string
  tenant_id?: string
  action: string
  resource_type: string
  resource_id?: string
  metadata: Record<string, unknown>
  ip_address: string
  user_agent: string
  created_at: string
}

type Envelope<T> = { success: boolean; data: T }

export const auditService = {
  tenant(limit = 100) {
    return tenantApi.get<Envelope<AuditEntry[]>>(`/tenant/audit?limit=${limit}`)
  },
  app(limit = 100) {
    return authApi.get<Envelope<AuditEntry[]>>(`/app/audit?limit=${limit}`)
  },
}
