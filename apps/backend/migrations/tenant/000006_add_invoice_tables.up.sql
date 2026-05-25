CREATE TABLE invoice_headers (
    id CHAR(26) PRIMARY KEY,
    source_message_id CHAR(26),
    cufe VARCHAR(128) NOT NULL,
    invoice_number VARCHAR(100),
    issuer_name VARCHAR(255),
    issuer_tax_id VARCHAR(100),
    receiver_name VARCHAR(255),
    receiver_tax_id VARCHAR(100),
    currency_code VARCHAR(10),
    issue_date TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    payment_code VARCHAR(50),
    subtotal NUMERIC(18,2),
    tax_total NUMERIC(18,2),
    grand_total NUMERIC(18,2),
    document_ref_s3_key TEXT,
    extraction_source VARCHAR(20) NOT NULL,
    raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_invoice_headers_cufe
    ON invoice_headers(cufe);

CREATE INDEX ix_invoice_headers_source_message_id
    ON invoice_headers(source_message_id);

CREATE INDEX ix_invoice_headers_invoice_number
    ON invoice_headers(invoice_number);

CREATE TABLE invoice_lines (
    id CHAR(26) PRIMARY KEY,
    invoice_header_id CHAR(26) NOT NULL REFERENCES invoice_headers(id) ON DELETE CASCADE,
    line_number INTEGER NOT NULL,
    item_code VARCHAR(100),
    description TEXT,
    quantity NUMERIC(18,6),
    unit_price NUMERIC(18,6),
    line_tax_total NUMERIC(18,2),
    line_total NUMERIC(18,2),
    raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_invoice_lines_header_line_number
    ON invoice_lines(invoice_header_id, line_number);

CREATE INDEX ix_invoice_lines_invoice_header_id
    ON invoice_lines(invoice_header_id);
