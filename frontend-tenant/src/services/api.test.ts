import axios from 'axios'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('axios', () => ({
  default: {
    create: vi.fn(),
  },
}))

describe('tenant API refresh', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    localStorage.clear()
    sessionStorage.clear()
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('clears auth and tenant when refresh payload has no access_token', async () => {
    const requestHandlers: Array<(config: any) => any> = []
    const responseHandlers: Array<(error: any) => any> = []
    const authApiInstance = createInstanceMock({ data: { data: {} } })
    const tenantApiInstance = createInstanceMock(null)

    vi.mocked(axios.create)
      .mockReturnValueOnce(authApiInstance as any)
      .mockReturnValueOnce(tenantApiInstance as any)

    function createInstanceMock(refreshResponse: unknown) {
      const instance = vi.fn((config: any) => Promise.resolve({ data: 'retried', config }))
      Object.assign(instance, {
        interceptors: {
          request: { use: vi.fn((handler) => requestHandlers.push(handler)) },
          response: { use: vi.fn((_ok, errorHandler) => responseHandlers.push(errorHandler)) },
        },
        post: vi.fn(() => Promise.resolve(refreshResponse)),
      })
      return instance
    }

    await import('./api')
    const { useAuthStore } = await import('@/stores/auth')
    const { useTenantStore } = await import('@/stores/tenant')
    const auth = useAuthStore()
    const tenant = useTenantStore()

    auth.setSession({ accessToken: 'access', refreshToken: 'refresh', user: { id: 'user-1', email: 'user@app.local' } })
    tenant.setMemberships([{ tenant_id: 'tenant-1', role: 'admin' }])

    await expect(responseHandlers[0]({
      response: { status: 401 },
      config: { url: '/tenant/settings', headers: {} },
    })).rejects.toBeTruthy()

    expect(auth.accessToken).toBeNull()
    expect(auth.refreshToken).toBe(false)
    expect(tenant.selectedTenantId).toBeNull()
    expect(authApiInstance.post).toHaveBeenCalledWith('/auth/refresh', {})
  })
})