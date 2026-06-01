-- Tenant isolation foundation.
--
-- This migration is intentionally idempotent because the current migration runner does
-- not track applied versions. It enables RLS and installs policies, but does not use
-- FORCE ROW LEVEL SECURITY yet; application code is refactored to set tenant/user GUCs
-- before forcing policies in a later hardening step.

CREATE OR REPLACE FUNCTION app_current_tenant_id()
RETURNS UUID
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
  raw_value TEXT;
BEGIN
  raw_value := NULLIF(current_setting('app.current_tenant_id', true), '');
  IF raw_value IS NULL THEN
    RETURN NULL;
  END IF;
  RETURN raw_value::UUID;
EXCEPTION WHEN invalid_text_representation THEN
  RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION app_current_user_id()
RETURNS UUID
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
  raw_value TEXT;
BEGIN
  raw_value := NULLIF(current_setting('app.current_user_id', true), '');
  IF raw_value IS NULL THEN
    RETURN NULL;
  END IF;
  RETURN raw_value::UUID;
EXCEPTION WHEN invalid_text_representation THEN
  RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION app_is_platform_admin()
RETURNS BOOLEAN
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
  raw_value TEXT;
BEGIN
  raw_value := NULLIF(current_setting('app.is_platform_admin', true), '');
  IF raw_value IS NULL THEN
    RETURN FALSE;
  END IF;
  RETURN raw_value::BOOLEAN;
EXCEPTION WHEN invalid_text_representation THEN
  RETURN FALSE;
END;
$$;

ALTER TABLE user_tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS user_tenants_select_isolation ON user_tenants;
CREATE POLICY user_tenants_select_isolation
ON user_tenants
FOR SELECT
USING (
  app_is_platform_admin()
  OR tenant_id = app_current_tenant_id()
  OR user_id = app_current_user_id()
);

DROP POLICY IF EXISTS user_tenants_insert_isolation ON user_tenants;
CREATE POLICY user_tenants_insert_isolation
ON user_tenants
FOR INSERT
WITH CHECK (
  app_is_platform_admin()
  OR tenant_id = app_current_tenant_id()
);

DROP POLICY IF EXISTS user_tenants_update_isolation ON user_tenants;
CREATE POLICY user_tenants_update_isolation
ON user_tenants
FOR UPDATE
USING (
  app_is_platform_admin()
  OR tenant_id = app_current_tenant_id()
)
WITH CHECK (
  app_is_platform_admin()
  OR tenant_id = app_current_tenant_id()
);

DROP POLICY IF EXISTS user_tenants_delete_isolation ON user_tenants;
CREATE POLICY user_tenants_delete_isolation
ON user_tenants
FOR DELETE
USING (
  app_is_platform_admin()
  OR tenant_id = app_current_tenant_id()
);

DROP POLICY IF EXISTS tenant_settings_isolation ON tenant_settings;
CREATE POLICY tenant_settings_isolation
ON tenant_settings
FOR ALL
USING (
  app_is_platform_admin()
  OR tenant_id = app_current_tenant_id()
)
WITH CHECK (
  app_is_platform_admin()
  OR tenant_id = app_current_tenant_id()
);

DROP POLICY IF EXISTS audit_logs_select_isolation ON audit_logs;
CREATE POLICY audit_logs_select_isolation
ON audit_logs
FOR SELECT
USING (
  app_is_platform_admin()
  OR (tenant_id IS NOT NULL AND tenant_id = app_current_tenant_id())
);

DROP POLICY IF EXISTS audit_logs_insert_isolation ON audit_logs;
CREATE POLICY audit_logs_insert_isolation
ON audit_logs
FOR INSERT
WITH CHECK (
  app_is_platform_admin()
  OR (tenant_id IS NOT NULL AND tenant_id = app_current_tenant_id())
);

DROP POLICY IF EXISTS audit_logs_update_isolation ON audit_logs;
CREATE POLICY audit_logs_update_isolation
ON audit_logs
FOR UPDATE
USING (app_is_platform_admin())
WITH CHECK (app_is_platform_admin());

DROP POLICY IF EXISTS audit_logs_delete_isolation ON audit_logs;
CREATE POLICY audit_logs_delete_isolation
ON audit_logs
FOR DELETE
USING (app_is_platform_admin());
