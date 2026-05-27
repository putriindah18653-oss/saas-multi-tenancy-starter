<template>
  <div class="space-y-4">
    <h2 class="text-xl font-semibold text-slate-900">Tenant Dashboard</h2>
    <div class="rounded-xl border border-slate-200 bg-white p-4">
      <p class="text-sm text-slate-600">Tenant aktif: {{ tenant.selectedMembership?.tenant_name || tenant.selectedTenantId || 'belum dipilih' }}</p>
      <p class="mt-2 text-sm text-slate-500">Halaman manajemen user tenant masuk Task 11.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useTenantStore } from '@/stores/tenant'
const tenant = useTenantStore()
</script>
