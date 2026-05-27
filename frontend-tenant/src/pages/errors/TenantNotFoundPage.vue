<template>
  <div class="mx-auto max-w-2xl rounded-xl border border-slate-200 bg-white p-6 text-center">
    <p class="text-sm font-semibold uppercase tracking-wide text-amber-600">Tenant required</p>
    <h1 class="mt-2 text-2xl font-semibold text-slate-900">No active tenant selected</h1>
    <p class="mt-2 text-sm text-slate-600">Select a valid tenant membership before opening tenant workspace pages.</p>
    <RouterLink to="/tenant" class="mt-4 inline-flex rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white">
      Back to tenant dashboard
    </RouterLink>
  </div>
</template>
