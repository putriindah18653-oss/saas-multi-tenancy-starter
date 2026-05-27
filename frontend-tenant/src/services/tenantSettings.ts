import { tenantApi } from '@/services/api'

export type TenantSettings = {
  tenant_id: string
  display_name: string
  logo_url: string
  timezone: string
  locale: string
  currency: string
  metadata: Record<string, unknown>
}

type Envelope<T> = { success: boolean; data: T }

export const tenantSettingsService = {
  get() {
    return tenantApi.get<Envelope<TenantSettings>>('/tenant/settings/')
  },
  update(payload: Partial<TenantSettings>) {
    return tenantApi.patch<Envelope<TenantSettings>>('/tenant/settings/', payload)
  },
}
