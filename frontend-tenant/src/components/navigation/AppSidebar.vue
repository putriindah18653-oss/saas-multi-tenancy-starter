<template>
  <aside class="rounded-xl border border-slate-200 bg-white p-3">
    <nav class="space-y-1">
      <RouterLink
        v-for="item in items"
        :key="item.to"
        :to="item.to"
        class="block rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-slate-100"
        active-class="bg-slate-900 text-white hover:bg-slate-900"
      >
        {{ item.label }}
      </RouterLink>
    </nav>
  </aside>
</template>

<script setup lang="ts">
defineProps<{ items: Array<{ to: string; label: string }> }>()
</script>
