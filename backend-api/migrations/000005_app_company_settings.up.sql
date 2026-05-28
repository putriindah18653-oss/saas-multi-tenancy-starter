CREATE TABLE IF NOT EXISTS app_settings (
  id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
  display_name TEXT NOT NULL DEFAULT 'PortalOnline',
  legal_name TEXT NOT NULL DEFAULT '',
  logo_url TEXT NOT NULL DEFAULT '',
  website_url TEXT NOT NULL DEFAULT '',
  support_email TEXT NOT NULL DEFAULT '',
  support_phone TEXT NOT NULL DEFAULT '',
  timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
  locale TEXT NOT NULL DEFAULT 'id-ID',
  currency TEXT NOT NULL DEFAULT 'IDR',
  address TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO app_settings (id, display_name)
VALUES (TRUE, 'PortalOnline')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions(scope, key, description) VALUES
('app','app.settings.read','Read app company settings'),
('app','app.settings.update','Update app company settings')
ON CONFLICT(key) DO NOTHING;

INSERT INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.scope = r.scope
WHERE r.scope = 'app'
  AND r.name IN ('owner-app','admin')
  AND p.key IN ('app.settings.read','app.settings.update')
ON CONFLICT DO NOTHING;
