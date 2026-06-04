# SaaS Multi-Tenancy Starter

Monorepo starter SaaS multi-tenant dengan backend Go, PostgreSQL shared database, Redis, dan dua frontend Vue: owner console dan tenant console.

## Stack

- Backend: Go, chi, pgxpool
- Database: PostgreSQL
- Cache: Redis
- Frontend owner: Vue 3, Vite, TypeScript, Tailwind, Pinia
- Frontend tenant: Vue 3, Vite, TypeScript, Tailwind, Pinia
- Deploy/local dev: Docker Compose

## Struktur repo

```text
backend-api/       Go API, migrations, upload storage
frontend-owner/    Owner/app console
frontend-tenant/   Tenant console
docker-compose.yml Development stack
docker-compose.prod.yml Production-style stack
```

## Quick start development

```bash
cp .env.example .env
docker compose config
docker compose up -d postgres redis
docker compose up --build
```

Port default:

- Backend API: http://localhost:8080
- Owner frontend: http://localhost:5173
- Tenant frontend: http://localhost:5174
- PostgreSQL: localhost:5432
- Redis: localhost:6379

LAN access:

- Owner frontend: `http://<host-lan-ip>:5173`
- Tenant frontend: `http://<host-lan-ip>:5174`
- Backend API: `http://<host-lan-ip>:8080`

Untuk local/LAN development, biarkan `VITE_API_BASE_URL` kosong agar browser menurunkan API host dari host frontend saat ini. Jika diset ke `http://localhost:8080/api/v1`, device LAN lain akan memanggil localhost miliknya sendiri, bukan mesin dev.

Jika LAN IP berubah, set `VITE_HMR_HOST` sebelum start frontend:

```bash
VITE_HMR_HOST=<host-lan-ip> docker compose up -d frontend-owner frontend-tenant
```

## Development hot reload

`docker-compose.yml` menjalankan kedua frontend dalam Vite dev mode dengan bind-mounted source dan polling enabled.

```bash
docker compose up -d
docker compose logs -f frontend-owner frontend-tenant
```

Restart frontend setelah dependency/config berubah:

```bash
docker compose up -d --force-recreate frontend-owner frontend-tenant
```

Catatan dev:

- `frontend-owner` source mounted: `./frontend-owner:/app`
- `frontend-tenant` source mounted: `./frontend-tenant:/app`
- `node_modules` disimpan di Docker named volumes
- `CHOKIDAR_USEPOLLING=true` untuk file watching di bind mounts
- `VITE_DEV_SERVER_HOST`, `VITE_DEV_SERVER_PORT`, `VITE_HMR_HOST`, dan `VITE_HMR_PORT` tersedia di masing-masing frontend `.env.example` untuk dev/HMR lewat Docker atau LAN.
- Untuk production frontend, set `VITE_API_BASE_URL` eksplisit. Dev fallback otomatis ke `http(s)://<frontend-host>:8080/api/v1`, tetapi production build tidak boleh bergantung pada fallback.

## Test dan build

```bash
cd backend-api && go test ./...
cd frontend-owner && npm run build && npm run test
cd frontend-tenant && npm run build && npm run test
```

## Dev seed login dan onboarding

Schema migration tidak membuat demo user. Untuk local/dev saja, apply seed manual setelah migration:

```bash
cd backend-api
psql "$DATABASE_URL" -f seeds/dev/demo_users.up.sql
```

Seed local/dev membuat akun demo:

- Email: `owner@app.local`
- Password: `DemoPass123!`
- Email: `admin@tenant.local`
- Password: `DemoPass123!`

Jangan jalankan seed ini pada shared/staging/prod. Cleanup local/dev:

```bash
cd backend-api
psql "$DATABASE_URL" -f seeds/dev/demo_users.down.sql
```

Flow awal:

1. Login owner lewat owner frontend atau API `POST /api/v1/auth/login`.
2. Buat tenant dari owner console.
3. Creator otomatis menjadi `owner-tenant` di tenant baru.
4. Invite user/admin tenant dari tenant user management.
5. User baru menerima `temporary_password` sekali; jangan log atau share di tempat publik.

## Frontend routes

Owner frontend:

- `/auth/login`
- `/app`
- `/app/tenants`
- `/app/company-settings`
- `/app/profile`
- `/app/audit`

Tenant frontend:

- `/auth/login`
- `/auth/change-password`
- `/tenant`
- `/tenant/profile`
- `/tenant/users`
- `/tenant/settings`
- `/tenant/audit`

Tenant frontend mengirim `Authorization` dan `X-Tenant-ID` otomatis dari selected tenant store.

## Fitur utama

- Auth JWT access/refresh token.
- Refresh token persistence, rotation, session revoke, logout revoke.
- Forced password change untuk temporary password.
- Redis-backed rate limit untuk login/refresh.
- Owner tenant management.
- Tenant user invite dan role-based access control.
- Tenant settings: display name, logo URL, timezone, locale, currency.
- Owner company settings dan logo upload.
- Profile settings owner/tenant: nama, phone, address, bio, avatar upload, change password.
- Owner dan tenant audit log UI/API.
- Responsive sidebar dan top navigation untuk owner/tenant console.
- Dark/light theme persistence.

## Uploads

Backend menyimpan upload di `backend-api/storage/uploads`.

Avatar/profile dan logo memakai upload endpoint backend, lalu frontend menyimpan path hasil upload seperti `/uploads/...`. Validasi keamanan tetap harus dilakukan di backend. Client hanya memberi validasi UX.

## Build dan checks

Frontend tenant:

```bash
cd frontend-tenant
npm run build
```

Frontend owner:

```bash
cd frontend-owner
npm run build
```

Backend:

```bash
cd backend-api
go test ./...
```

Saat ini script test frontend belum tersedia. Tambahkan Vitest/Playwright bila butuh automated UI coverage.

## Production

```bash
cp .env.example .env
docker compose -f docker-compose.prod.yml config
docker compose -f docker-compose.prod.yml up -d
```

Env penting production:

- `POSTGRES_PASSWORD`
- `DATABASE_URL`
- `JWT_ACCESS_SECRET`
- `JWT_REFRESH_SECRET`
- `JWT_REFRESH_COOKIE_SECURE=true`
- `JWT_REFRESH_COOKIE_SAME_SITE=lax` (atau `none` bila frontend/API beda site; wajib HTTPS/secure cookie)
- `CORS_ALLOWED_ORIGINS`

## Security notes

- Set `JWT_ACCESS_SECRET` dan `JWT_REFRESH_SECRET`; jangan pakai secret kosong.
- Refresh token dikirim via HttpOnly cookie path `/api/v1/auth`; frontend tidak menyimpan token di localStorage/sessionStorage.
- Jangan apply `backend-api/seeds/dev/*` ke shared/staging/prod.
- Jangan log password, token, refresh token, temporary password, atau secret.
- Query tenant wajib filter `tenant_id`.
- `X-Tenant-ID` wajib divalidasi terhadap membership user.
- Gunakan HTTPS pada production.
- Batasi `CORS_ALLOWED_ORIGINS` sesuai domain deployment.

## Roadmap singkat

- CI lint/test/build + migration check.
- Vitest component tests untuk profile/upload/sidebar/settings.
- Playwright smoke tests untuk auth, tenant navigation, profile, settings.
- Observability metrics/tracing.
