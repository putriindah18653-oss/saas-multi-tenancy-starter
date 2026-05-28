<template>
  <div
    role="alert"
    class="rounded-[var(--radius-card)] border px-4 py-3 text-sm"
    :class="classes"
  >
    <p class="font-medium">{{ title }}</p>
    <p v-if="$slots.default" class="mt-1 opacity-90"><slot /></p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ tone?: 'danger' | 'success' | 'warning' | 'info'; title: string }>(), {
  tone: 'info',
})

const classes = computed(() => ({
  danger: 'border-red-400/25 bg-red-400/10 text-[var(--danger)]',
  success: 'border-emerald-400/25 bg-emerald-400/10 text-[var(--success)]',
  warning: 'border-amber-400/25 bg-amber-400/10 text-[var(--warning)]',
  info: 'border-blue-400/25 bg-blue-400/10 text-[var(--info)]',
}[props.tone]))
</script>
