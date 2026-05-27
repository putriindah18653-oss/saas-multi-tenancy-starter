DROP TABLE IF EXISTS tenant_settings;
DROP TABLE IF EXISTS refresh_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS must_change_password;
