CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  app_role TEXT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, tenant_id)
);

CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scope TEXT NOT NULL CHECK (scope IN ('app','tenant')),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  UNIQUE(scope, name)
);

CREATE TABLE permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scope TEXT NOT NULL CHECK (scope IN ('app','tenant')),
  key TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE role_permissions (
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  PRIMARY KEY(role_id, permission_id)
);

CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  tenant_id UUID NULL REFERENCES tenants(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id UUID NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  ip_address TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_user_tenants_user_id ON user_tenants(user_id);
CREATE INDEX idx_user_tenants_tenant_id ON user_tenants(tenant_id);
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_actor_user_id ON audit_logs(actor_user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE UNIQUE INDEX idx_users_single_owner_app ON users(app_role) WHERE app_role = 'owner-app';

-- Demo-only platform owner seed.
-- Initial credentials: owner@app.local / DemoPass123!
-- Production rule: replace/remove this seed and provision owner-app through a one-time,
-- secret-backed bootstrap process before exposing the service.
INSERT INTO users (id, name, email, password_hash, app_role, is_active)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  'Platform Owner',
  'owner@app.local',
  crypt('DemoPass123!', gen_salt('bf')),
  'owner-app',
  TRUE
)
ON CONFLICT (email) DO UPDATE
SET name = EXCLUDED.name,
    app_role = EXCLUDED.app_role,
    is_active = EXCLUDED.is_active,
    updated_at = now();

INSERT INTO roles(scope, name, description) VALUES
('app','owner-app','Application owner'),('app','admin','Application admin'),('app','finance','Application finance'),('app','support','Application support'),('app','marketing','Application marketing'),
('tenant','owner-tenant','Tenant owner'),('tenant','admin','Tenant admin'),('tenant','finance','Tenant finance'),('tenant','support','Tenant support')
ON CONFLICT(scope, name) DO NOTHING;

INSERT INTO permissions(scope, key, description) VALUES
('app','app.tenants.read','Read tenants'),('app','app.tenants.create','Create tenants'),('app','app.tenants.update','Update tenants'),('app','app.tenants.delete','Delete tenants'),('app','app.users.read','Read app users'),('app','app.users.manage','Manage app users'),('app','app.audit.read','Read app audit'),('app','app.settings.read','Read app company settings'),('app','app.settings.update','Update app company settings'),
('tenant','tenant.dashboard.read','Read tenant dashboard'),('tenant','tenant.users.read','Read tenant users'),('tenant','tenant.users.invite','Invite tenant users'),('tenant','tenant.users.update','Update tenant users'),('tenant','tenant.users.remove','Remove tenant users'),('tenant','tenant.settings.read','Read tenant settings'),('tenant','tenant.settings.update','Update tenant settings'),('tenant','tenant.audit.read','Read tenant audit'),('tenant','tenant.billing.read','Read tenant billing'),('tenant','tenant.billing.manage','Manage tenant billing'),('tenant','tenant.reports.read','Read tenant reports'),('tenant','tenant.support.manage','Manage tenant support')
ON CONFLICT(key) DO NOTHING;

INSERT INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.scope = r.scope
WHERE r.name IN ('owner-app','owner-tenant')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.scope = r.scope AND p.key NOT LIKE '%.delete' AND p.key NOT LIKE '%.remove'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;
