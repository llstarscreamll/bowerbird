ALTER TABLE catalog_items
    DROP CONSTRAINT IF EXISTS catalog_items_creation_source_check;

ALTER TABLE catalog_items
    ADD CONSTRAINT catalog_items_creation_source_check
        CHECK (creation_source IN ('manual', 'invoice', 'import'));

CREATE TABLE catalog_imports (
    id CHAR(26) PRIMARY KEY,
    file_key TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL,
    total_rows BIGINT NOT NULL DEFAULT 0,
    created_count BIGINT NOT NULL DEFAULT 0,
    updated_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    byte_offset BIGINT NOT NULL DEFAULT 0,
    last_file_row INTEGER NOT NULL DEFAULT 0,
    failure_reason TEXT,
    requested_by_user_id TEXT NOT NULL,
    requested_by_email TEXT NOT NULL,
    requested_by_name TEXT NOT NULL,
    cancelled_by_user_id TEXT,
    cancelled_by_email TEXT,
    cancelled_by_name TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT catalog_imports_status_check CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'cancelled'))
);

CREATE UNIQUE INDEX ux_catalog_imports_one_active
    ON catalog_imports ((true))
    WHERE status IN ('queued', 'processing');

CREATE INDEX ix_catalog_imports_created_at_id
    ON catalog_imports (created_at DESC, id DESC);

CREATE INDEX ix_catalog_imports_created_at
    ON catalog_imports (created_at);

CREATE TABLE catalog_import_errors (
    id CHAR(26) PRIMARY KEY,
    import_id CHAR(26) NOT NULL REFERENCES catalog_imports(id) ON DELETE CASCADE,
    file_row INTEGER NOT NULL,
    column_name VARCHAR(32),
    internal_code TEXT,
    name TEXT,
    kind TEXT,
    code VARCHAR(64) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT catalog_import_errors_file_row_check CHECK (file_row >= 1)
);

CREATE UNIQUE INDEX ux_catalog_import_errors_import_row_column
    ON catalog_import_errors (import_id, file_row, COALESCE(column_name, ''));

CREATE INDEX ix_catalog_import_errors_import_row_id
    ON catalog_import_errors (import_id, file_row, id);

CREATE INDEX ix_catalog_items_name_id
    ON catalog_items (name, id);
