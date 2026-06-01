# Database Schema — PostgreSQL Multi-Tenant with RLS

## Strategy: Row-Level Security (RLS)

The best choice for a single-instance VPS. All tenants live in one database, with security enforced at the PostgreSQL level.

### Advantages vs alternatives
| Strategy | Suitable for | Drawbacks |
|----------|------------|------------|
| **RLS (chosen)** | VPS, <1000 tenants | More complex setup up front |
| Schema per tenant | 10-100 tenants, strong isolation | Hard to maintain, heavy migrations |
| DB per tenant | Enterprise, strict compliance | Very expensive on a VPS |

---

## Core Tables

### Tenants (Platform Level)
```sql
CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          VARCHAR(100) UNIQUE NOT NULL,  -- kabarsiang
    name          VARCHAR(255) NOT NULL,
    plan          VARCHAR(50) DEFAULT 'free',    -- free | starter | pro | enterprise (cache of subscriptions.plan_code; see billing.md)
    status        VARCHAR(50) DEFAULT 'active',  -- provisioning | trial | active | past_due | suspended | canceled (see tenant-onboarding.md + billing.md)
    primary_domain VARCHAR(255),                 -- kabarsiang.id (nullable, can be null)
    owner_email   VARCHAR(255) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,         -- soft verification, non-blocking (tenant-onboarding.md)
    settings      JSONB DEFAULT '{}',            -- feature flags, branding, etc.
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Tenant domains (1 tenant can have many domains)
CREATE TABLE tenant_domains (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    domain      VARCHAR(255) UNIQUE NOT NULL,   -- kabarsiang.id
    type        VARCHAR(50) NOT NULL,           -- public | dashboard | sso
    is_primary  BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tenant_domains_domain ON tenant_domains(domain);
CREATE INDEX idx_tenant_domains_tenant ON tenant_domains(tenant_id);
```

### Platform Users (Platform Owner)
```sql
CREATE TABLE platform_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50) NOT NULL,   -- superuser | admin
    name          VARCHAR(255),
    is_active     BOOLEAN DEFAULT TRUE,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);
-- NO tenant_id, does not use RLS
```

### Tenant Users (Tenant Level — uses RLS)
```sql
CREATE TABLE tenant_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),           -- nullable if SSO only
    name          VARCHAR(255),
    role          VARCHAR(100) NOT NULL,  -- owner-tenant | admin | [custom]
    is_active     BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- Composite index — ALWAYS put tenant_id first
CREATE INDEX idx_tenant_users_tenant_email ON tenant_users(tenant_id, email);
CREATE INDEX idx_tenant_users_tenant_id    ON tenant_users(tenant_id);
```

### Custom Roles per Tenant
```sql
CREATE TABLE tenant_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    permissions JSONB DEFAULT '[]',  -- ["post:write", "post:publish", "user:read"]
    is_system   BOOLEAN DEFAULT FALSE, -- TRUE for owner-tenant and admin (cannot be deleted)
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_tenant_roles_tenant ON tenant_roles(tenant_id);
```

---

## Row-Level Security Setup

> **The isolation guarantee rests on 3 things that ALL MUST be present.** Missing any one = isolation leaks silently:
> 1. `ENABLE` **and** `FORCE` Row Level Security on every tenant table.
> 2. The application connects with a role that is **subject to** RLS (not a superuser, not `BYPASSRLS`, and FORCE closes the table-owner gap).
> 3. The policy has both `USING` (reads) **and** `WITH CHECK` (writes) — both.

### Why FORCE is required (not just ENABLE)

`ENABLE ROW LEVEL SECURITY` **does not apply to the table owner** nor to roles with the `BYPASSRLS` attribute. If migrations are run by `app_user`, then `app_user` becomes the table owner → RLS is **skipped entirely** even though it is a non-superuser. So "not a superuser" is NOT enough. `FORCE ROW LEVEL SECURITY` forces the policy to apply even for the owner — this is what closes the gap.

### Two Roles: Tenant (subject to RLS) vs Platform (bypass)

