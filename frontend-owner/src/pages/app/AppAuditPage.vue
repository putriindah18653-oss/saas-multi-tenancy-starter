<template>
  <div class="space-y-4">
    <UiAlert v-if="error" title="Gagal memuat audit log" tone="danger">{{ error }}</UiAlert>

    <AppCard>
      <div class="mb-4 flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
        <div class="grid gap-2 md:grid-cols-[minmax(220px,1fr)_180px_180px_260px] xl:max-w-5xl">
          <label class="relative block min-w-0">
            <span class="sr-only">Cari audit log</span>
            <input
              v-model.trim="search"
              class="owner-input pr-9"
              type="search"
              placeholder="Cari audit log..."
            />
            <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]">⌕</span>
          </label>

          <label class="block">
            <span class="sr-only">Filter action</span>
            <select v-model="actionFilter" class="owner-input">
              <option value="">Semua action</option>
              <option v-for="action in actionOptions" :key="action" :value="action">{{ action }}</option>
            </select>
          </label>

          <label class="block">
            <span class="sr-only">Filter resource</span>
            <select v-model="resourceFilter" class="owner-input">
              <option value="">Semua resource</option>
              <option v-for="resource in resourceOptions" :key="resource" :value="resource">{{ resource }}</option>
            </select>
          </label>

          <div class="relative">
            <button
              type="button"
              class="owner-input flex h-10 items-center justify-between text-left"
              :aria-expanded="datePickerOpen"
              aria-controls="audit-date-range-popover"
              aria-haspopup="dialog"
              aria-label="Filter tanggal audit log"
              @click="datePickerOpen = !datePickerOpen"
            >
              <span :class="dateFrom || dateTo ? 'text-[var(--text-primary)]' : 'text-[var(--text-muted)]'">{{ dateRangeLabel }}</span>
              <span class="text-[var(--text-muted)]">⌄</span>
            </button>

            <div v-if="datePickerOpen" id="audit-date-range-popover" class="date-range-popover" role="dialog" aria-label="Filter tanggal audit log">
              <div class="date-range-header">
                <button type="button" class="date-range-nav" aria-label="Previous month" @click="shiftMonth(-1)">‹</button>
                <p>{{ calendarTitle }}</p>
                <button type="button" class="date-range-nav" aria-label="Next month" @click="shiftMonth(1)">›</button>
              </div>

              <div class="date-range-weekdays">
                <span v-for="day in weekdays" :key="day">{{ day }}</span>
              </div>

              <div class="date-range-grid">
                <button
                  v-for="day in calendarDays"
                  :key="day.key"
                  type="button"
                  class="date-range-day"
                  :class="{
                    'date-range-day--muted': !day.currentMonth,
                    'date-range-day--selected': isSameDay(day.date, dateFrom) || isSameDay(day.date, dateTo),
                    'date-range-day--in-range': isInSelectedRange(day.date),
                  }"
                  :aria-label="day.date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })"
                  @click="selectCalendarDate(day.date)"
                  @keydown.enter.prevent="selectCalendarDate(day.date)"
                  @keydown.space.prevent="selectCalendarDate(day.date)"
                >
                  {{ day.date.getDate() }}
                </button>
              </div>

              <div class="date-range-actions">
                <button type="button" @click="clearDateRange">Clear</button>
                <button type="button" @click="datePickerOpen = false">Done</button>
              </div>
            </div>
          </div>
        </div>

        <div class="relative flex justify-end">
          <button
            type="button"
            class="inline-flex h-10 w-10 items-center justify-center rounded-[var(--radius-control)] border border-[var(--border-strong)] bg-[var(--button-secondary-bg)] text-[var(--text-primary)] transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-55"
            aria-label="Export audit log"
            :aria-expanded="exportMenuOpen"
            aria-controls="audit-export-menu"
            aria-haspopup="menu"
            :disabled="filteredEntries.length === 0"
            @click="exportMenuOpen = !exportMenuOpen"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M12 3v12m0 0 4-4m-4 4-4-4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
              <path d="M5 14v5a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </button>

          <div
            v-if="exportMenuOpen"
            id="audit-export-menu"
            class="absolute right-0 top-12 z-20 w-40 overflow-hidden rounded-[var(--radius-card)] border border-[var(--border)] bg-[var(--surface-elevated)] py-1 shadow-2xl"
            role="menu"
          >
            <button type="button" class="export-menu-item" role="menuitem" @click="exportData('csv')">Export current filtered CSV</button>
            <button type="button" class="export-menu-item" role="menuitem" @click="exportData('xlsx')">Export current filtered XLSX</button>
            <button type="button" class="export-menu-item" role="menuitem" @click="exportData('pdf')">Export current filtered PDF</button>
          </div>
        </div>
      </div>

      <UiAlert title="Showing latest loaded window only" tone="warning">
        Only latest 200 audit entries are loaded. Filters, pagination, and export apply only to this current loaded window.
      </UiAlert>

      <div v-if="loading" class="space-y-3">
        <div v-for="i in 7" :key="i" class="h-14 animate-pulse rounded-[var(--radius-button)] bg-white/[0.05]" />
      </div>

      <UiEmptyState
        v-else-if="filteredEntries.length === 0"
        :title="entries.length ? 'Audit log tidak ditemukan' : 'Belum ada audit log'"
        :description="entries.length ? 'Ubah kata kunci atau filter.' : 'Aktivitas platform akan tampil di sini setelah event terekam.'"
      />

      <template v-else>
        <div class="overflow-x-auto rounded-[var(--radius-card)] border border-[var(--border)]">
          <table class="owner-table">
            <thead>
              <tr>
                <th>Waktu</th>
                <th>Action</th>
                <th>Tenant</th>
                <th>Resource</th>
                <th>Actor</th>
                <th>Client</th>
                <th class="w-10"></th>
              </tr>
            </thead>
            <tbody>
              <template v-for="entry in paginatedEntries" :key="entry.id">
                <tr>
                  <td class="whitespace-nowrap text-[var(--text-muted)]">{{ formatDateTime(entry.created_at) }}</td>
                  <td>
                    <span class="rounded-full border border-[var(--border)] bg-white/[0.03] px-2 py-1 font-mono text-xs text-[var(--text-primary)]">{{ entry.action }}</span>
                  </td>
                  <td class="font-mono text-xs">{{ entry.tenant_id || '-' }}</td>
                  <td>
                    <span class="text-[var(--text-primary)]">{{ entry.resource_type }}</span>
                    <span v-if="entry.resource_id" class="ml-1 font-mono text-xs text-[var(--text-muted)]">{{ entry.resource_id }}</span>
                  </td>
                  <td class="font-mono text-xs">{{ entry.actor_user_id || '-' }}</td>
                  <td class="max-w-[220px] truncate text-xs text-[var(--text-muted)]" :title="clientTitle(entry)">
                    {{ entry.ip_address || '-' }}
                  </td>
                  <td class="text-right">
                    <button
                      type="button"
                      class="rounded-[var(--radius-button)] px-2 py-1 text-sm text-[var(--text-muted)] hover:bg-white/[0.06] hover:text-[var(--text-primary)]"
                      :aria-expanded="expandedId === entry.id"
                      tabindex="0"
                      :aria-controls="`audit-entry-${entry.id}`"
                      :aria-label="`${expandedId === entry.id ? 'Collapse' : 'Expand'} audit entry ${entry.id}`"
                      @click="toggleEntry(entry.id)"
                      @keydown.enter.prevent="toggleEntry(entry.id)"
                      @keydown.space.prevent="toggleEntry(entry.id)"
                    >
                      {{ expandedId === entry.id ? '−' : '+' }}
                    </button>
                  </td>
                </tr>
                <tr v-if="expandedId === entry.id" :id="`audit-entry-${entry.id}`" class="bg-white/[0.02]">
                  <td colspan="7" class="space-y-3">
                    <div class="grid gap-3 text-xs md:grid-cols-2">
                      <div>
                        <p class="font-semibold text-[var(--text-primary)]">Audit ID</p>
                        <p class="mt-1 break-all font-mono text-[var(--text-muted)]">{{ entry.id }}</p>
                      </div>
                      <div>
                        <p class="font-semibold text-[var(--text-primary)]">User Agent</p>
                        <p class="mt-1 break-all text-[var(--text-muted)]">{{ entry.user_agent || '-' }}</p>
                      </div>
                    </div>
                    <div>
                      <p class="mb-2 text-xs font-semibold text-[var(--text-primary)]">Metadata</p>
                      <pre class="max-h-72 overflow-auto rounded-[var(--radius-button)] border border-[var(--border)] bg-black/20 p-3 text-xs text-[var(--text-secondary)]">{{ formatMetadata(entry.metadata) }}</pre>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>

        <div class="mt-4 flex flex-col gap-3 border-t border-[var(--border)] pt-4 text-sm text-[var(--text-muted)] md:flex-row md:items-center md:justify-between">
          <p>Showing {{ startRow }}–{{ endRow }} of {{ filteredEntries.length }} current loaded rows (latest 200 max)</p>
          <div class="flex flex-wrap items-center gap-2">
            <AppButton variant="ghost" :disabled="page === 1" @click="goToPage(1)">First</AppButton>
            <AppButton variant="ghost" :disabled="page === 1" @click="goToPage(page - 1)">Prev</AppButton>
            <span class="px-2 font-medium text-[var(--text-secondary)]">Page {{ page }} of {{ totalPages }}</span>
            <AppButton variant="ghost" :disabled="page === totalPages" @click="goToPage(page + 1)">Next</AppButton>
            <AppButton variant="ghost" :disabled="page === totalPages" @click="goToPage(totalPages)">Last</AppButton>
          </div>
        </div>
      </template>
    </AppCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppButton from '@/components/common/AppButton.vue'
