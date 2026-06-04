import { createRouter, createWebHistory, type NavigationGuardNext, type RouteLocationNormalized } from 'vue-router'
import AuthLayout from '@/layouts/AuthLayout.vue'
import AppAdminLayout from '@/layouts/AppAdminLayout.vue'
import ForbiddenPage from '@/pages/errors/ForbiddenPage.vue'
import { useAuthStore } from '@/stores/auth'
import { canOwner, type OwnerPermission } from '@/services/rbac'

// Lazy-loaded page components for code splitting
const LoginPage = () => import('@/pages/auth/LoginPage.vue')
const ChangePasswordPage = () => import('@/pages/auth/ChangePasswordPage.vue')
const AppDashboard = () => import('@/pages/app/AppDashboard.vue')
const TenantList = () => import('@/pages/app/TenantList.vue')
const TenantCreate = () => import('@/pages/app/TenantCreate.vue')
const TenantDetail = () => import('@/pages/app/TenantDetail.vue')
const AppAuditPage = () => import('@/pages/app/AppAuditPage.vue')
const ProfileSettingsPage = () => import('@/pages/app/ProfileSettingsPage.vue')
const CompanySettingsPage = () => import('@/pages/app/CompanySettingsPage.vue')
const UnderDevelopmentPage = () => import('@/pages/app/UnderDevelopmentPage.vue')

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
      { path: '', name: 'app-home', component: AppDashboard, meta: { title: 'Dashboard', requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
      { path: 'tenants', name: 'app-tenant-list', component: TenantList, meta: { title: 'Customers', requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
      { path: 'tenants/create', name: 'app-tenant-create', component: TenantCreate, meta: { title: 'Create Customer', requiresAuth: true, scope: 'owner', permission: 'app.tenants.create' } },
      { path: 'tenants/:id', name: 'app-tenant-detail', component: TenantDetail, meta: { title: 'Customer Detail', requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
      { path: 'audit', name: 'app-audit', component: AppAuditPage, meta: { title: 'Audit Log', requiresAuth: true, scope: 'owner', permission: 'app.audit.read' } },
      { path: 'profile', name: 'app-profile', component: ProfileSettingsPage, meta: { title: 'Profile', requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
      { path: 'settings', name: 'app-settings', component: CompanySettingsPage, meta: { title: 'Company Settings', requiresAuth: true, scope: 'owner', permission: 'app.settings.read' } },
      { path: ':pathMatch(.*)*', name: 'app-wip', component: UnderDevelopmentPage, meta: { title: 'Under Development', requiresAuth: true, scope: 'owner', permission: 'app.tenants.read' } },
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

  // Restore session from HttpOnly refresh cookie only when navigating to protected routes
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    const restored = await auth.tryRestoreSession()
    if (!restored) {
      next({ name: 'auth-login' })
      return
    }
  }

  // For guest-only pages, try HttpOnly refresh cookie before showing login
  if (to.meta.guestOnly && !auth.isAuthenticated) {
    await auth.tryRestoreSession()
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

  if (scope === 'owner') {
    if (!permission || !canOwner(auth.user, permission as OwnerPermission)) {
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
