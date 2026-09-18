CREATE TABLE parties (
    id CHAR(26) PRIMARY KEY,
    tax_id VARCHAR(100),
    name VARCHAR(255) NOT NULL,
    roles TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'provisional',
    creation_source VARCHAR(32) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT parties_creation_source_check CHECK (creation_source IN ('manual', 'invoice'))
);

CREATE UNIQUE INDEX ux_parties_tax_id
    ON parties (tax_id)
    WHERE tax_id IS NOT NULL AND btrim(tax_id) <> '';

CREATE INDEX ix_parties_name ON parties (name);

CREATE INDEX ix_parties_roles ON parties USING GIN (roles);
