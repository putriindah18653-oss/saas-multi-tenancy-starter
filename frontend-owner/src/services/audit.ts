import { authApi } from '@/services/api'
import type { AuditEntry, Envelope } from '@/contracts/api'

export const auditService = {
  app(limit = 100) {
    return authApi.get<Envelope<AuditEntry[]>>(`/app/audit?limit=${limit}`)
  },
}
