import { createRouter, createWebHistory, type NavigationGuardNext, type RouteLocationNormalized } from 'vue-router'
import AuthLayout from '@/layouts/AuthLayout.vue'
import AppAdminLayout from '@/layouts/AppAdminLayout.vue'
import LoginPage from '@/pages/auth/LoginPage.vue'
import ChangePasswordPage from '@/pages/auth/ChangePasswordPage.vue'
import AppDashboard from '@/pages/app/AppDashboard.vue'
import TenantList from '@/pages/app/TenantList.vue'
import TenantCreate from '@/pages/app/TenantCreate.vue'
import TenantDetail from '@/pages/app/TenantDetail.vue'
import AppAuditPage from '@/pages/app/AppAuditPage.vue'
import ForbiddenPage from '@/pages/errors/ForbiddenPage.vue'
import { useAuthStore } from '@/stores/auth'
import { canOwner, type OwnerPermission } from '@/services/rbac'

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
  {
    path: '/app',
    component: AppAdminLayout,
    meta: { requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' },
    children: [
      { path: '', name: 'app-home', component: AppDashboard, meta: { requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
      { path: 'tenants', name: 'app-tenant-list', component: TenantList, meta: { requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
      { path: 'tenants/create', name: 'app-tenant-create', component: TenantCreate, meta: { requiresAuth: true, scope: 'owner', permission: 'app.tenants.create' } },
      { path: 'tenants/:id', name: 'app-tenant-detail', component: TenantDetail, meta: { requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
      { path: 'audit', name: 'app-audit', component: AppAuditPage, meta: { requiresAuth: true, scope: 'owner', permission: 'app.audit.read' } },
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

  // Restore session only when navigating to protected routes
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    if (auth.refreshToken) {
      const restored = await auth.tryRestoreSession()
      if (!restored) {
        next({ name: 'auth-login' })
        return
      }
    } else {
      next({ name: 'auth-login' })
      return
    }
  }

  // For guest-only pages, redirect if already authenticated (no async restore here)
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

  if (scope === 'owner') {
    if (!permission || !canOwner(auth.user, permission as OwnerPermission)) {
      next({ name: 'forbidden' })
      return
    }
  }

  next()
})