```sql
-- Role 1: tenant application connection — SUBJECT TO RLS
CREATE ROLE app_user LOGIN PASSWORD 'app_password' NOSUPERUSER NOBYPASSRLS;
GRANT CONNECT ON DATABASE yourdb TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;

-- Role 2: platform connection (superuser/owner admin) — BYPASS RLS for cross-tenant queries
-- Used ONLY by platform routes via a separate pool (see backend-golang.md: platformPool)
CREATE ROLE platform_user LOGIN PASSWORD 'platform_password' NOSUPERUSER BYPASSRLS;
GRANT CONNECT ON DATABASE yourdb TO platform_user;
GRANT USAGE ON SCHEMA public TO platform_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO platform_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO platform_user;
```

> **Migrations are run by a third role (owner/migrator), NOT `app_user`.** So that `app_user` never becomes the table owner. Even if it must be the owner, FORCE RLS still protects you — but separating the migration role is the best practice.

### Enable + FORCE RLS on every tenant table

```sql
ALTER TABLE tenant_users  ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_users  FORCE  ROW LEVEL SECURITY;
ALTER TABLE tenant_roles  ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_roles  FORCE  ROW LEVEL SECURITY;
-- ... ENABLE + FORCE for EVERY table that has tenant_id

-- platform_users, tenants, tenant_domains, plans = platform tables → do NOT use RLS
-- (accessed via platform_user / platformPool, not the tenant connection)
-- `plans` is the platform-managed billing catalog; tenant-scoped billing tables
-- (subscriptions, invoices, payments) DO use RLS — see billing.md.
```

### Helper: read the active tenant safely

```sql
-- Wrap current_setting in a function to centralize the empty-string guard.
-- missing_ok=true → if the GUC is not set yet, return '' (not an error).
-- NULLIF('', '') → NULL → cast to uuid is safe (does not throw '' invalid uuid error).
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
$$;
```

### RLS Policy — USING (reads) + WITH CHECK (writes)

```sql
-- the active tenant is set per transaction by Go:
--   SELECT set_config('app.current_tenant_id', $1, true)   (see backend-golang.md)

-- USING       → which rows are VISIBLE for SELECT/UPDATE/DELETE
-- WITH CHECK  → rows resulting from INSERT/UPDATE MUST satisfy this (prevent cross-tenant writes)
-- If the tenant is not set, current_tenant_id() = NULL → 0 rows visible, 0 rows writable.

CREATE POLICY tenant_isolation ON tenant_users
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());

CREATE POLICY tenant_isolation ON tenant_roles
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());
-- ... the same policy for EVERY tenant table
```

> **Why `WITH CHECK` must be explicit:** without `WITH CHECK`, PostgreSQL uses the `USING` expression for the write check on UPDATE, but **INSERT has no default `USING`** — meaning an INSERT could write another tenant's `tenant_id` if you rely on `USING` alone. Writing both explicitly closes this gap.

> **Platform bypass — explicit, not "set tenant_id = NULL".** A platform route does NOT set `app.current_tenant_id`. With an `app_user` connection that means 0 rows (safe by default). For cross-tenant queries, the platform uses the **`platform_user` pool (BYPASSRLS)** which skips the policy entirely. Never give `app_user` the `BYPASSRLS` attribute.

### Set Tenant in Golang (sqlc + pgx)

Tenant data access is wrapped in a transaction: `set_config` sets the active tenant, then the sqlc `Queries` run **in the same tx**. See `backend-golang.md` for the full `store.InTenant` helper + naming convention.

```go
// SET LOCAL does not accept the placeholder $1, so use set_config(name, value, is_local=true).
// is_local=true → scope to this tx only, reset automatically on commit/rollback.

func (s *Store) InTenant(ctx context.Context, fn func(q *db.Queries) error) error {
    t := tenant.MustFromContext(ctx)

    tx, err := s.pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    // Set the active tenant for this transaction
    if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant_id', $1, true)`, t.ID); err != nil {
        return err
    }

    if err := fn(db.New(tx)); err != nil { return err } // Queries bound to tx → RLS active
    return tx.Commit(ctx)
}

