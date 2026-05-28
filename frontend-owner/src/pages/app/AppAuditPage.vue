<template>
  <div class="space-y-6">
    <div>
      <h2 class="owner-page-title">App Audit Log</h2>
      <p class="owner-page-subtitle">Jejak aksi platform dan tenant untuk investigasi owner.</p>
    </div>

    <UiAlert v-if="error" title="Gagal memuat audit log" tone="danger">{{ error }}</UiAlert>

    <AppCard>
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 6" :key="i" class="h-12 animate-pulse rounded-[var(--radius-button)] bg-white/[0.05]" />
      </div>

      <UiEmptyState v-else-if="entries.length === 0" title="Belum ada audit log" description="Aktivitas platform akan tampil di sini setelah event terekam." />

      <div v-else class="overflow-x-auto">
        <table class="owner-table">
          <thead>
            <tr>
              <th>Waktu</th>
              <th>Action</th>
              <th>Tenant</th>
              <th>Resource</th>
              <th>Actor</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in entries" :key="entry.id">
              <td class="whitespace-nowrap text-[var(--text-muted)]">{{ formatDateTime(entry.created_at) }}</td>
              <td>
                <span class="rounded-full border border-[var(--border)] bg-white/[0.03] px-2 py-1 font-mono text-xs text-[var(--text-primary)]">{{ entry.action }}</span>
              </td>
              <td class="font-mono text-xs">{{ entry.tenant_id || '-' }}</td>
              <td>
                <span class="text-[var(--text-primary)]">{{ entry.resource_type }}</span>
                <span class="ml-1 font-mono text-xs text-[var(--text-muted)]">{{ entry.resource_id || '' }}</span>
              </td>
              <td class="font-mono text-xs">{{ entry.actor_user_id || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </AppCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
import { auditService, type AuditEntry } from '@/services/audit'
import { formatDateTime } from '@/utils/format'

const entries = ref<AuditEntry[]>([])
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  loading.value = true
  error.value = ''
  try {
    entries.value = (await auditService.app()).data.data
  } catch (e: any) {
    error.value = e?.response?.data?.error?.message || 'Gagal memuat audit log.'
  } finally {
    loading.value = false
  }
})
</script>
