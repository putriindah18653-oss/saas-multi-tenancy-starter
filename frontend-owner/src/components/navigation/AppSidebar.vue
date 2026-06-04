<template>
  <div>
    <button
      v-if="ui.sidebarMobileOpen"
      type="button"
      class="sidebar-backdrop"
      aria-label="Close sidebar"
      @click="ui.closeMobileSidebar"
    />

    <aside
      id="owner-sidebar"
      class="sidebar"
      :class="{ 'sidebar--collapsed': ui.sidebarCollapsed, 'sidebar--mobile-open': ui.sidebarMobileOpen }"
    >
      <div class="sidebar-header">
        <RouterLink v-slot="{ href, navigate }" to="/app" custom>
          <a :href="href" class="sidebar-brand" @click="handleNavigate($event, navigate)">
            <span class="brand-dot" />
            <span class="brand-text">
              <span class="brand-name">PortalOnline</span>
            </span>
          </a>
        </RouterLink>

        <button
          type="button"
          class="sidebar-toggle"
          :aria-label="ui.sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
          aria-controls="owner-sidebar"
          :aria-expanded="!ui.sidebarCollapsed"
          @click="ui.toggleSidebar"
        >
          {{ ui.sidebarCollapsed ? '›' : '‹' }}
        </button>
      </div>

      <nav class="sidebar-nav" aria-label="Main navigation">
        <template v-for="item in items" :key="item.label">
          <p v-if="item.type === 'group'" class="nav-group-label">{{ item.label }}</p>

          <div v-else-if="item.children?.length" class="nav-dropdown">
            <button
              type="button"
              class="nav-item nav-item--root"
              :class="{ 'nav-item--active': isDropdownActive(item), 'nav-item--muted': item.mock }"
              :title="ui.sidebarCollapsed ? item.label : undefined"
              :aria-expanded="isDropdownOpen(item.label)"
              @click="toggleDropdown(item.label)"
            >
              <span class="nav-icon" v-if="item.icon" v-html="item.icon"></span>
              <span class="nav-icon" v-else aria-hidden="true">●</span>
              <span class="nav-label">{{ item.label }}</span>
              <span v-if="item.badge" class="nav-badge">{{ item.badge }}</span>
              <span class="nav-chevron" :class="{ 'nav-chevron--open': isDropdownOpen(item.label) }">⌄</span>
            </button>

            <div v-if="isDropdownOpen(item.label) && !ui.sidebarCollapsed" class="nav-submenu">
              <RouterLink
                v-for="child in item.children"
                :key="child.to"
                v-slot="{ href, navigate, isActive, isExactActive }"
                :to="child.to"
                custom
              >
                <a
                  :href="href"
                  class="nav-subitem"
                  :class="{ 'nav-subitem--active': isActive || isExactActive, 'nav-item--muted': child.mock }"
                  @click="handleNavigate($event, navigate)"
                >
                  <span class="nav-subdot" />
                  <span class="nav-label">{{ child.label }}</span>
                  <span v-if="child.badge" class="nav-badge">{{ child.badge }}</span>
                </a>
              </RouterLink>
            </div>
          </div>

          <RouterLink v-else v-slot="{ href, navigate, isActive, isExactActive }" :to="item.to" custom>
            <a
              :href="href"
              class="nav-item nav-item--root"
              :class="{ 'nav-item--active': isActive || isExactActive, 'nav-item--muted': item.mock }"
              :title="ui.sidebarCollapsed ? item.label : undefined"
              @click="handleNavigate($event, navigate)"
            >
              <span class="nav-icon" v-if="item.icon" v-html="item.icon"></span>
              <span class="nav-icon" v-else aria-hidden="true">●</span>
              <span class="nav-label">{{ item.label }}</span>
              <span v-if="item.badge" class="nav-badge">{{ item.badge }}</span>
            </a>
          </RouterLink>
        </template>
      </nav>

    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useUiStore } from '@/stores/ui'

type SidebarChildItem = {
  to: string
  label: string
  badge?: string
  mock?: boolean
  permission?: string
}

type SidebarLinkItem = {
  type?: 'link'
  to?: string
  label: string
  icon?: string
  children?: SidebarChildItem[]
  badge?: string
  mock?: boolean
  permission?: string
}

type SidebarGroupItem = {
  type: 'group'
  label: string
}

type SidebarItem = SidebarLinkItem | SidebarGroupItem

defineProps<{ items: SidebarItem[] }>()

const ui = useUiStore()
const route = useRoute()
const openDropdowns = ref<Record<string, boolean>>({
  Reports: true,
})

function isDropdownOpen(label: string) {
  return openDropdowns.value[label] ?? false
}

function toggleDropdown(label: string) {
  openDropdowns.value[label] = !isDropdownOpen(label)
}

function isDropdownActive(item: SidebarLinkItem) {
  return item.children?.some((child) => route.path === child.to || route.fullPath === child.to) ?? false
}

function handleNavigate(event: MouseEvent, navigate: (event?: MouseEvent) => void) {
  navigate(event)
  ui.closeMobileSidebar()
}
</script>
