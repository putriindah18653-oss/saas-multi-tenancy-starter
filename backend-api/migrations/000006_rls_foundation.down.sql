DROP POLICY IF EXISTS audit_logs_delete_isolation ON audit_logs;
DROP POLICY IF EXISTS audit_logs_update_isolation ON audit_logs;
DROP POLICY IF EXISTS audit_logs_insert_isolation ON audit_logs;
DROP POLICY IF EXISTS audit_logs_select_isolation ON audit_logs;

DROP POLICY IF EXISTS tenant_settings_isolation ON tenant_settings;

DROP POLICY IF EXISTS user_tenants_delete_isolation ON user_tenants;
DROP POLICY IF EXISTS user_tenants_update_isolation ON user_tenants;
DROP POLICY IF EXISTS user_tenants_insert_isolation ON user_tenants;
DROP POLICY IF EXISTS user_tenants_select_isolation ON user_tenants;

ALTER TABLE IF EXISTS audit_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS tenant_settings DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS user_tenants DISABLE ROW LEVEL SECURITY;

DROP FUNCTION IF EXISTS app_is_platform_admin();
DROP FUNCTION IF EXISTS app_current_user_id();
DROP FUNCTION IF EXISTS app_current_tenant_id();
