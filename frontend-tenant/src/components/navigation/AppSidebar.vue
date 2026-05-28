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
        <RouterLink to="/tenant" class="sidebar-brand" @click="ui.closeMobileSidebar">
          <span class="brand-dot" />
          <span class="brand-text">
            <span class="brand-name">PortalOnline</span>
            <span class="brand-subtitle">Tenant Console</span>
          </span>
        </RouterLink>
        <button type="button" class="sidebar-toggle" aria-label="Toggle sidebar" @click="ui.toggleSidebar">
          {{ ui.sidebarCollapsed ? '›' : '‹' }}
        </button>
      </div>

      <nav class="sidebar-nav">
        <RouterLink
          v-for="item in items"
          :key="item.to"
          :to="item.to"
          class="nav-item nav-item--root"
          @click="ui.closeMobileSidebar"
        >
          <span class="nav-icon">{{ item.icon }}</span>
          <span class="nav-label">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <p class="sidebar-footer-label">Workspace</p>
        <p class="sidebar-footer-title">Tenant Operations</p>
        <p class="sidebar-footer-copy">Kelola user, role, settings, dan audit aktivitas tenant.</p>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { useUiStore } from '@/stores/ui'

defineProps<{ items: Array<{ to: string; label: string; icon?: string }> }>()

const ui = useUiStore()
</script>
