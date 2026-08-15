DELETE FROM tenant_entitlements;
DROP TABLE IF EXISTS tenant_entitlements;
ALTER TABLE users DROP COLUMN IF EXISTS platform_role;
