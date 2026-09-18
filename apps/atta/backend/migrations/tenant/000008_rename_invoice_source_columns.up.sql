ALTER TABLE invoice_headers
    RENAME COLUMN source_message_id TO source_id;

ALTER TABLE invoice_headers
    ALTER COLUMN source_id TYPE VARCHAR(255);

ALTER TABLE invoice_headers
    ADD COLUMN source_name VARCHAR(50);

UPDATE invoice_headers
SET source_name = 'inbox-message'
WHERE source_name IS NULL;

ALTER TABLE invoice_headers
    ALTER COLUMN source_name SET NOT NULL;

DROP INDEX IF EXISTS ix_invoice_headers_source_message_id;

CREATE INDEX ix_invoice_headers_source_name_source_id
    ON invoice_headers(source_name, source_id);
