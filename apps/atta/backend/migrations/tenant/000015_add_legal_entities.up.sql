CREATE TABLE legal_entities (
    id CHAR(26) PRIMARY KEY,
    tax_id VARCHAR(32) NOT NULL,
    scheme_id VARCHAR(8) NOT NULL,
    legal_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_legal_entities_tax_id
    ON legal_entities (tax_id);
