<template>
  <select
    :value="tenant.selectedTenantId || ''"
    class="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm"
    @change="onChange"
  >
    <option value="">No tenant</option>
    <option v-for="m in tenant.memberships" :key="m.tenant_id" :value="m.tenant_id">
      {{ m.tenant_name || m.tenant_id }} ({{ m.role }})
    </option>
  </select>
</template>

<script setup lang="ts">
import { useTenantStore } from '@/stores/tenant'

const tenant = useTenantStore()

function onChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  tenant.selectTenant(value || null)
}
</script>
