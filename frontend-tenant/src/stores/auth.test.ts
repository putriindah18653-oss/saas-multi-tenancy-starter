import { setActivePinia, createPinia } from 'pinia'

describe('tenant auth store refresh-token storage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    sessionStorage.clear()
  })

  it('does not persist refresh tokens in web storage', async () => {
    const { useAuthStore } = await import('@/stores/auth')
    const auth = useAuthStore()

    auth.setSession({ accessToken: 'access', refreshToken: 'refresh-1', user: { id: 'u1', email: 'user@app.local' } })

    expect(auth.refreshToken).toBe(true)
    expect(sessionStorage.getItem('refresh_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('clears legacy refresh tokens from web storage on clearSession', async () => {
    sessionStorage.setItem('refresh_token', 'legacy-session')
    localStorage.setItem('refresh_token', 'legacy-local')
    const { useAuthStore } = await import('@/stores/auth')
    const auth = useAuthStore()

    auth.setSession({ accessToken: 'access', refreshToken: 'refresh-1' })
    auth.clearSession()

    expect(auth.refreshToken).toBe(false)
    expect(sessionStorage.getItem('refresh_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })
})