// Usage in a service — RLS automatically filters tenant_id
func (s *Service) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) {
    var users []db.ListUsersRow
    err := s.store.InTenant(ctx, func(q *db.Queries) error {
        var err error
        users, err = q.ListUsers(ctx) // sqlc generated; RLS filters the tenant
        return err
    })
    return users, err
}
```

---

## Migration Strategy (golang-migrate)

### Folder structure
```
migrations/
  000001_create_tenants.up.sql
  000001_create_tenants.down.sql
  000002_create_platform_users.up.sql
  000002_create_platform_users.down.sql
  000003_create_tenant_users.up.sql
  000003_create_tenant_users.down.sql
  000004_enable_rls.up.sql
  000004_enable_rls.down.sql
```

### Multi-Tenant Migration Tips
```sql
-- When adding a new column, always include tenant_id in any new index
-- Example: 000010_add_articles.up.sql
CREATE TABLE articles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title       VARCHAR(500) NOT NULL,
    content     TEXT,
    status      VARCHAR(50) DEFAULT 'draft',
    created_by  UUID REFERENCES tenant_users(id),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE articles ENABLE ROW LEVEL SECURITY;
ALTER TABLE articles FORCE  ROW LEVEL SECURITY;   -- required: close the table-owner gap

CREATE POLICY tenant_isolation ON articles
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());

CREATE INDEX idx_articles_tenant_status ON articles(tenant_id, status);
CREATE INDEX idx_articles_tenant_created ON articles(tenant_id, created_at DESC);
```

---

## Checklist for Every New Table

- [ ] Has a `tenant_id UUID NOT NULL REFERENCES tenants(id)` column
- [ ] `ENABLE ROW LEVEL SECURITY` **and** `FORCE ROW LEVEL SECURITY`
- [ ] `CREATE POLICY tenant_isolation` with **`USING` + `WITH CHECK`** using `current_tenant_id()`
- [ ] Composite index `(tenant_id, ...)` — `tenant_id` always in the first position
- [ ] No query without `tenant_id` in the WHERE (except platform routes via `platformPool`)
- [ ] Isolation test updated for this table (see "Tenant Isolation Testing")

---

## Query Performance Tips

```sql
-- GOOD: tenant_id at the front of the composite index
CREATE INDEX idx_users_tenant_email ON tenant_users(tenant_id, email);
SELECT * FROM tenant_users WHERE tenant_id = $1 AND email = $2;

-- BAD: query without tenant_id (full table scan)
SELECT * FROM tenant_users WHERE email = $2; -- ❌ don't do this

-- GOOD: pagination with tenant safety
SELECT * FROM articles
WHERE tenant_id = $1 AND status = 'published'
ORDER BY created_at DESC
LIMIT 20 OFFSET $2;
```

---

## Tenant Isolation Testing

Isolation must not be "trusted" — it must be proven and re-tested in every CI run. Two layers of testing: **(A)** configuration assertions (roles & RLS are correct) and **(B)** behavioral tests (tenant A cannot touch tenant B's data). Run both using an `app_user` connection (which is subject to RLS) against the test database.

### A. Configuration Assertions (catch misconfig)

```sql
-- 1. app_user MUST NOT be BYPASSRLS (if true → all isolation is dead)
SELECT rolname, rolbypassrls, rolsuper
FROM pg_roles WHERE rolname = 'app_user';
-- Expected: rolbypassrls = false, rolsuper = false

-- 2. Every tenant_id-bearing table MUST have rowsecurity AND forcerowsecurity = true.
--    This query lists tables that have a tenant_id column BUT whose RLS is incomplete.
--    The result MUST be empty; any rows = a table with a hole.
SELECT c.relname,
       c.relrowsecurity      AS enabled,
       c.relforcerowsecurity AS forced
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
  AND c.relkind = 'r'
  AND EXISTS (
      SELECT 1 FROM information_schema.columns col
      WHERE col.table_schema = 'public'
        AND col.table_name = c.relname
        AND col.column_name = 'tenant_id'
  )
  AND (c.relrowsecurity = false OR c.relforcerowsecurity = false);

