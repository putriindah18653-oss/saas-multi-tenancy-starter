import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

const get = vi.fn()

vi.mock('@/services/tenants', () => ({
  tenantsService: {
    get: (...args: unknown[]) => get(...args),
    update: vi.fn(),
    remove: vi.fn(),
  },
}))

// Owner has full management permission so the status <select> is enabled.
vi.mock('@/services/rbac', () => ({ canOwner: () => true }))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'tenant-1' } }),
  useRouter: () => ({ push: vi.fn() }),
  RouterLink: { template: '<a><slot /></a>' },
}))

import TenantDetail from './TenantDetail.vue'

describe('owner TenantDetail status contract', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    get.mockResolvedValue({ data: { data: { id: 'tenant-1', name: 'Acme', slug: 'acme', status: 'active' } } })
  })

  it('only offers active and suspended in the status dropdown', async () => {
    const wrapper = mount(TenantDetail, {
      global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    const options = wrapper.findAll('#tenant-status option').map((o) => o.attributes('value'))
    expect(options).toEqual(['active', 'suspended'])
    // Soft delete stays a dedicated button, never a status option.
    expect(options).not.toContain('deleted')
    expect(options).not.toContain('inactive')
    expect(wrapper.text()).toContain('Soft delete')
  })
})
