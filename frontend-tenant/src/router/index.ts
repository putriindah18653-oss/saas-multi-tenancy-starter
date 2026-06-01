import { createRouter, createWebHistory, type NavigationGuardNext, type RouteLocationNormalized } from 'vue-router'
import AuthLayout from '@/layouts/AuthLayout.vue'
import TenantLayout from '@/layouts/TenantLayout.vue'
import ForbiddenPage from '@/pages/errors/ForbiddenPage.vue'
import TenantNotFoundPage from '@/pages/errors/TenantNotFoundPage.vue'
import { useAuthStore } from '@/stores/auth'
import { useTenantStore } from '@/stores/tenant'
import { canTenant, type TenantPermission } from '@/services/rbac'
import { hydrateSession } from '@/services/session'

// Lazy-loaded page components for code splitting
const LoginPage = () => import('@/pages/auth/LoginPage.vue')
const ChangePasswordPage = () => import('@/pages/auth/ChangePasswordPage.vue')
const TenantDashboard = () => import('@/pages/tenant/TenantDashboard.vue')
const TenantUsersPage = () => import('@/pages/tenant/users/TenantUsersPage.vue')
const TenantSettingsPage = () => import('@/pages/tenant/TenantSettingsPage.vue')
const ProfileSettingsPage = () => import('@/pages/tenant/ProfileSettingsPage.vue')
const TenantAuditPage = () => import('@/pages/tenant/TenantAuditPage.vue')

const routes = [
  { path: '/', redirect: '/auth/login' },
  {
    path: '/auth',
    component: AuthLayout,
    children: [
      { path: 'login', name: 'auth-login', component: LoginPage, meta: { guestOnly: true } },
      { path: 'change-password', name: 'auth-change-password', component: ChangePasswordPage, meta: { requiresAuth: true } },
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

// Update document.title on navigation
router.afterEach((to) => {
  document.title = (to.meta.title as string) || 'PortalOnline'
})
