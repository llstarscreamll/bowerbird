CREATE TABLE connected_accounts (
    id CHAR(26) PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    email_address VARCHAR(255) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active',
    encrypted_credentials BYTEA NOT NULL,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_connected_accounts_provider_email
    ON connected_accounts(provider, email_address);

CREATE INDEX ix_connected_accounts_status
    ON connected_accounts(status);

CREATE TABLE email_messages (
    id CHAR(26) PRIMARY KEY,
    account_id CHAR(26) NOT NULL REFERENCES connected_accounts(id) ON DELETE CASCADE,
    provider_message_id VARCHAR(255) NOT NULL,
    provider_thread_id VARCHAR(255),
    subject TEXT,
    sender_email VARCHAR(255),
    received_at TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(30) NOT NULL DEFAULT 'synced',
    raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_email_messages_account_provider_message
    ON email_messages(account_id, provider_message_id);

CREATE INDEX ix_email_messages_account_id
    ON email_messages(account_id);

CREATE INDEX ix_email_messages_received_at
    ON email_messages(received_at);

CREATE TABLE email_attachments (
    id CHAR(26) PRIMARY KEY,
    message_id CHAR(26) NOT NULL REFERENCES email_messages(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    mime_type VARCHAR(255),
    size_bytes BIGINT,
    sha256 CHAR(64) NOT NULL,
    s3_key TEXT NOT NULL,
    raw_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_email_attachments_message_sha256
    ON email_attachments(message_id, sha256);

CREATE INDEX ix_email_attachments_message_id
    ON email_attachments(message_id);