import AppCard from '@/components/common/AppCard.vue'
import UiAlert from '@/components/common/UiAlert.vue'
import UiEmptyState from '@/components/common/UiEmptyState.vue'
import { auditService, type AuditEntry } from '@/services/audit'
import { formatDateTime } from '@/utils/format'

const entries = ref<AuditEntry[]>([])
const loading = ref(false)
const error = ref('')
const search = ref('')
const actionFilter = ref('')
const resourceFilter = ref('')
const dateFrom = ref('')
const dateTo = ref('')
const datePickerOpen = ref(false)
const calendarCursor = ref(startOfMonth(new Date()))
const exportMenuOpen = ref(false)
const page = ref(1)
const pageSize = 10
const expandedId = ref('')

const actionOptions = computed(() => uniqueSorted(entries.value.map((entry) => entry.action)))
const resourceOptions = computed(() => uniqueSorted(entries.value.map((entry) => entry.resource_type)))
const weekdays = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const calendarTitle = computed(() => new Intl.DateTimeFormat('en-US', { month: 'long', year: 'numeric' }).format(calendarCursor.value))
const dateRangeLabel = computed(() => {
  if (dateFrom.value && dateTo.value) return `${formatDateInput(dateFrom.value)} - ${formatDateInput(dateTo.value)}`
  if (dateFrom.value) return `${formatDateInput(dateFrom.value)} - ...`
  return 'Pilih tanggal'
})
const calendarDays = computed(() => {
  const firstDay = startOfMonth(calendarCursor.value)
  const start = new Date(firstDay)
  start.setDate(firstDay.getDate() - firstDay.getDay())

  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(start)
    date.setDate(start.getDate() + index)
    return {
      date,
      key: toDateKey(date),
      currentMonth: date.getMonth() === calendarCursor.value.getMonth(),
    }
  })
})

