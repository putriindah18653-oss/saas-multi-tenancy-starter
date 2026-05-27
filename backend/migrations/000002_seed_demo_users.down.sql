DELETE FROM user_tenants
WHERE user_id = '33333333-3333-3333-3333-333333333333'
  AND tenant_id = '22222222-2222-2222-2222-222222222222';

DELETE FROM users
WHERE email IN ('owner@app.local', 'admin@tenant.local');

DELETE FROM tenants
WHERE slug = 'tenant-alpha-demo';
