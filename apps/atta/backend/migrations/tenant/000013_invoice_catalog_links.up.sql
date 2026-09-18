ALTER TABLE invoice_headers
    ADD COLUMN issuer_party_id CHAR(26) REFERENCES parties(id) ON DELETE SET NULL,
    ADD COLUMN linking_status VARCHAR(32) NOT NULL DEFAULT 'pending';

CREATE INDEX ix_invoice_headers_issuer_party_id
    ON invoice_headers (issuer_party_id);

ALTER TABLE invoice_lines
    ADD COLUMN item_id CHAR(26) REFERENCES catalog_items(id) ON DELETE SET NULL,
    ADD COLUMN link_status VARCHAR(32) NOT NULL DEFAULT 'unmatched',
    ADD COLUMN link_method VARCHAR(32),
    ADD COLUMN link_locked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN suggestions JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX ix_invoice_lines_item_id ON invoice_lines (item_id);
CREATE INDEX ix_invoice_lines_link_status ON invoice_lines (link_status);
