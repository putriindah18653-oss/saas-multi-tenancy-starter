<template>
  <div class="app-shell">
    <a href="#main-content" class="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[100] focus:rounded-[var(--radius-button)] focus:bg-[var(--accent)] focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:text-[var(--bg-app)]">Skip to main content</a>
    <AppSidebar :items="items" />
    <div class="app-main" :class="{ 'app-main--collapsed': ui.sidebarCollapsed }">
      <TopNav :title="pageTitle" />
      <main id="main-content" class="content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from '@/components/navigation/TopNav.vue'
import AppSidebar from '@/components/navigation/AppSidebar.vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const route = useRoute()
const pageTitle = computed(() => (route.meta.title as string | undefined) || 'Dashboard')

const icons = {
  dashboard: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>`,
  customers: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
  finance: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>`,
  reports: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>`,
  support: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>`,
  todo: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>`,
  audit: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>`,
  settings: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1.08-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1.08 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1.08z"/></svg>`,
  security: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>`,
  health: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="sidebar-icon"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>`,
}

const items = [
  { type: 'group', label: 'Overview' },
  { to: '/app', label: 'Dashboard', icon: icons.dashboard },

  { type: 'group', label: 'Operations' },
  {
    label: 'Customers',
    icon: icons.customers,
    mock: true,
    children: [
      { to: '/app/tenants', label: 'All Customers' },
      { to: '/app/tenants?status=trial', label: 'Trial Requests', badge: '3', mock: true },
      { to: '/app/tenants?status=active', label: 'Active', mock: true },
      { to: '/app/tenants?status=suspended', label: 'Suspended', mock: true },
    ],
  },
  {
    label: 'Finance',
    icon: icons.finance,
    mock: true,
    children: [
      { to: '/app/billing', label: 'Billing & Plans', badge: 'Mock', mock: true },
      { to: '/app/invoices', label: 'Invoices', mock: true },
      { to: '/app/payments/history', label: 'Payment History', mock: true },
    ],
  },
  {
    label: 'Reports',
    icon: icons.reports,
    mock: true,
    children: [
      { to: '/app/reports/overview', label: 'SaaS Overview', mock: true },
      { to: '/app/reports/revenue', label: 'Revenue & MRR', mock: true },
      { to: '/app/reports/subscriptions', label: 'Subscriptions', mock: true },
      { to: '/app/reports/usage', label: 'Usage Analytics', mock: true },
      { to: '/app/reports/churn', label: 'Churn & Retention', mock: true },
      { to: '/app/reports/plan-performance', label: 'Plan Performance', mock: true },
      { to: '/app/reports/trial-conversion', label: 'Trial Conversion', mock: true },
    ],
  },
  {
    label: 'Support',
    icon: icons.support,
    mock: true,
    children: [
      { to: '/app/support', label: 'Support Tickets', badge: '8', mock: true },
      { to: '/app/broadcast', label: 'Broadcast', mock: true },
      { to: '/app/templates', label: 'Templates', mock: true },
    ],
  },
  { to: '/app/todo', label: 'SaaS Todo', icon: icons.todo, mock: true },

  { type: 'group', label: 'System' },
  { to: '/app/audit', label: 'Audit Log', icon: icons.audit },
  { to: '/app/settings', label: 'Settings', icon: icons.settings },
  { to: '/app/security', label: 'Security', icon: icons.security, mock: true },
  { to: '/app/health', label: 'System Health', icon: icons.health, badge: 'OK', mock: true },
]

onMounted(() => {
  ui.syncDesktopSidebar()
  window.addEventListener('resize', ui.syncDesktopSidebar)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', ui.syncDesktopSidebar)
})
</script>
