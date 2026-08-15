ALTER TABLE users ADD COLUMN IF NOT EXISTS platform_role VARCHAR(50);

CREATE TABLE IF NOT EXISTS tenant_entitlements (
    id CHAR(26) PRIMARY KEY,
    tenant_id CHAR(26) NOT NULL REFERENCES tenants(id),
    feature_key VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    source VARCHAR(50) NOT NULL DEFAULT 'manual',
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ends_at TIMESTAMP WITH TIME ZONE,
    created_by CHAR(26),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, feature_key)
);

CREATE INDEX IF NOT EXISTS idx_tenant_entitlements_tenant ON tenant_entitlements (tenant_id);

INSERT INTO tenant_entitlements (id, tenant_id, feature_key, status, source, starts_at, created_at, updated_at)
SELECT
    substr(md5(t.id || ':' || f.feature_key), 1, 26),
    t.id,
    f.feature_key,
    'active',
    'manual',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM tenants t
CROSS JOIN (VALUES
    ('invoicing.workspace'),
    ('invoicing.capture_from_email'),
    ('mail.inbox'),
    ('mail.organize')
) AS f(feature_key)
ON CONFLICT (tenant_id, feature_key) DO NOTHING;
