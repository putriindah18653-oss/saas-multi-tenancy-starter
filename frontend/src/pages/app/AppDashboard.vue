<template>
  <div class="space-y-4">
    <h2 class="text-xl font-semibold text-slate-900">App Dashboard</h2>
    <div class="rounded-xl border border-slate-200 bg-white p-4">
      <p class="text-sm text-slate-600">Selamat datang, {{ auth.user?.full_name || auth.user?.email || 'user' }}.</p>
      <p class="mt-2 text-sm text-slate-500">Halaman tenant CRUD masuk Task 10.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
const auth = useAuthStore()
</script>
