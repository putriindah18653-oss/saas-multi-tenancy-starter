import { createRouter, createWebHistory, type NavigationGuardNext, type RouteLocationNormalized } from 'vue-router'
import AuthLayout from '@/layouts/AuthLayout.vue'
import TenantLayout from '@/layouts/TenantLayout.vue'
import LoginPage from '@/pages/auth/LoginPage.vue'
import ChangePasswordPage from '@/pages/auth/ChangePasswordPage.vue'
import TenantDashboard from '@/pages/tenant/TenantDashboard.vue'
import TenantUsersPage from '@/pages/tenant/users/TenantUsersPage.vue'
import TenantSettingsPage from '@/pages/tenant/TenantSettingsPage.vue'
import ProfileSettingsPage from '@/pages/tenant/ProfileSettingsPage.vue'
import TenantAuditPage from '@/pages/tenant/TenantAuditPage.vue'
import ForbiddenPage from '@/pages/errors/ForbiddenPage.vue'
import TenantNotFoundPage from '@/pages/errors/TenantNotFoundPage.vue'
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'
import { canTenant, type TenantPermission } from '@/services/rbac'
import { hydrateSession } from '@/services/session'

const routes = [
  { path: '/', redirect: '/auth/login' },
  {
    path: '/auth',
    component: AuthLayout,
    children: [
      { path: 'login', name: 'auth-login', component: LoginPage, meta: { guestOnly: true } },
      { path: 'change-password', name: 'auth-change-password', component: ChangePasswordPage, meta: { requiresAuth: true } },
      { path: 'register-owner', redirect: { name: 'auth-login' } },
    ],
  },
  { path: '/forbidden', name: 'forbidden', component: ForbiddenPage, meta: { requiresAuth: true } },
  { path: '/tenant-not-found', name: 'tenant-not-found', component: TenantNotFoundPage, meta: { requiresAuth: true } },
  {
    path: '/tenant',
    component: TenantLayout,
    meta: { requiresAuth: true, scope: 'tenant', permission: 'tenant.dashboard.read' },
    children: [
      { path: '', name: 'tenant-home', component: TenantDashboard, meta: { title: 'Dashboard', requiresAuth: true, scope: 'tenant', permission: 'tenant.dashboard.read' } },
      { path: 'profile', name: 'tenant-profile', component: ProfileSettingsPage, meta: { title: 'Profile', requiresAuth: true, scope: 'tenant', permission: 'tenant.dashboard.read' } },
      { path: 'users', name: 'tenant-users', component: TenantUsersPage, meta: { title: 'Users', requiresAuth: true, scope: 'tenant', permission: 'tenant.users.read' } },
      { path: 'settings', name: 'tenant-settings', component: TenantSettingsPage, meta: { title: 'Settings', requiresAuth: true, scope: 'tenant', permission: 'tenant.settings.read' } },
      { path: 'audit', name: 'tenant-audit', component: TenantAuditPage, meta: { title: 'Audit Log', requiresAuth: true, scope: 'tenant', permission: 'tenant.audit.read' } },
    ],
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

function resolveRoutePermission(to: RouteLocationNormalized): string | undefined {
  return [...to.matched].reverse().find((record) => record.meta.permission)?.meta.permission as string | undefined
}

function resolveRouteScope(to: RouteLocationNormalized): string | undefined {
  return [...to.matched].reverse().find((record) => record.meta.scope)?.meta.scope as string | undefined
}

router.beforeEach(async (to: RouteLocationNormalized, _from: RouteLocationNormalized, next: NavigationGuardNext) => {
  const auth = useAuthStore()
  const tenant = useTenantStore()

  if (!auth.hydrated && (to.meta.requiresAuth || auth.refreshToken)) {
    await hydrateSession()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    next({ name: 'auth-login' })
    return
  }

  if (to.meta.guestOnly && auth.isAuthenticated) {
    next({ name: auth.defaultHomeRoute })
    return
  }

  if (auth.user?.must_change_password && to.name !== 'auth-change-password') {
    next({ name: 'auth-change-password' })
    return
  }

  const scope = resolveRouteScope(to)
  const permission = resolveRoutePermission(to)

  if (scope === 'tenant') {
    if (!tenant.selectedTenantId || !tenant.selectedMembership) {
      next({ name: 'tenant-not-found' })
      return
    }

    if (!permission || !canTenant(tenant.selectedMembership, permission as TenantPermission)) {
      next({ name: 'forbidden' })
      return
    }
  }

  next()
})
