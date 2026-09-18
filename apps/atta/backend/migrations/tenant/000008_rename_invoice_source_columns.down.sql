DROP INDEX IF EXISTS ix_invoice_headers_source_name_source_id;

ALTER TABLE invoice_headers
    DROP COLUMN source_name;

ALTER TABLE invoice_headers
    RENAME COLUMN source_id TO source_message_id;

ALTER TABLE invoice_headers
    ALTER COLUMN source_message_id TYPE CHAR(26);

CREATE INDEX ix_invoice_headers_source_message_id
    ON invoice_headers(source_message_id);
