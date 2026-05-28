<template>
  <section class="rounded-[var(--radius-kpi)] border border-[var(--border)] bg-[var(--surface)] p-4">
    <p class="text-xs font-medium uppercase tracking-wide text-[var(--text-muted)]">{{ label }}</p>
    <div class="mt-2 flex items-end justify-between gap-3">
      <p class="text-2xl font-semibold text-[var(--text-primary)]">{{ value }}</p>
      <p v-if="hint" class="text-xs text-[var(--text-muted)]">{{ hint }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{ label: string; value: string | number; hint?: string }>()
</script>
