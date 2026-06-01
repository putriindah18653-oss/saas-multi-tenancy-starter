<template>
  <div class="space-y-6">
    <div>
      <h2 class="tenant-page-title">Audit Log</h2>
      <p class="tenant-page-subtitle">Jejak aksi penting pada tenant aktif.</p>
    </div>

    <UiAlert v-if="error" title="Gagal memuat audit log" tone="danger">{{ error }}</UiAlert>

    <AppCard>
      <div class="overflow-x-auto">
        <table class="tenant-table">
          <thead>
            <tr>
              <th>Waktu</th>
              <th>Action</th>
              <th>Resource</th>
              <th>Actor</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading" v-for="i in 4" :key="`audit-skeleton-${i}`">
              <td colspan="4"><div class="h-10 animate-pulse rounded-[var(--tenant-radius-button)] bg-white/[0.05]" /></td>
            </tr>
            <tr v-for="entry in entries" v-else :key="entry.id">
              <td class="text-[var(--tenant-text-muted)]">{{ new Date(entry.created_at).toLocaleString() }}</td>
              <td class="font-medium text-[var(--tenant-text-primary)]">{{ entry.action }}</td>
              <td>{{ entry.resource_type }} {{ entry.resource_id || '' }}</td>
              <td class="text-[var(--tenant-text-muted)]">{{ entry.actor_user_id || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <UiEmptyState
        v-if="!loading && entries.length === 0"
        class="mt-4"
        title="Belum ada audit log"
        description="Aktivitas penting tenant akan muncul di sini."
      />
    </AppCard>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
import { auditService, type AuditEntry } from '@/services/audit'

const entries = ref<AuditEntry[]>([])
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  loading.value = true
  error.value = ''
  try {
    entries.value = (await auditService.tenant()).data.data || []
  } catch (e: unknown) {
    error.value = 'Gagal memuat audit log.'
    console.error('[tenant-audit] load failed', e)
  } finally {
    loading.value = false
  }
})
</script>
