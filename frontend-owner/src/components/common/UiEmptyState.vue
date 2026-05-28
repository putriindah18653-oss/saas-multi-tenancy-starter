<template>
  <div class="rounded-[var(--radius-card)] border border-dashed border-[var(--border-strong)] bg-white/[0.03] px-6 py-10 text-center">
    <p class="text-base font-medium text-[var(--text-primary)]">{{ title }}</p>
    <p v-if="description" class="mx-auto mt-2 max-w-md text-sm text-[var(--text-muted)]">{{ description }}</p>
    <div v-if="$slots.actions" class="mt-5 flex justify-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; description?: string }>()
</script>
