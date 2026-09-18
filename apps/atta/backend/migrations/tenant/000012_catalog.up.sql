CREATE TABLE catalog_items (
    id CHAR(26) PRIMARY KEY,
    name TEXT NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'unknown',
    status VARCHAR(32) NOT NULL DEFAULT 'provisional',
    creation_source VARCHAR(32) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT catalog_items_kind_check CHECK (kind IN ('goods', 'service', 'asset', 'unknown')),
    CONSTRAINT catalog_items_status_check CHECK (status IN ('provisional', 'confirmed')),
    CONSTRAINT catalog_items_creation_source_check CHECK (creation_source IN ('manual', 'invoice'))
);

CREATE INDEX ix_catalog_items_kind ON catalog_items (kind);
CREATE INDEX ix_catalog_items_status ON catalog_items (status);

CREATE TABLE catalog_item_aliases (
    id CHAR(26) PRIMARY KEY,
    item_id CHAR(26) NOT NULL REFERENCES catalog_items(id) ON DELETE CASCADE,
    scheme VARCHAR(64) NOT NULL,
    party_id CHAR(26) REFERENCES parties(id) ON DELETE SET NULL,
    value VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_catalog_item_aliases_scheme_party_value
    ON catalog_item_aliases (scheme, COALESCE(party_id, ''), value);

CREATE INDEX ix_catalog_item_aliases_item_id ON catalog_item_aliases (item_id);

CREATE TABLE catalog_match_memories (
    id CHAR(26) PRIMARY KEY,
    evidence_key VARCHAR(128) NOT NULL,
    party_id CHAR(26) REFERENCES parties(id) ON DELETE SET NULL,
    item_code VARCHAR(100),
    description_fingerprint VARCHAR(128),
    evidence_kind VARCHAR(64) NOT NULL,
    item_id CHAR(26) REFERENCES catalog_items(id) ON DELETE CASCADE,
    action VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT catalog_match_memories_action_check CHECK (action IN ('link', 'never_match'))
);

CREATE UNIQUE INDEX ux_catalog_match_memories_evidence_key
    ON catalog_match_memories (evidence_key);

CREATE INDEX ix_catalog_match_memories_party_code
    ON catalog_match_memories (party_id, item_code);
