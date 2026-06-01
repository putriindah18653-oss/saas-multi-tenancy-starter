<template>
  <div
    role="alert"
    class="rounded-[var(--radius-card)] border px-4 py-3 text-sm relative"
    :class="classes"
  >
    <p class="font-medium">{{ title }}</p>
    <p v-if="$slots.default" class="mt-1 opacity-90"><slot /></p>
    <button
      type="button"
      class="absolute right-2 top-2 rounded-full p-1 text-current opacity-50 hover:opacity-100 transition"
      :aria-label="dismissLabel || 'Close alert'"
      @click="$emit('dismiss')"
    >
      ✕
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  tone?: 'danger' | 'success' | 'warning' | 'info'
  title: string
  dismissLabel?: string
}>(), {
  tone: 'info',
})

defineEmits<{ dismiss: [] }>()

const classes = computed(() => ({
  danger: 'border-red-400/25 bg-red-400/10 text-[var(--danger)]',
  success: 'border-emerald-400/25 bg-emerald-400/10 text-[var(--success)]',
  warning: 'border-amber-400/25 bg-amber-400/10 text-[var(--warning)]',
  info: 'border-blue-400/25 bg-blue-400/10 text-[var(--info)]',
}[props.tone]))
</script>
