<template>
  <aside class="rounded-[var(--radius-card)] border border-[var(--border)] bg-[var(--bg-sidebar)] p-3 lg:sticky lg:top-20 lg:self-start">
    <nav class="flex gap-1 overflow-x-auto lg:block lg:space-y-1">
      <RouterLink
        v-for="item in items"
        :key="item.to"
        :to="item.to"
        class="whitespace-nowrap rounded-[var(--radius-button)] px-3 py-2 text-sm font-medium text-[var(--text-secondary)] transition hover:bg-white/5 hover:text-[var(--text-primary)] lg:block"
        active-class="bg-[var(--accent)] text-white hover:bg-[var(--accent)] hover:text-white"
      >
        {{ item.label }}
      </RouterLink>
    </nav>
  </aside>
</template>

<script setup lang="ts">
defineProps<{ items: Array<{ to: string; label: string }> }>()
</script>
