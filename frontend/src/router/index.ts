import { createRouter, createWebHistory, type NavigationGuardNext, type RouteLocationNormalized } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const PlaceholderPublic = { template: '<div class="min-h-screen grid place-items-center text-slate-600">Auth pages siap di Task 09</div>' }
const PlaceholderApp = { template: '<div class="min-h-screen grid place-items-center text-slate-600">App admin pages siap di Task 09/10</div>' }
const PlaceholderTenant = { template: '<div class="min-h-screen grid place-items-center text-slate-600">Tenant pages siap di Task 09/11</div>' }

const routes = [
  { path: '/', redirect: '/auth/login' },
  { path: '/auth/login', name: 'auth-login', component: PlaceholderPublic, meta: { guestOnly: true } },
  { path: '/auth/register-owner', name: 'auth-register-owner', component: PlaceholderPublic, meta: { guestOnly: true } },
  { path: '/app', name: 'app-home', component: PlaceholderApp, meta: { requiresAuth: true, scope: 'app' } },
  { path: '/tenant', name: 'tenant-home', component: PlaceholderTenant, meta: { requiresAuth: true, scope: 'tenant' } },
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
