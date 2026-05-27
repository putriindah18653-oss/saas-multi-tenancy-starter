<template>
  <div class="min-h-screen bg-slate-100">
    <div class="mx-auto flex min-h-screen w-full max-w-6xl items-center justify-center p-6">
      <div class="grid w-full overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-lg md:grid-cols-2">
        <section class="hidden bg-slate-900 p-10 text-slate-100 md:block">
          <h1 class="text-2xl font-semibold">SaaS Multi-Tenancy Starter</h1>
          <p class="mt-3 text-slate-300">Auth flow dan layout dasar siap. Lanjut fitur halaman di task berikutnya.</p>
        </section>
        <section class="p-6 md:p-10">
          <RouterView />
        </section>
      </div>
    </div>
  </div>
</template>
