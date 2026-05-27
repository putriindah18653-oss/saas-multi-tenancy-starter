# SaaS Multi-Tenancy Starter

Starter monorepo untuk aplikasi SaaS multi-tenant dengan isolasi data tenant pada shared database.

## Stack

- Backend: Go (chi, pgxpool)
- Database: PostgreSQL
- Cache: Redis
- Frontend: Vue 3 + Vite + TypeScript + Tailwind + Pinia
- Deploy: Docker Compose

## Quick start (development)

```bash
cp .env.example .env
docker compose config
docker compose up -d postgres redis
```

Jalankan migration:

```bash
docker run --rm \
  --network saas-multi-tenancy-starter_app_net \
  -v "$PWD/backend/migrations:/migrations" \
  migrate/migrate:v4.17.1 \
  -path=/migrations \
  -database "postgres://postgres:postgres@postgres:5432/saas_starter?sslmode=disable" \
  up
```

Jalankan app:

```bash
docker compose up --build
```

Port:

- Backend: http://localhost:8080
- Frontend: http://localhost:5173
- PostgreSQL: localhost:5432
- Redis: localhost:6379

## Migration

Up:

```bash
docker run --rm \
  --network saas-multi-tenancy-starter_app_net \
  -v "$PWD/backend/migrations:/migrations" \
  migrate/migrate:v4.17.1 \
  -path=/migrations \
  -database "postgres://postgres:postgres@postgres:5432/saas_starter?sslmode=disable" \
  up
```

Down 1 step:

```bash
docker run --rm \
  --network saas-multi-tenancy-starter_app_net \
  -v "$PWD/backend/migrations:/migrations" \
  migrate/migrate:v4.17.1 \
  -path=/migrations \
  -database "postgres://postgres:postgres@postgres:5432/saas_starter?sslmode=disable" \
  down 1
```

## Onboarding flow

1. Create first owner-app:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register-owner \
  -H "Content-Type: application/json" \
  -d '{"name":"Owner App","email":"owner@app.local","password":"StrongPass123!"}'
```

2. Login:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@app.local","password":"StrongPass123!"}'
```

3. Owner-app creates tenant:

```bash
curl -X POST http://localhost:8080/api/v1/app/tenants/ \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant Alpha","slug":"tenant-alpha"}'
```

4. Owner-tenant invites admin tenant:

```bash
curl -X POST http://localhost:8080/api/v1/tenant/users/invite \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H "X-Tenant-ID: <TENANT_ID>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tenant Admin","email":"admin@tenant.local","role":"admin"}'
```

## Frontend routes

- `/auth/register-owner`
- `/auth/login`
- `/app`
- `/app/tenants`
- `/tenant`
- `/tenant/users`

Frontend kirim `Authorization` dan `X-Tenant-ID` otomatis dari store tenant terpilih.

## Frontend UX notes

- Form auth sudah pakai `autocomplete` + placeholder aman (tanpa contoh token/secret nyata).
- Saat invite user tenant, temporary password ditampilkan sekali di UI (dengan tombol copy/hide).
- Hindari menyimpan temporary password di log, screenshot, atau ticket publik.

## Security notes

- Ganti JWT secrets sebelum shared/prod.
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

- Refresh token rotation + revoke list
- Rate limiting per route/tenant
- Observability (metrics, tracing)
- E2E tests auth/tenant/user
- CI lint/test/build + migration check
