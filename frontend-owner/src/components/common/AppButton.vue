<template>
  <button
    :type="type"
    class="inline-flex min-h-10 items-center justify-center gap-2 px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-55"
    :class="variantClass"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
    type?: 'button' | 'submit' | 'reset'
  }>(),
  { variant: 'primary', type: 'button' },
)

const variantClass = computed(() => {
  const base = 'rounded-[var(--radius-button)] border'
  return {
    primary: `${base} border-transparent bg-[var(--button-primary-bg)] text-white hover:bg-[var(--button-primary-bg-hover)]`,
    secondary: `${base} border-[var(--border-strong)] bg-[var(--button-secondary-bg)] text-[var(--text-primary)] hover:bg-white/10`,
    danger: `${base} border-red-400/30 bg-[var(--button-danger-bg)] text-[var(--danger)] hover:bg-red-400/20`,
    ghost: 'rounded-[var(--radius-button)] border border-transparent text-[var(--text-secondary)] hover:bg-white/5 hover:text-[var(--text-primary)]',
  }[props.variant]
})
</script>
