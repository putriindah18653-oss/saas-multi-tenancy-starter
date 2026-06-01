import { ownerApi } from '@/services/api'
import type { AppCompanySettings, Envelope, UploadResult } from '@/contracts/api'

export const defaultAppCompanySettings: AppCompanySettings = {
  display_name: 'PortalOnline',
  legal_name: '',
  logo_url: '',
  website_url: '',
  support_email: '',
  support_phone: '',
  timezone: 'Asia/Jakarta',
  locale: 'id-ID',
  currency: 'IDR',
  address: '',
  metadata: {},
}

export const appSettingsService = {
  get() {
    return ownerApi.get<Envelope<AppCompanySettings>>('/app/settings/')
  },
  update(payload: Partial<AppCompanySettings>) {
    return ownerApi.patch<Envelope<AppCompanySettings>>('/app/settings/', payload)
  },
  uploadLogo(file: File) {
    const form = new FormData()
    form.append('image', file)
    return ownerApi.post<Envelope<UploadResult>>('/app/uploads/logo', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
    })
  },
}
