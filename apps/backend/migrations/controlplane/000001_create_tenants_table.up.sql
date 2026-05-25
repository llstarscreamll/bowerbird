CREATE TABLE IF NOT EXISTS tenants (
    id CHAR(26) PRIMARY KEY,
    organization_name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    db_name VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast tenant resolution
CREATE INDEX idx_tenants_slug ON tenants(slug);
