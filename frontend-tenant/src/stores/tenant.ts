import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { TenantMembership } from '@/contracts/api'

export type { TenantMembership } from '@/contracts/api'

const SELECTED_TENANT_KEY = 'selected_tenant_id'

export const useTenantStore = defineStore('tenant', () => {
  const memberships = ref<TenantMembership[]>([])
  const selectedTenantId = ref<string | null>(localStorage.getItem(SELECTED_TENANT_KEY))

  const hasTenant = computed(() => !!selectedTenantId.value)
  const selectedMembership = computed(() => memberships.value.find((m) => m.tenant_id === selectedTenantId.value) ?? null)
  const validTenantIds = computed(() => memberships.value.map((m) => m.tenant_id))

  function persistSelectedTenant(tenantId: string | null) {
    if (tenantId) localStorage.setItem(SELECTED_TENANT_KEY, tenantId)
    else localStorage.removeItem(SELECTED_TENANT_KEY)
  }

  function setMemberships(items: TenantMembership[]) {
    memberships.value = items

    const selectedIsValid = !!selectedTenantId.value && items.some((m) => m.tenant_id === selectedTenantId.value)
    const nextSelectedTenantId = selectedIsValid ? selectedTenantId.value : items[0]?.tenant_id ?? null

    selectedTenantId.value = nextSelectedTenantId
    persistSelectedTenant(nextSelectedTenantId)
  }

  function selectTenant(tenantId: string | null) {
    const nextSelectedTenantId = tenantId && memberships.value.some((m) => m.tenant_id === tenantId) ? tenantId : null
    selectedTenantId.value = nextSelectedTenantId
    persistSelectedTenant(nextSelectedTenantId)
    return nextSelectedTenantId === tenantId
  }

  function clearTenant() {
    memberships.value = []
    selectedTenantId.value = null
    persistSelectedTenant(null)
  }

  return { memberships, selectedTenantId, hasTenant, selectedMembership, validTenantIds, setMemberships, selectTenant, clearTenant }
})
