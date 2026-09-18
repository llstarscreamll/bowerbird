-- Permissions for tenant secrets vault
INSERT INTO permissions (id, code, description) VALUES
('01JW58TAT9M0N4R8M1P3Q6R9Y5', 'secrets:read', 'Ver metadatos de secretos de la organización'),
('01JW58TAT9M0N4R8M1P3Q6R9Y6', 'secrets:write', 'Crear y rotar secretos de la organización'),
('01JW58TAT9M0N4R8M1P3Q6R9Y7', 'secrets:delete', 'Eliminar secretos de la organización')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.code IN ('secrets:read', 'secrets:write', 'secrets:delete')
ON CONFLICT DO NOTHING;

-- Existing tenant users get admin so secrets ACL is usable after migration
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u
CROSS JOIN roles r
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

CREATE TABLE secrets (
    id CHAR(26) PRIMARY KEY,
    purpose VARCHAR(100) NOT NULL,
    kind VARCHAR(50) NOT NULL,
    label VARCHAR(255) NOT NULL,
    description VARCHAR(500),
    ciphertext BYTEA NOT NULL,
    version INT NOT NULL DEFAULT 1,
    key_id VARCHAR(64) NOT NULL DEFAULT 'local-aes-v1',
    last_used_at TIMESTAMPTZ,
    created_by CHAR(26) NOT NULL,
    updated_by CHAR(26) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT secrets_purpose_label_unique UNIQUE (purpose, label)
);

CREATE INDEX secrets_purpose_idx ON secrets (purpose);
CREATE INDEX secrets_purpose_last_used_idx ON secrets (purpose, last_used_at DESC NULLS LAST, created_at ASC);

CREATE TABLE secret_audit_events (
    id CHAR(26) PRIMARY KEY,
    secret_id CHAR(26),
    purpose VARCHAR(100) NOT NULL,
    action VARCHAR(32) NOT NULL,
    actor_user_id CHAR(26) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX secret_audit_events_secret_id_idx ON secret_audit_events (secret_id);
CREATE INDEX secret_audit_events_created_at_idx ON secret_audit_events (created_at DESC);
