import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

const themeKey = 'portalonline-owner-theme'
const sidebarKey = 'portalonline-owner-sidebar-collapsed'

export const useUiStore = defineStore('ui', () => {
  const theme = ref<'light' | 'dark'>((localStorage.getItem(themeKey) as 'light' | 'dark' | null) ?? 'dark')
  const sidebarCollapsed = ref(localStorage.getItem(sidebarKey) === 'true')
  const sidebarMobileOpen = ref(false)

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function syncDesktopSidebar() {
    if (window.matchMedia('(min-width: 761px)').matches) {
      sidebarMobileOpen.value = false
    }
  }

  function openMobileSidebar() {
    sidebarMobileOpen.value = true
  }

  function closeMobileSidebar() {
    sidebarMobileOpen.value = false
  }

  watch(
    theme,
    (value) => {
      localStorage.setItem(themeKey, value)
      document.documentElement.dataset.theme = value
    },
    { immediate: true },
  )

  watch(sidebarCollapsed, (value) => {
    localStorage.setItem(sidebarKey, String(value))
  })

  return {
    theme,
    sidebarCollapsed,
    sidebarMobileOpen,
    toggleTheme,
    toggleSidebar,
    syncDesktopSidebar,
    openMobileSidebar,
    closeMobileSidebar,
  }
})
