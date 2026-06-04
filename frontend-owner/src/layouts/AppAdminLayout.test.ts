import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import AppAdminLayout from './AppAdminLayout.vue'
import { useAuthStore } from '@/stores/auth'

vi.mock('@/components/navigation/TopNav.vue', () => ({ default: { props: ['title'], template: '<header data-test="topnav">{{ title }}</header>' } }))
vi.mock('@/components/navigation/AppSidebar.vue', () => ({
  default: {
    props: ['items'],
    template: '<aside data-test="sidebar">{{ JSON.stringify(items) }}</aside>',
  },
}))

describe('AppAdminLayout', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  })

  it('filters owner sidebar entries by permissions', async () => {
    const auth = useAuthStore()
    auth.setSession({
      accessToken: 'access',
      user: { id: 'u1', email: 'auditor@app.local', permissions: ['app.tenants.read'] },
    })

    const router = createRouter({
      history: createWebHistory(),
      routes: [{ path: '/app', component: { template: '<div />' }, meta: { title: 'Dashboard' } }],
    })
    router.push('/app')
    await router.isReady()

    const wrapper = mount(AppAdminLayout, {
      global: {
        plugins: [router],
        stubs: { RouterView: { template: '<main />' } },
      },
    })

    const sidebarText = wrapper.get('[data-test="sidebar"]').text()
    expect(sidebarText).toContain('All Customers')
    expect(sidebarText).not.toContain('Create Customer')
    expect(sidebarText).not.toContain('Audit Log')
    expect(sidebarText).not.toContain('Settings')
  })
})
