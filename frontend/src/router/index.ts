import { createRouter, createWebHistory, type NavigationGuardNext, type RouteLocationNormalized } from 'vue-router'
import AuthLayout from '@/layouts/AuthLayout.vue'
import AppAdminLayout from '@/layouts/AppAdminLayout.vue'
import TenantLayout from '@/layouts/TenantLayout.vue'
import LoginPage from '@/pages/auth/LoginPage.vue'
import RegisterOwnerPage from '@/pages/auth/RegisterOwnerPage.vue'
import AppDashboard from '@/pages/app/AppDashboard.vue'
import TenantList from '@/pages/app/TenantList.vue'
import TenantCreate from '@/pages/app/TenantCreate.vue'
import TenantDetail from '@/pages/app/TenantDetail.vue'
import TenantDashboard from '@/pages/tenant/TenantDashboard.vue'
import TenantUsersPage from '@/pages/tenant/users/TenantUsersPage.vue'
import { useAuthStore } from '@/stores/auth'

const routes = [
  { path: '/', redirect: '/auth/login' },
  {
    path: '/auth',
    component: AuthLayout,
    children: [
      { path: 'login', name: 'auth-login', component: LoginPage, meta: { guestOnly: true } },
      { path: 'register-owner', name: 'auth-register-owner', component: RegisterOwnerPage, meta: { guestOnly: true } },
    ],
  },
  {
    path: '/app',
    component: AppAdminLayout,
    meta: { requiresAuth: true, scope: 'app' },
    children: [
      { path: '', name: 'app-home', component: AppDashboard },
      { path: 'tenants', name: 'app-tenant-list', component: TenantList },
      { path: 'tenants/create', name: 'app-tenant-create', component: TenantCreate },
      { path: 'tenants/:id', name: 'app-tenant-detail', component: TenantDetail },
    ],
  },
  {
    path: '/tenant',
    component: TenantLayout,
    meta: { requiresAuth: true, scope: 'tenant' },
    children: [
      { path: '', name: 'tenant-home', component: TenantDashboard },
      { path: 'users', name: 'tenant-users', component: TenantUsersPage },
    ],
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to: RouteLocationNormalized, _from: RouteLocationNormalized, next: NavigationGuardNext) => {
  const auth = useAuthStore()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    next({ name: 'auth-login' })
    return
  }

  if (to.meta.guestOnly && auth.isAuthenticated) {
    next({ name: auth.defaultHomeRoute })
    return
  }

  next()
})