// Precompute search index with compact JSON (avoids per-keystroke pretty-print)
const searchIndex = computed(() => {
  return entries.value.map((entry) => [
    entry.id, entry.action, entry.resource_type, entry.resource_id,
    entry.tenant_id, entry.actor_user_id, entry.ip_address, entry.user_agent,
    JSON.stringify(entry.metadata),
  ].map((v) => String(v ?? '').toLowerCase()))
})

const filteredEntries = computed(() => {
  const keyword = search.value.toLowerCase()

  return entries.value.filter((entry, index) => {
    if (actionFilter.value && entry.action !== actionFilter.value) return false
    if (resourceFilter.value && entry.resource_type !== resourceFilter.value) return false
    if (!isWithinDateRange(entry.created_at)) return false
    if (!keyword) return true
    return searchIndex.value[index]?.some((value) => value.includes(keyword)) ?? false
  })
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredEntries.value.length / pageSize)))
const paginatedEntries = computed(() => {
  const start = (page.value - 1) * pageSize
  return filteredEntries.value.slice(start, start + pageSize)
})
const startRow = computed(() => filteredEntries.value.length === 0 ? 0 : (page.value - 1) * pageSize + 1)
const endRow = computed(() => Math.min(page.value * pageSize, filteredEntries.value.length))

