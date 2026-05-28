<template>
  <section class="rounded-[var(--tenant-radius-kpi)] border border-[var(--tenant-border)] bg-[var(--tenant-surface)] p-4">
    <p class="text-xs font-medium uppercase tracking-wide text-[var(--tenant-text-muted)]">{{ label }}</p>
    <div class="mt-2 flex items-end justify-between gap-3">
      <p class="text-2xl font-semibold text-[var(--tenant-text-primary)]">{{ value }}</p>
      <p v-if="hint" class="text-xs text-[var(--tenant-text-muted)]">{{ hint }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{ label: string; value: string | number; hint?: string }>()
</script>
