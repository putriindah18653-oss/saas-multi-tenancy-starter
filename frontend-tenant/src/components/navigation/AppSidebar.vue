<template>
  <aside class="rounded-[var(--tenant-radius-card)] border border-[var(--tenant-border)] bg-[var(--tenant-bg-sidebar)] p-3">
    <nav class="flex gap-1 overflow-x-auto lg:block lg:space-y-1">
      <RouterLink
        v-for="item in items"
        :key="item.to"
        :to="item.to"
        class="whitespace-nowrap rounded-[var(--tenant-radius-button)] px-3 py-2 text-sm text-[var(--tenant-text-secondary)] transition hover:bg-white/[0.06] hover:text-[var(--tenant-text-primary)] lg:block"
        active-class="bg-[var(--tenant-accent)] text-slate-950 hover:bg-[var(--tenant-accent-hover)] hover:text-slate-950"
      >
        {{ item.label }}
      </RouterLink>
    </nav>
  </aside>
</template>

<script setup lang="ts">
defineProps<{ items: Array<{ to: string; label: string }> }>()
</script>
