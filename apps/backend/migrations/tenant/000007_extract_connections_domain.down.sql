-- Re-add dropped columns to connections
ALTER TABLE connections
    ADD COLUMN last_synced_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN last_error TEXT;

-- Migrate data back
UPDATE connections c
SET last_synced_at = isc.last_synced_at,
    last_error = isc.last_error
FROM inbox_sync_cursors isc
WHERE c.id = isc.connection_id;

-- Drop inbox_sync_cursors
DROP TABLE inbox_sync_cursors;

-- Drop new columns from connections
ALTER TABLE connections
    DROP COLUMN owner_user_id,
    DROP COLUMN sharing_policy,
    DROP COLUMN granted_scopes;

-- Rename indices back
ALTER INDEX ux_connections_provider_email RENAME TO ux_connected_accounts_provider_email;
ALTER INDEX ix_connections_status RENAME TO ix_connected_accounts_status;

-- Rename table back
ALTER TABLE connections RENAME TO connected_accounts;
