SaaS Multi-Tenant Starter

Project ini adalah starter kit / base project untuk membangun berbagai aplikasi SaaS multi-tenant di masa depan, misalnya POS, CRM, HRM, inventory, billing, dan aplikasi SaaS lainnya.

Project harus dibuat sebagai aplikasi baru dari nol, belum menggunakan data real, tetapi secara desain aplikasi harus production-minded, aman, rapi, scalable, dan siap dikembangkan.

Stack utama:

- Backend: Golang
- Database: PostgreSQL
- Cache/session/rate limit: Redis
- Frontend: Vue 3
- Styling: Tailwind CSS
- Deployment: Docker dan Docker Compose
- Repository: Monorepo

Struktur repository yang diinginkan:

my-project/
├── backend/
│   ├── cmd/
│   ├── internal/
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   ├── Dockerfile
│   └── package.json
├── docker-compose.yml
├── docker-compose.prod.yml
├── README.md
└── PROJECT.md

Tujuan utama project:

1. Menyediakan pondasi aplikasi SaaS multi-tenant.
2. Menjamin pemisahan data antar tenant.
3. Menyediakan sistem autentikasi dan otorisasi berbasis RBAC.
4. Menyediakan struktur backend dan frontend yang clean, modular, dan mudah dikembangkan.
5. Menyediakan docker compose untuk development dan production.
6. Menyediakan dokumentasi teknis di README.md dan PROJECT.md.

==================================================
KONSEP MULTI-TENANCY
==================================================

Gunakan pendekatan multi-tenant dengan shared database dan tenant_id pada tabel-tabel tenant-scoped.

Aturan penting:

- Setiap data milik tenant wajib memiliki tenant_id.
- User tenant hanya boleh membaca dan mengubah data pada tenant miliknya.
- Tidak boleh ada query tenant-scoped tanpa filter tenant_id.
- Backend harus memiliki middleware untuk resolve tenant context.
- Tenant context bisa didapat dari subdomain, header, atau membership user.
- Untuk tahap awal, gunakan header `X-Tenant-ID` saat development.
- Desain kode agar ke depan bisa mendukung subdomain seperti:
  - tenant-a.app.com
  - tenant-b.app.com

Wajib buat proteksi agar tidak terjadi kebocoran antar tenant.

Minimal tabel tenant:

- tenants
- users
- user_tenants
- roles
- permissions
- role_permissions
- audit_logs

==================================================
RBAC
==================================================

Buat sistem RBAC dua level:

1. App-level roles
2. Tenant-level roles

App-level roles:

- owner-app
- admin

Role app-level yang disiapkan untuk pengembangan berikutnya:

- finance
- support
- marketing

Tenant-level roles:

- owner-tenant
- admin
- finance
- support

Aturan RBAC:

- owner-app adalah superadmin aplikasi.
- owner-app dapat melihat dan mengelola semua tenant.
- admin app dapat membantu operasional aplikasi, tetapi tidak otomatis boleh masuk ke semua data tenant kecuali diberi izin.
- owner-tenant adalah pemilik tenant.
- owner-tenant dapat mengelola user dalam tenant miliknya.
- admin tenant dapat mengelola konfigurasi tenant.
- finance tenant hanya boleh mengakses fitur finance tenant.
- support tenant hanya boleh mengakses fitur support tenant.

Permission harus dibuat modular, misalnya:

App permissions:

- app.tenants.read
- app.tenants.create
- app.tenants.update
- app.tenants.delete
- app.users.read
- app.users.manage
- app.audit.read

Tenant permissions:

- tenant.dashboard.read
- tenant.users.read
- tenant.users.invite
- tenant.users.update
- tenant.users.remove
- tenant.settings.read
- tenant.settings.update
- tenant.billing.read
- tenant.billing.manage
- tenant.support.read
- tenant.support.manage

Implementasikan helper/middleware untuk mengecek permission, misalnya:

- RequireAuth
- RequireAppRole
- RequireTenantRole
- RequirePermission
- RequireTenantAccess

==================================================
BACKEND GOLANG
==================================================

Buat backend menggunakan Golang dengan struktur clean architecture sederhana.

Struktur backend yang diinginkan:

backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── redis/
│   ├── middleware/
│   ├── auth/
│   ├── rbac/
│   ├── tenant/
│   ├── user/
│   ├── audit/
│   ├── common/
│   └── http/
│       ├── handler/
│       ├── router/
│       └── response/
├── migrations/
├── Dockerfile
├── go.mod
└── go.sum

Gunakan package yang stabil dan umum.

Rekomendasi:

- HTTP router: chi atau gin
- Database: pgx atau sqlc
- Migration: golang-migrate
- Config: env-based config
- Password hashing: bcrypt atau argon2
- JWT auth untuk access token
- Refresh token bisa disiapkan sederhana

Backend harus menyediakan minimal endpoint:

Public:

- POST /api/v1/auth/register-owner
- POST /api/v1/auth/login
- POST /api/v1/auth/refresh
- POST /api/v1/auth/logout

App admin:

- GET /api/v1/app/tenants
- POST /api/v1/app/tenants
- GET /api/v1/app/tenants/:id
- PATCH /api/v1/app/tenants/:id
- DELETE /api/v1/app/tenants/:id

Tenant:

- GET /api/v1/tenant/me
- GET /api/v1/tenant/dashboard
- GET /api/v1/tenant/users
- POST /api/v1/tenant/users/invite
- PATCH /api/v1/tenant/users/:id/role
- DELETE /api/v1/tenant/users/:id

RBAC:

- GET /api/v1/me
- GET /api/v1/me/permissions
- GET /api/v1/me/tenants

Health:

- GET /health
- GET /ready

Backend wajib memiliki:

- request ID middleware
- logging middleware
- recovery middleware
- auth middleware
- tenant context middleware
- permission middleware
- error response standard
- audit log untuk aksi penting
- database migration
- seed default roles dan permissions
- environment config
- graceful shutdown

==================================================
DATABASE POSTGRESQL
==================================================

Gunakan PostgreSQL.

Buat migration SQL awal.

Minimal schema:

users:
- id UUID primary key
- name
- email unique
- password_hash
- app_role nullable
- is_active
- created_at
- updated_at

tenants:
- id UUID primary key
- name
- slug unique
- status
- created_at
- updated_at

user_tenants:
- id UUID primary key
- user_id
- tenant_id
- role
- is_active
- created_at
- updated_at
- unique(user_id, tenant_id)

roles:
- id UUID primary key
- scope app/tenant
- name
- description

permissions:
- id UUID primary key
- scope app/tenant
- key unique
- description

role_permissions:
- role_id
- permission_id

audit_logs:
- id UUID primary key
- actor_user_id nullable
- tenant_id nullable
- action
- resource_type
- resource_id nullable
- metadata JSONB
- ip_address
- user_agent
- created_at

Tambahkan index penting:

- users.email
- tenants.slug
- user_tenants.user_id
- user_tenants.tenant_id
- audit_logs.tenant_id
- audit_logs.actor_user_id
- audit_logs.created_at

Wajib pastikan semua query tenant-scoped menggunakan tenant_id.

==================================================
REDIS
==================================================

Gunakan Redis untuk:

- cache ringan
- blacklist/logout token jika diperlukan
- rate limiting login endpoint
- future session support

Buat wrapper sederhana di backend/internal/redis.

==================================================
FRONTEND VUE
==================================================

Buat frontend menggunakan Vue 3 dan Tailwind CSS.

Struktur frontend:

frontend/
├── src/
│   ├── app/
│   ├── components/
│   ├── layouts/
│   ├── pages/
│   │   ├── auth/
│   │   ├── app/
│   │   └── tenant/
│   ├── router/
│   ├── stores/
│   ├── services/
│   └── main.ts
├── Dockerfile
├── package.json
└── vite.config.ts

Gunakan:

- Vue 3
- Vite
- Tailwind CSS
- Pinia
- Vue Router
- Axios atau fetch wrapper

Frontend minimal memiliki halaman:

Auth:

- Login
- Register owner app pertama

App admin:

- App dashboard
- Tenant list
- Create tenant
- Tenant detail

Tenant:

- Tenant dashboard
- Tenant users
- Invite user
- Tenant settings placeholder

