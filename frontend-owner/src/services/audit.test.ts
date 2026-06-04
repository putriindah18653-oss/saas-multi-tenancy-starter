const get = vi.fn()

vi.mock('@/services/api', () => ({
  authApi: { get },
}))

describe('auditService', () => {
  beforeEach(() => {
    get.mockResolvedValue({ data: { success: true, data: [] } })
  })

  it('passes limit through Axios params', async () => {
    const { auditService } = await import('./audit')

    await auditService.app(200)

    expect(get).toHaveBeenCalledWith('/app/audit', { params: { limit: 200 } })
  })
})