watch([search, actionFilter, resourceFilter, dateFrom, dateTo], () => {
  page.value = 1
  expandedId.value = ''
  exportMenuOpen.value = false
})

watch(totalPages, (value) => {
  if (page.value > value) page.value = value
})

async function loadEntries() {
  loading.value = true
  error.value = ''
  try {
    entries.value = (await auditService.app(200)).data.data
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    error.value = err?.response?.data?.error?.message || 'Gagal memuat audit log.'
    console.error('[audit] load failed', e)
  } finally {
    loading.value = false
  }
}

function toggleEntry(id: string) {
  expandedId.value = expandedId.value === id ? '' : id
}

function goToPage(target: number) {
  page.value = Math.min(Math.max(target, 1), totalPages.value)
  expandedId.value = ''
}

function isWithinDateRange(value: string) {
  const current = new Date(value)
  if (Number.isNaN(current.getTime())) return true

  if (dateFrom.value) {
    const from = new Date(`${dateFrom.value}T00:00:00`)
    if (current < from) return false
  }

  if (dateTo.value) {
    const to = new Date(`${dateTo.value}T23:59:59.999`)
    if (current > to) return false
  }

  return true
}

function selectCalendarDate(date: Date) {
  const selected = toDateKey(date)

  if (!dateFrom.value || (dateFrom.value && dateTo.value)) {
    dateFrom.value = selected
    dateTo.value = ''
    return
  }

  if (selected < dateFrom.value) {
    dateTo.value = dateFrom.value
    dateFrom.value = selected
    datePickerOpen.value = false
    return
  }

  dateTo.value = selected
  datePickerOpen.value = false
}

function clearDateRange() {
  dateFrom.value = ''
  dateTo.value = ''
  datePickerOpen.value = false
}

function shiftMonth(step: number) {
  calendarCursor.value = new Date(calendarCursor.value.getFullYear(), calendarCursor.value.getMonth() + step, 1)
}

function isSameDay(date: Date, value: string) {
  return !!value && toDateKey(date) === value
}

function isInSelectedRange(date: Date) {
  if (!dateFrom.value || !dateTo.value) return false
  const key = toDateKey(date)
  return key > dateFrom.value && key < dateTo.value
}

function startOfMonth(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), 1)
}

