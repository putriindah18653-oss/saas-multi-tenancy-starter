import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export type TenantMembership = {
  tenant_id: string
  tenant_name?: string
  role: string
}

export const useTenantStore = defineStore('tenant', () => {
  const memberships = ref<TenantMembership[]>([])
  const selectedTenantId = ref<string | null>(null)

  const hasTenant = computed(() => !!selectedTenantId.value)
  const selectedMembership = computed(() => memberships.value.find((m) => m.tenant_id === selectedTenantId.value) ?? null)

  function setMemberships(items: TenantMembership[]) {
    memberships.value = items
    if (!selectedTenantId.value && items.length > 0) selectedTenantId.value = items[0].tenant_id
  }

  function selectTenant(tenantId: string | null) {
    selectedTenantId.value = tenantId
  }

  function clearTenant() {
    memberships.value = []
    selectedTenantId.value = null
  }

  return { memberships, selectedTenantId, hasTenant, selectedMembership, setMemberships, selectTenant, clearTenant }
})
