ALTER TABLE parties
    ADD COLUMN scheme_id VARCHAR(8),
    ADD COLUMN taxpayer_kind VARCHAR(8),
    ADD COLUMN tax_level_codes TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE parties
    ADD CONSTRAINT parties_scheme_id_check CHECK (scheme_id IS NULL OR scheme_id IN ('31', '13')),
    ADD CONSTRAINT parties_taxpayer_kind_check CHECK (taxpayer_kind IS NULL OR taxpayer_kind IN ('1', '2'));

CREATE TABLE party_emails (
    id CHAR(26) PRIMARY KEY,
    party_id CHAR(26) NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
    value VARCHAR(255) NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'general',
    source VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT party_emails_kind_check CHECK (kind IN ('general', 'tax_mailbox')),
    CONSTRAINT party_emails_source_check CHECK (source IN ('invoice', 'manual'))
);

CREATE UNIQUE INDEX ux_party_emails_value
    ON party_emails (party_id, lower(btrim(value)));

CREATE INDEX ix_party_emails_party_id ON party_emails (party_id);

CREATE TABLE party_phones (
    id CHAR(26) PRIMARY KEY,
    party_id CHAR(26) NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
    value VARCHAR(64) NOT NULL,
    normalized VARCHAR(64) NOT NULL,
    source VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT party_phones_source_check CHECK (source IN ('invoice', 'manual'))
);

CREATE UNIQUE INDEX ux_party_phones_normalized
    ON party_phones (party_id, normalized);

CREATE INDEX ix_party_phones_party_id ON party_phones (party_id);

CREATE TABLE party_addresses (
    id CHAR(26) PRIMARY KEY,
    party_id CHAR(26) NOT NULL REFERENCES parties(id) ON DELETE CASCADE,
    line VARCHAR(255) NOT NULL DEFAULT '',
    city VARCHAR(128) NOT NULL DEFAULT '',
    department VARCHAR(128) NOT NULL DEFAULT '',
    postal_zone VARCHAR(32) NOT NULL DEFAULT '',
    country_code VARCHAR(8) NOT NULL DEFAULT '',
    kind VARCHAR(32) NOT NULL DEFAULT 'other',
    fingerprint VARCHAR(512) NOT NULL,
    source VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT party_addresses_kind_check CHECK (kind IN ('physical', 'registration', 'other')),
    CONSTRAINT party_addresses_source_check CHECK (source IN ('invoice', 'manual'))
);

CREATE UNIQUE INDEX ux_party_addresses_fingerprint
    ON party_addresses (party_id, fingerprint);

CREATE INDEX ix_party_addresses_party_id ON party_addresses (party_id);