-- 3. Every RLS-enabled table has a policy with WITH CHECK (qual + with_check populated).
--    The result MUST be empty.
SELECT tablename
FROM pg_policies
WHERE schemaname = 'public' AND (qual IS NULL OR with_check IS NULL);
```

### B. Behavioral Tests (Go + pgx)

```go
package isolation_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/stretchr/testify/require"
)

// setTenant mimics store.InTenant: set_config local inside the same tx.
func setTenant(t *testing.T, tx pgx.Tx, id uuid.UUID) {
    _, err := tx.Exec(context.Background(),
        `SELECT set_config('app.current_tenant_id', $1, true)`, id.String())
    require.NoError(t, err)
}

func TestTenantIsolation(t *testing.T) {
    ctx := context.Background()
    // CONNECTION MUST be app_user (subject to RLS) — not superuser/platform_user.
    conn, err := pgx.Connect(ctx, appUserDSN)
    require.NoError(t, err)
    defer conn.Close(ctx)

    // Seed: two tenants + one user each. Seeding uses platform/superuser
    // outside this test; here we assume tenantA & userA, tenantB & userB already exist.

    t.Run("SELECT only sees the active tenant", func(t *testing.T) {
        tx, _ := conn.Begin(ctx)
        defer tx.Rollback(ctx)
        setTenant(t, tx, tenantA)

        var count int
        require.NoError(t, tx.QueryRow(ctx,
            `SELECT count(*) FROM tenant_users WHERE id = $1`, userB).Scan(&count))
        require.Equal(t, 0, count, "tenant A MUST NOT see a tenant B user")
    })

    t.Run("No active tenant → 0 rows", func(t *testing.T) {
        tx, _ := conn.Begin(ctx)
        defer tx.Rollback(ctx)
        // deliberately NO set_config

        var count int
        require.NoError(t, tx.QueryRow(ctx,
            `SELECT count(*) FROM tenant_users`).Scan(&count))
        require.Equal(t, 0, count, "with no active tenant it must be 0 rows (fail-closed)")
    })

    t.Run("Cross-tenant INSERT rejected by WITH CHECK", func(t *testing.T) {
        tx, _ := conn.Begin(ctx)
        defer tx.Rollback(ctx)
        setTenant(t, tx, tenantA)

        // Try to write a row belonging to tenant B while the active tenant = A.
        _, err := tx.Exec(ctx,
            `INSERT INTO tenant_users (tenant_id, email, role) VALUES ($1, $2, $3)`,
            tenantB, "evil@b.com", "admin")
        require.Error(t, err, "WITH CHECK must reject an INSERT with another tenant_id")
    })

    t.Run("UPDATE cannot move a row to another tenant", func(t *testing.T) {
        tx, _ := conn.Begin(ctx)
        defer tx.Rollback(ctx)
        setTenant(t, tx, tenantA)

        ct, err := tx.Exec(ctx,
            `UPDATE tenant_users SET tenant_id = $1 WHERE id = $2`, tenantB, userA)
        // May error (WITH CHECK) or affect 0 rows — both mean the row did not move.
        if err == nil {
            require.Equal(t, int64(0), ct.RowsAffected())
        }
    })
}
```

### C. CI Integration

Add to the pipeline (see `deployment.md`):

```yaml
# .github/workflows/test.yml (snippet)
- name: RLS isolation tests
  env:
    APP_USER_DSN: postgres://app_user:test@localhost:5432/yourdb_test
  run: |
    # 1. Run all migrations as the migrator role (not app_user)
    migrate -path ./migrations -database "$MIGRATOR_DSN" up
    # 2. Configuration assertions: the part-A queries, fail the build if any rows
    psql "$MIGRATOR_DSN" -v ON_ERROR_STOP=1 -f ./scripts/assert_rls.sql
    # 3. Behavioral tests
    go test ./internal/isolation/... -run TestTenantIsolation -v
```

> **Required in CI, not optional.** A new table easily forgets `FORCE` or `WITH CHECK`. The part-A assertions catch this automatically: queries #2/#3 return rows → `assert_rls.sql` exits non-zero → red build. This is what turns "isolation" from a hope into a guarantee verified on every commit.
