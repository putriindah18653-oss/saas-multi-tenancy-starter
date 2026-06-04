import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'

let hydrationPromise: Promise<boolean> | null = null

export async function hydrateSession() {
  const auth = useAuthStore()
  const tenant = useTenantStore()

  if (auth.hydrated) return auth.isAuthenticated
  if (hydrationPromise) return hydrationPromise

  auth.setHydrationState({ hydrating: true })

  hydrationPromise = authService
    .refreshToken()
    .then(async (refreshResponse) => {
      const refreshPayload = refreshResponse.data.data
      auth.setSession({
        accessToken: refreshPayload.access_token,
        user: refreshPayload.user ?? auth.user,
      })

      const meResponse = await authService.me()
      const mePayload = meResponse.data.data
      const { tenant_memberships = [], ...user } = mePayload

      auth.setSession({
        accessToken: refreshPayload.access_token,
        refreshToken: refreshPayload.refresh_token ?? auth.refreshToken,
        user,
      })
      tenant.setMemberships(tenant_memberships)
      auth.setHydrationState({ hydrated: true, hydrating: false })

      return true
    })
    .catch((err) => {
      console.error('[session] hydration failed', err)
      auth.clearSession()
      tenant.clearTenant()
      return false
    })
    .finally(() => {
      hydrationPromise = null
    })

  return hydrationPromise
}
