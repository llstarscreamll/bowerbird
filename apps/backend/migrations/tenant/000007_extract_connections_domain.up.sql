-- Rename table to connection domain
ALTER TABLE connected_accounts RENAME TO connections;

-- Add new columns for connections
ALTER TABLE connections 
    ADD COLUMN owner_user_id CHAR(26),
    ADD COLUMN sharing_policy VARCHAR(50) DEFAULT 'tenant_all' NOT NULL,
    ADD COLUMN granted_scopes JSONB DEFAULT '[]'::jsonb NOT NULL;

-- Create inbox_sync_cursors table
CREATE TABLE inbox_sync_cursors (
    connection_id CHAR(26) PRIMARY KEY REFERENCES connections(id) ON DELETE CASCADE,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    status VARCHAR(30) DEFAULT 'idle' NOT NULL
);

-- Migrate data from connections to inbox_sync_cursors
INSERT INTO inbox_sync_cursors (connection_id, last_synced_at, last_error, status)
SELECT id, last_synced_at, last_error, 'idle'
FROM connections;

-- Drop obsolete columns from connections
ALTER TABLE connections
    DROP COLUMN last_synced_at,
    DROP COLUMN last_error;

-- Rename indices to reflect new table name
ALTER INDEX ux_connected_accounts_provider_email RENAME TO ux_connections_provider_email;
ALTER INDEX ix_connected_accounts_status RENAME TO ix_connections_status;
