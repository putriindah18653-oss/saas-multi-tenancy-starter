<template>
  <div class="mx-auto max-w-2xl rounded-xl border border-slate-200 bg-white p-6 text-center">
    <p class="text-sm font-semibold uppercase tracking-wide text-red-600">403 Forbidden</p>
    <h1 class="mt-2 text-2xl font-semibold text-slate-900">Access denied</h1>
    <p class="mt-2 text-sm text-slate-600">Your account does not have permission to open this page or perform this action.</p>
    <RouterLink :to="homeRoute" class="mt-4 inline-flex rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white">
      Back to dashboard
    </RouterLink>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const homeRoute = computed(() => ({ name: auth.defaultHomeRoute }))
</script>
