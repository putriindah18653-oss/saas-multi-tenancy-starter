<template>
  <div class="min-h-screen bg-[var(--bg-app)] px-4 py-8 text-[var(--text-primary)]">
    <div class="mx-auto grid min-h-[calc(100vh-4rem)] w-full max-w-6xl items-center gap-8 lg:grid-cols-[1.05fr_0.95fr]">
      <section class="hidden rounded-[24px] border border-[var(--border)] bg-[var(--surface)] p-10 lg:block">
        <div class="inline-flex rounded-full border border-[var(--border)] bg-white/5 px-3 py-1 text-xs font-medium text-[var(--text-secondary)]">
          Owner Control Panel
        </div>
        <h1 class="mt-8 max-w-xl text-4xl font-semibold tracking-tight text-[var(--text-primary)]">
          SaaS Multi-Tenancy Starter
        </h1>
        <p class="mt-4 max-w-lg text-sm leading-6 text-[var(--text-muted)]">
          Kelola tenant, audit aktivitas, dan akses owner dari dashboard gelap yang konsisten.
        </p>
        <div class="mt-10 grid gap-3 text-sm text-[var(--text-secondary)]">
          <div class="rounded-[var(--radius-card)] border border-[var(--border)] bg-white/[0.03] p-4">
            <p class="font-medium text-[var(--text-primary)]">Secure by role</p>
            <p class="mt-1 text-[var(--text-muted)]">Route guard dan permission tetap mengikuti kontrak yang sudah ada.</p>
          </div>
          <div class="rounded-[var(--radius-card)] border border-[var(--border)] bg-white/[0.03] p-4">
            <p class="font-medium text-[var(--text-primary)]">Operational focus</p>
            <p class="mt-1 text-[var(--text-muted)]">Tenant management, quick action, dan audit trail dalam satu shell.</p>
          </div>
        </div>
      </section>

      <section class="mx-auto w-full max-w-md rounded-[24px] border border-[var(--border)] bg-[var(--surface)] p-6 shadow-2xl shadow-black/30 sm:p-8">
        <RouterView />
      </section>
    </div>
  </div>
</template>