function toDateKey(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function formatDateInput(value: string) {
  const [year, month, day] = value.split('-')
  if (!year || !month || !day) return value
  return `${month}/${day}/${year}`
}

function exportData(format: 'csv' | 'xlsx' | 'pdf') {
  exportMenuOpen.value = false

  if (format === 'csv') {
    exportDelimited('csv')
    return
  }

  if (format === 'xlsx') {
    exportDelimited('xlsx')
    return
  }

  exportPdf()
}

function exportDelimited(format: 'csv' | 'xlsx') {
  const headers = ['created_at', 'action', 'tenant_id', 'resource_type', 'resource_id', 'actor_user_id', 'ip_address', 'user_agent', 'metadata', 'id']
  const rows = filteredEntries.value.map((entry) => [
    entry.created_at,
    entry.action,
    entry.tenant_id || '',
    entry.resource_type,
    entry.resource_id || '',
    entry.actor_user_id || '',
    entry.ip_address || '',
    entry.user_agent || '',
    formatMetadata(entry.metadata),
    entry.id,
  ])

  const delimiter = format === 'csv' ? ',' : '\t'
  const content = [headers, ...rows].map((row) => row.map(format === 'csv' ? csvEscape : tsvEscape).join(delimiter)).join('\n')
  downloadTextFile(
    content,
    `audit-log-${new Date().toISOString().slice(0, 10)}.${format}`,
    format === 'csv' ? 'text/csv;charset=utf-8;' : 'application/vnd.ms-excel;charset=utf-8;',
  )
}

function exportPdf() {
  const title = `Audit Log - ${new Date().toLocaleString()}`
  const rows = filteredEntries.value.map((entry) => `
    <tr>
      <td>${htmlEscape(formatDateTime(entry.created_at))}</td>
      <td>${htmlEscape(entry.action)}</td>
      <td>${htmlEscape(entry.tenant_id || '-')}</td>
      <td>${htmlEscape(entry.resource_type)} ${htmlEscape(entry.resource_id || '')}</td>
      <td>${htmlEscape(entry.actor_user_id || '-')}</td>
      <td>${htmlEscape(entry.ip_address || '-')}</td>
    </tr>
  `).join('')

  const popup = window.open('', '_blank')
  if (!popup) return

  popup.document.write(`
    <!doctype html>
    <html>
      <head>
        <title>${htmlEscape(title)}</title>
        <style>
          body { font-family: Arial, sans-serif; color: #111827; padding: 24px; }
          h1 { font-size: 18px; margin: 0 0 16px; }
          table { width: 100%; border-collapse: collapse; font-size: 11px; }
          th, td { border: 1px solid #d1d5db; padding: 7px; text-align: left; vertical-align: top; }
          th { background: #f3f4f6; font-size: 10px; text-transform: uppercase; }
        </style>
      </head>
      <body>
        <h1>${htmlEscape(title)}</h1>
        <table>
          <thead>
            <tr><th>Waktu</th><th>Action</th><th>Tenant</th><th>Resource</th><th>Actor</th><th>Client</th></tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </body>
    </html>
  `)
  popup.document.close()
  popup.focus()
  popup.print()
}

function csvEscape(value: unknown) {
  return `"${String(value ?? '').replace(/"/g, '""')}"`
}

function tsvEscape(value: unknown) {
  return String(value ?? '').replace(/[\t\r\n]+/g, ' ')
}

function htmlEscape(value: unknown) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

function downloadTextFile(content: string, filename: string, type: string) {
  const blob = new Blob([content], { type })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

function uniqueSorted(values: Array<string | undefined>) {
  return [...new Set(values.filter(Boolean) as string[])].sort((a, b) => a.localeCompare(b))
}

function formatMetadata(metadata: Record<string, unknown>) {
  if (!metadata || Object.keys(metadata).length === 0) return '{}'
  return JSON.stringify(metadata, null, 2)
}

function clientTitle(entry: AuditEntry) {
  return [entry.ip_address, entry.user_agent].filter(Boolean).join(' • ')
}

// Close popovers on Escape and click-outside
function closeAllPopovers() {
  datePickerOpen.value = false
  exportMenuOpen.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') closeAllPopovers()
}

function onClickOutside(e: MouseEvent) {
  const target = e.target as HTMLElement | null
  if (!target) return
  if (datePickerOpen.value && !target.closest('.date-range-popover') && !target.closest('[aria-label="Filter tanggal audit log"]')) {
    datePickerOpen.value = false
  }
  if (exportMenuOpen.value && !target.closest('[aria-label="Export audit log"]') && !target.closest('.export-menu-item')) {
    exportMenuOpen.value = false
  }
}

onMounted(() => {
  loadEntries()
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('click', onClickOutside)
  window.addEventListener('scroll', closeAllPopovers, { passive: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('click', onClickOutside)
  window.removeEventListener('scroll', closeAllPopovers)
})
</script>
