<template>
  <div>
    <button
      v-if="ui.sidebarMobileOpen"
      type="button"
      class="sidebar-backdrop"
      aria-label="Close sidebar"
      @click="ui.closeMobileSidebar"
    />

    <aside class="sidebar" :class="{ 'sidebar--collapsed': ui.sidebarCollapsed, 'sidebar--mobile-open': ui.sidebarMobileOpen }">
      <div class="sidebar-header">
        <RouterLink v-slot="{ href, navigate }" to="/app" custom>
          <a :href="href" class="sidebar-brand" @click="handleNavigate($event, navigate)">
            <span class="brand-dot" />
            <span class="brand-text">
              <span class="brand-name">PortalOnline</span>
              <span class="brand-subtitle">Owner Console</span>
            </span>
          </a>
        </RouterLink>

        <button type="button" class="sidebar-toggle" aria-label="Toggle sidebar" @click="ui.toggleSidebar">
          {{ ui.sidebarCollapsed ? '›' : '‹' }}
        </button>
      </div>

      <nav class="sidebar-nav" aria-label="Owner navigation">
        <RouterLink v-for="item in items" :key="item.to" v-slot="{ href, navigate, isActive, isExactActive }" :to="item.to" custom>
          <a
            :href="href"
            class="nav-item nav-item--root"
            :class="{ 'nav-item--active': item.children ? isActive : isExactActive }"
            @click="handleNavigate($event, navigate)"
          >
            <span class="nav-icon">{{ item.icon }}</span>
            <span class="nav-label">{{ item.label }}</span>
            <span v-if="item.children" class="nav-chevron">⌄</span>
          </a>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <p class="sidebar-footer-label">Workspace</p>
        <p class="sidebar-footer-title">Multi-tenant SaaS</p>
        <p class="sidebar-footer-copy">Monitor tenants, audit, dan platform health.</p>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { useUiStore } from '@/stores/ui'

type SidebarItem = {
  to: string
  label: string
  icon?: string
  children?: boolean
}

defineProps<{ items: SidebarItem[] }>()

const ui = useUiStore()

function handleNavigate(event: MouseEvent, navigate: (event?: MouseEvent) => void) {
  event.preventDefault()
  navigate(event)
  ui.closeMobileSidebar()
}
</script>
