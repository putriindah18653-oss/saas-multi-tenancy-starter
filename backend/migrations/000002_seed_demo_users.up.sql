-- Seed demo users and tenant (idempotent)
-- Demo credentials:
--   Platform owner : owner@app.local / DemoPass123!
--   Tenant admin   : admin@tenant.local / DemoPass123!

INSERT INTO users (id, name, email, password_hash, app_role, is_active)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  'Demo Platform Owner',
  'owner@app.local',
  crypt('DemoPass123!', gen_salt('bf')),
  'owner-app',
  TRUE
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO tenants (id, name, slug, status)
VALUES (
  '22222222-2222-2222-2222-222222222222',
  'Tenant Alpha Demo',
  'tenant-alpha-demo',
  'active'
)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO users (id, name, email, password_hash, app_role, is_active)
VALUES (
  '33333333-3333-3333-3333-333333333333',
  'Demo Tenant Admin',
  'admin@tenant.local',
  crypt('DemoPass123!', gen_salt('bf')),
  NULL,
  TRUE
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_tenants (id, user_id, tenant_id, role, is_active)
VALUES (
  '44444444-4444-4444-4444-444444444444',
  '33333333-3333-3333-3333-333333333333',
  '22222222-2222-2222-2222-222222222222',
  'admin',
  TRUE
)
ON CONFLICT (user_id, tenant_id) DO UPDATE
SET role = EXCLUDED.role,
    is_active = EXCLUDED.is_active,
    updated_at = now();
