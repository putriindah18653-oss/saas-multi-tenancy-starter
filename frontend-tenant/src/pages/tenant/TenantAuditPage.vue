<template>
  <div class="space-y-4">
    <div>
      <h2 class="text-xl font-semibold text-slate-900">Audit Log</h2>
      <p class="text-sm text-slate-500">Jejak aksi penting pada tenant aktif.</p>
    </div>
    <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>
    <div class="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <table class="min-w-full divide-y divide-slate-200 text-sm">
        <thead class="bg-slate-50 text-left text-slate-600">
          <tr><th class="px-4 py-3">Waktu</th><th class="px-4 py-3">Action</th><th class="px-4 py-3">Resource</th><th class="px-4 py-3">Actor</th></tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-for="entry in entries" :key="entry.id">
            <td class="px-4 py-3 text-slate-500">{{ new Date(entry.created_at).toLocaleString() }}</td>
            <td class="px-4 py-3 font-medium text-slate-900">{{ entry.action }}</td>
            <td class="px-4 py-3 text-slate-600">{{ entry.resource_type }} {{ entry.resource_id || '' }}</td>
            <td class="px-4 py-3 text-slate-500">{{ entry.actor_user_id || '-' }}</td>
          </tr>
          <tr v-if="!loading && entries.length === 0"><td colspan="4" class="px-4 py-6 text-center text-slate-500">Belum ada audit log.</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { auditService, type AuditEntry } from '@/services/audit'
const entries = ref<AuditEntry[]>([])
const loading = ref(false)
const error = ref('')
onMounted(async () => {
  loading.value = true
  try { entries.value = (await auditService.tenant()).data.data } catch { error.value = 'Gagal memuat audit log.' } finally { loading.value = false }
})
</script>
