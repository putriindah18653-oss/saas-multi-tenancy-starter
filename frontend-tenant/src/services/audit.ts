import { tenantApi } from '@/services/api'
import type { AuditEntry, Envelope } from '@/contracts/api'

export type { AuditEntry }

export const auditService = {
  tenant(limit = 100) {
    return tenantApi.get<Envelope<AuditEntry[]>>(`/tenant/audit?limit=${limit}`)
  },
}
