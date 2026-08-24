ALTER TABLE invoice_lines
    DROP COLUMN IF EXISTS suggestions,
    DROP COLUMN IF EXISTS link_locked,
    DROP COLUMN IF EXISTS link_method,
    DROP COLUMN IF EXISTS link_status,
    DROP COLUMN IF EXISTS item_id;

DROP INDEX IF EXISTS ix_invoice_headers_issuer_party_id;

ALTER TABLE invoice_headers
    DROP COLUMN IF EXISTS linking_status,
    DROP COLUMN IF EXISTS issuer_party_id;
