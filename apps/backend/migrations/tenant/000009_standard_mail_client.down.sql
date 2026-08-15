DROP INDEX IF EXISTS ix_email_messages_account_thread;
DROP INDEX IF EXISTS ix_email_messages_account_folder_received;

ALTER TABLE inbox_sync_cursors
    DROP COLUMN IF EXISTS history_id;

ALTER TABLE email_messages
    DROP COLUMN IF EXISTS is_draft,
    DROP COLUMN IF EXISTS is_starred,
    DROP COLUMN IF EXISTS is_read,
    DROP COLUMN IF EXISTS folder,
    DROP COLUMN IF EXISTS bcc_emails,
    DROP COLUMN IF EXISTS cc_emails,
    DROP COLUMN IF EXISTS to_emails,
    DROP COLUMN IF EXISTS snippet;
