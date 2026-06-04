import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/pages/auth/LoginPage.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/auth/ChangePasswordPage.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/layouts/AuthLayout.vue', () => ({ default: { template: '<router-view />' } }))
vi.mock('@/layouts/AppAdminLayout.vue', () => ({ default: { template: '<router-view />' } }))
vi.mock('@/pages/app/AppDashboard.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/app/TenantList.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/app/TenantCreate.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/app/TenantDetail.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/app/AppAuditPage.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/app/ProfileSettingsPage.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/app/CompanySettingsPage.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/errors/ForbiddenPage.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/pages/app/UnderDevelopmentPage.vue', () => ({ default: { template: '<div />' } }))
vi.mock('@/services/rbac', () => ({ canOwner: () => true }))

describe('owner router', () => {
  beforeEach(() => {
    vi.resetModules()
    history.replaceState({}, '', '/')
    sessionStorage.clear()
    setActivePinia(createPinia())
  })

  it('tries HttpOnly refresh-cookie session before showing guest-only login', async () => {
    const { router } = await import('./index')
    const { useAuthStore } = await import('@/stores/auth')
    const auth = useAuthStore()
    const restore = vi.spyOn(auth, 'tryRestoreSession').mockImplementation(async () => {
      auth.setSession({
        accessToken: 'new-access',
        user: { id: 'user-1', email: 'owner@app.local', permissions: ['app.tenants.read'] },
      })
      return true
    })

    await router.push('/auth/login')
    await router.isReady()

    expect(restore).toHaveBeenCalled()
    expect(router.currentRoute.value.name).toBe('app-home')
  })
})