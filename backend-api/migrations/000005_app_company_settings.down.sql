DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT id FROM permissions WHERE key IN ('app.settings.read','app.settings.update')
);

DELETE FROM permissions WHERE key IN ('app.settings.read','app.settings.update');

DROP TABLE IF EXISTS app_settings;
