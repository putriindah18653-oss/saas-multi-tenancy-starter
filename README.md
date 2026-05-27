# SaaS Multi-Tenancy Starter

Starter monorepo untuk aplikasi SaaS multi-tenant dengan isolasi data tenant pada shared database.

## Stack

- Backend: Go (chi, pgxpool)
- Database: PostgreSQL
- Cache: Redis
- Frontend owner: Vue 3 + Vite + TypeScript + Tailwind + Pinia
- Frontend tenant: Vue 3 + Vite + TypeScript + Tailwind + Pinia
- Deploy: Docker Compose

## Quick start development

```bash
cp .env.example .env
docker compose config
docker compose up -d postgres redis
```

Run migration dari `backend-api/migrations`, lalu jalankan app:

```bash
docker compose up --build
```

Port:

- Backend: http://localhost:8080
- Owner frontend: http://localhost:5173
- Tenant frontend: http://localhost:5174
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## Onboarding flow

1. Run migrations. Migration hard-seed satu owner-app untuk local/demo:

- Email: `owner@app.local`
- Password: `DemoPass123!`

Rotate password ini sebelum shared/prod.

2. Login ke endpoint:

```text
POST /api/v1/auth/login
Content-Type: application/json
Body: {"email":"owner@app.local","password":"DemoPass123!"}
```

Simpan `data.access_token`. Response login pakai field snake_case seperti `app_role` dan `tenant_memberships`.

3. Owner-app buat tenant:

```text
POST /api/v1/app/tenants
Authorization: Bearer <TOKEN>
Content-Type: application/json
Body: {"name":"Tenant Alpha","slug":"tenant-alpha"}
```

Creator otomatis menjadi `owner-tenant` di tenant baru.

4. Owner-tenant invite admin tenant:

```text
POST /api/v1/tenant/users/invite
Authorization: Bearer <TOKEN>
X-Tenant-ID: <TENANT_ID>
Content-Type: application/json
Body: {"name":"Tenant Admin","email":"admin@tenant.local","role":"admin"}
```

Untuk user baru, response invite menyertakan `temporary_password`. Untuk email yang sudah ada, user hanya ditautkan/diaktifkan ke tenant dan `temporary_password` tidak dikirim.

## Frontend routes

Owner frontend:

- `/auth/login`
- `/app`
- `/app/tenants`

Tenant frontend:

- `/auth/login`
- `/tenant`
- `/tenant/users`

Tenant frontend mengirim `Authorization` dan `X-Tenant-ID` otomatis dari selected tenant store.

## Frontend UX notes

- Form auth sudah pakai `autocomplete` + placeholder aman.
- Saat invite user tenant, temporary password ditampilkan sekali di UI jika backend membuat user baru.
- Hindari menyimpan temporary password di log, screenshot, atau ticket publik.

## Security notes

- Set `JWT_ACCESS_SECRET` dan `JWT_REFRESH_SECRET`; backend menolak start/token jika secret kosong.
- Ganti/rotate password seeded owner sebelum shared/prod.
- Jangan log password/token/secret.
- Query tenant wajib filter `tenant_id`.
- `X-Tenant-ID` wajib divalidasi ke membership user.
- Gunakan HTTPS pada production.

## Production

```bash
cp .env.example .env
docker compose -f docker-compose.prod.yml config
docker compose -f docker-compose.prod.yml up -d
```

Required env production:

- `POSTGRES_PASSWORD`
- `DATABASE_URL`
- `JWT_ACCESS_SECRET`
- `JWT_REFRESH_SECRET`
- `CORS_ALLOWED_ORIGINS`

## Next steps

- Observability metrics/tracing
- E2E tests auth/tenant/user
- CI lint/test/build + migration check

## Security/features added after hardening

- Refresh token persistence, rotation, session list, and revoke on logout/session revoke.
- Change password revokes active refresh sessions and clears temporary-password requirement.
- Redis-backed rate limit for login and refresh endpoints.
- Tenant settings API/UI for display name, logo URL, timezone, locale, and currency.
- Owner and tenant audit log API/UI for security/admin actions.
