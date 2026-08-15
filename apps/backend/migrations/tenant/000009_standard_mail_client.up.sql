-- Mail client model: folders, flags, recipients, history cursor

ALTER TABLE email_messages
    ADD COLUMN snippet TEXT,
    ADD COLUMN to_emails TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN cc_emails TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN bcc_emails TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN folder VARCHAR(50) NOT NULL DEFAULT 'inbox',
    ADD COLUMN is_read BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_draft BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX ix_email_messages_account_folder_received
    ON email_messages (account_id, folder, received_at DESC);

CREATE INDEX ix_email_messages_account_thread
    ON email_messages (account_id, provider_thread_id);

ALTER TABLE inbox_sync_cursors
    ADD COLUMN history_id VARCHAR(255);
