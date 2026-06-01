import { tenantApi } from '@/services/api'
import type { Envelope, TenantSettings } from '@/contracts/api'

export type { TenantSettings }

export const tenantSettingsService = {
  get() {
    return tenantApi.get<Envelope<TenantSettings>>('/tenant/settings/')
  },
  update(payload: Partial<TenantSettings>) {
    return tenantApi.patch<Envelope<TenantSettings>>('/tenant/settings/', payload)
  },
}
