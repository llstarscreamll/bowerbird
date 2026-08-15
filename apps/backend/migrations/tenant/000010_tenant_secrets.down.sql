DROP TABLE IF EXISTS secret_audit_events;
DROP TABLE IF EXISTS secrets;

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('secrets:read', 'secrets:write', 'secrets:delete')
);

DELETE FROM permissions WHERE code IN ('secrets:read', 'secrets:write', 'secrets:delete');