Layout:

- AuthLayout
- AppAdminLayout
- TenantLayout

Frontend wajib memiliki:

- route guard berdasarkan auth
- route guard berdasarkan role/permission
- API client wrapper
- token storage
- logout
- tenant switcher sederhana
- permission helper

==================================================
DOCKER
==================================================

Buat docker-compose.yml untuk development.

Services:

- backend
- frontend
- postgres
- redis

Buat docker-compose.prod.yml untuk production-oriented setup.

Development compose harus mendukung:

- backend hot reload jika memungkinkan
- frontend dev server
- postgres volume
- redis volume
- network internal

Contoh port:

- backend: 8080
- frontend: 5173
- postgres: 5432
- redis: 6379

Gunakan .env.example untuk environment variable.

Environment minimal:

APP_ENV=development
APP_PORT=8080
DATABASE_URL=postgres://postgres:postgres@postgres:5432/saas_starter?sslmode=disable
REDIS_ADDR=redis:6379
JWT_ACCESS_SECRET=change-me-access-secret
JWT_REFRESH_SECRET=change-me-refresh-secret
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_HOURS=168
CORS_ALLOWED_ORIGINS=http://localhost:5173

==================================================
SECURITY REQUIREMENTS
==================================================

Wajib perhatikan security dari awal:

- password harus di-hash, jangan simpan plain text
- JWT secret dari env
- CORS configurable
- input validation
- SQL injection protection
- tenant isolation wajib
- role/permission check wajib
- audit log untuk aksi penting
- jangan log password/token
- gunakan secure error response
- siap untuk production hardening

Tambahkan komentar pada bagian penting yang menjelaskan kenapa tenant_id wajib dicek.

==================================================
DELIVERABLE
==================================================

Bangun project secara bertahap.

Tahap 1:
- Buat struktur monorepo
- Buat docker-compose.yml
- Buat backend skeleton
- Buat frontend skeleton
- Buat README.md
- Buat PROJECT.md

Tahap 2:
- Buat database migration
- Buat koneksi PostgreSQL
- Buat koneksi Redis
- Buat health endpoint
- Buat seed roles dan permissions

Tahap 3:
- Buat auth register owner dan login
- Buat JWT middleware
- Buat tenant context middleware
- Buat RBAC middleware

Tahap 4:
- Buat tenant management endpoint
- Buat tenant user management endpoint
- Buat audit logging

Tahap 5:
- Buat frontend auth
- Buat layout app admin
- Buat layout tenant
- Buat route guard
- Buat tenant switcher

Tahap 6:
- Finalisasi Dockerfile
- Finalisasi README.md
- Finalisasi PROJECT.md
- Tambahkan instruksi run development dan production

==================================================
CARA KERJA YANG SAYA INGINKAN
==================================================

Kerjakan dengan cara berikut:

1. Jangan langsung membuat fitur terlalu kompleks.
2. Buat pondasi yang bersih, aman, dan mudah dikembangkan.
3. Gunakan kode yang jelas dan maintainable.
4. Setiap file penting harus memiliki isi yang benar-benar usable, bukan placeholder kosong.
5. Jangan membuat dependency yang tidak perlu.
6. Jika ada pilihan teknis, pilih yang paling sederhana tapi production-friendly.
7. Fokus pada tenant isolation, RBAC, dan struktur project.
8. Dokumentasikan keputusan arsitektur di PROJECT.md.
9. Pastikan project bisa dijalankan dengan docker compose.
10. Pastikan command dasar dijelaskan di README.md.

==================================================
OUTPUT YANG DIHARAPKAN
==================================================

Berikan hasil dalam bentuk file dan folder project.

Setelah selesai, tampilkan:

1. Struktur folder final.
2. Cara menjalankan development:
   docker compose up --build
3. Cara menjalankan migration.
4. Cara membuat owner-app pertama.
5. Cara login.
6. Contoh flow:
   - owner-app login
   - owner-app membuat tenant
   - owner-tenant login
   - owner-tenant mengundang admin tenant
7. Catatan security tenant isolation.
8. Next step pengembangan.

Mulai dari membuat struktur project monorepo sesuai spesifikasi di atas.