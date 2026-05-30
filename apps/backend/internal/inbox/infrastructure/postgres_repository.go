package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
)

type PostgresRepository struct {
	registry *database.Registry
}

func NewPostgresRepository(registry *database.Registry) *PostgresRepository {
	return &PostgresRepository{registry: registry}
}

func (r *PostgresRepository) GetSyncCursor(ctx context.Context, connectionID string) (*domain.InboxSyncCursor, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		SELECT connection_id, last_synced_at, last_error, status
		FROM inbox_sync_cursors
		WHERE connection_id = $1
	`
	var cursor domain.InboxSyncCursor
	err = pool.QueryRow(ctx, query, connectionID).Scan(
		&cursor.ConnectionID,
		&cursor.LastSyncedAt,
		&cursor.LastError,
		&cursor.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found is fine, we can create one
		}
		return nil, fmt.Errorf("failed to get sync cursor: %w", err)
	}

	return &cursor, nil
}

func (r *PostgresRepository) UpsertSyncCursor(ctx context.Context, cursor *domain.InboxSyncCursor) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		INSERT INTO inbox_sync_cursors (connection_id, last_synced_at, last_error, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (connection_id) DO UPDATE SET
			last_synced_at = EXCLUDED.last_synced_at,
			last_error = EXCLUDED.last_error,
			status = EXCLUDED.status
	`
	_, err = pool.Exec(ctx, query, cursor.ConnectionID, cursor.LastSyncedAt, cursor.LastError, cursor.Status)
	if err != nil {
		return fmt.Errorf("failed to upsert sync cursor: %w", err)
	}

	return nil
}

func (r *PostgresRepository) UpsertEmailMessage(ctx context.Context, message *domain.EmailMessage) (bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		INSERT INTO email_messages (
			id,
			account_id,
			provider_message_id,
			provider_thread_id,
			subject,
			sender_email,
			received_at,
			sync_status,
			raw_data,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (account_id, provider_message_id)
		DO UPDATE SET
			provider_thread_id = EXCLUDED.provider_thread_id,
			subject = EXCLUDED.subject,
			sender_email = EXCLUDED.sender_email,
			received_at = EXCLUDED.received_at,
			sync_status = EXCLUDED.sync_status,
			raw_data = EXCLUDED.raw_data,
			updated_at = EXCLUDED.updated_at
		RETURNING id, (xmax = 0) AS inserted
	`

	var inserted bool
	err = pool.QueryRow(
		ctx,
		query,
		message.ID,
		message.AccountID,
		message.ProviderMessageID,
		message.ProviderThreadID,
		message.Subject,
		message.SenderEmail,
		message.ReceivedAt,
		message.SyncStatus,
		defaultRawData(message.RawData),
		message.CreatedAt,
		message.UpdatedAt,
	).Scan(&message.ID, &inserted)
	if err != nil {
		return false, fmt.Errorf("failed to upsert email message: %w", err)
	}

	return inserted, nil
}

func (r *PostgresRepository) GetEmailMessageByProviderID(ctx context.Context, accountID, providerMessageID string) (*domain.EmailMessage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		SELECT
			id,
			account_id,
			provider_message_id,
			provider_thread_id,
			subject,
			sender_email,
			received_at,
			sync_status,
			raw_data,
			created_at,
			updated_at
		FROM email_messages
		WHERE account_id = $1 AND provider_message_id = $2
	`

	var message domain.EmailMessage
	err = pool.QueryRow(ctx, query, accountID, providerMessageID).Scan(
		&message.ID,
		&message.AccountID,
		&message.ProviderMessageID,
		&message.ProviderThreadID,
		&message.Subject,
		&message.SenderEmail,
		&message.ReceivedAt,
		&message.SyncStatus,
		&message.RawData,
		&message.CreatedAt,
		&message.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEmailMessageNotFound
		}
		return nil, fmt.Errorf("failed to get email message by provider id: %w", err)
	}

	return &message, nil
}

func (r *PostgresRepository) UpsertEmailAttachment(ctx context.Context, attachment *domain.EmailAttachment) (bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		INSERT INTO email_attachments (
			id,
			message_id,
			filename,
			mime_type,
			size_bytes,
			sha256,
			s3_key,
			raw_data,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (message_id, sha256)
		DO UPDATE SET
			filename = EXCLUDED.filename,
			mime_type = EXCLUDED.mime_type,
			size_bytes = EXCLUDED.size_bytes,
			s3_key = EXCLUDED.s3_key,
			raw_data = EXCLUDED.raw_data,
			updated_at = EXCLUDED.updated_at
		RETURNING id, (xmax = 0) AS inserted
	`

	var inserted bool
	err = pool.QueryRow(
		ctx,
		query,
		attachment.ID,
		attachment.MessageID,
		attachment.Filename,
		attachment.MimeType,
		attachment.SizeBytes,
		attachment.SHA256,
		attachment.S3Key,
		defaultRawData(attachment.RawData),
		attachment.CreatedAt,
		attachment.UpdatedAt,
	).Scan(&attachment.ID, &inserted)
	if err != nil {
		return false, fmt.Errorf("failed to upsert email attachment: %w", err)
	}

	return inserted, nil
}

func (r *PostgresRepository) ListEmailAttachmentsByMessageID(ctx context.Context, messageID string) ([]domain.EmailAttachment, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		SELECT
			id,
			message_id,
			filename,
			mime_type,
			size_bytes,
			sha256,
			s3_key,
			raw_data,
			created_at,
			updated_at
		FROM email_attachments
		WHERE message_id = $1
		ORDER BY created_at ASC
	`

	rows, err := pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list email attachments: %w", err)
	}
	defer rows.Close()

	attachments := make([]domain.EmailAttachment, 0)
	for rows.Next() {
		var attachment domain.EmailAttachment
		if err := rows.Scan(
			&attachment.ID,
			&attachment.MessageID,
			&attachment.Filename,
			&attachment.MimeType,
			&attachment.SizeBytes,
			&attachment.SHA256,
			&attachment.S3Key,
			&attachment.RawData,
			&attachment.CreatedAt,
			&attachment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan email attachment: %w", err)
		}

		attachments = append(attachments, attachment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating email attachments: %w", err)
	}

	return attachments, nil
}

func (r *PostgresRepository) ListUnifiedMessages(ctx context.Context) ([]domain.UnifiedMessage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		SELECT
			m.id,
			c.provider,
			c.id AS account_id,
			c.email_address AS account_email,
			COALESCE(m.subject, '(Sin asunto)') AS subject,
			COALESCE(m.sender_email, 'Desconocido') AS sender,
			COALESCE(NULLIF(m.raw_data->>'snippet', ''), m.raw_data->>'Snippet', '') AS snippet,
			COALESCE(m.received_at, m.created_at) AS received_at,
			COALESCE(m.sync_status, 'new') AS processing_status,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%.xml') AS has_xml,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%.pdf') AS has_pdf
		FROM email_messages m
		JOIN connections c ON m.account_id = c.id
		ORDER BY received_at DESC NULLS LAST
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list unified messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.UnifiedMessage, 0)
	for rows.Next() {
		var msg domain.UnifiedMessage
		if err := rows.Scan(
			&msg.ID,
			&msg.Provider,
			&msg.AccountID,
			&msg.AccountEmail,
			&msg.Subject,
			&msg.Sender,
			&msg.Snippet,
			&msg.ReceivedAt,
			&msg.ProcessingStatus,
			&msg.HasXML,
			&msg.HasPDF,
		); err != nil {
			return nil, fmt.Errorf("failed to scan unified message: %w", err)
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while iterating unified messages: %w", err)
	}

	return messages, nil
}

func (r *PostgresRepository) ListMessagesByAccount(ctx context.Context, accountID string, limit, offset int) ([]domain.UnifiedMessage, error) {
	// Not implemented fully yet but required by interface
	return nil, errors.New("ListMessagesByAccount not implemented")
}

func (r *PostgresRepository) GetMessageByID(ctx context.Context, messageID string) (*domain.UnifiedMessage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		SELECT
			m.id,
			c.provider,
			c.id AS account_id,
			c.email_address AS account_email,
			COALESCE(m.subject, '(Sin asunto)') AS subject,
			COALESCE(m.sender_email, 'Desconocido') AS sender,
			COALESCE(NULLIF(m.raw_data->>'snippet', ''), m.raw_data->>'Snippet', '') AS snippet,
			COALESCE(
				NULLIF(m.raw_data->>'plain_text_body', ''),
				NULLIF(m.raw_data->>'PlainTextBody', ''),
				NULLIF(m.raw_data->>'snippet', ''),
				m.raw_data->>'Snippet',
				''
			) AS body_text,
			m.raw_data,
			COALESCE(m.received_at, m.created_at) AS received_at,
			COALESCE(m.sync_status, 'new') AS processing_status,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%.xml') AS has_xml,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%.pdf') AS has_pdf
		FROM email_messages m
		JOIN connections c ON m.account_id = c.id
		WHERE m.id = $1
	`

	var msg domain.UnifiedMessage
	err = pool.QueryRow(ctx, query, messageID).Scan(
		&msg.ID,
		&msg.Provider,
		&msg.AccountID,
		&msg.AccountEmail,
		&msg.Subject,
		&msg.Sender,
		&msg.Snippet,
		&msg.BodyText,
		&msg.RawData,
		&msg.ReceivedAt,
		&msg.ProcessingStatus,
		&msg.HasXML,
		&msg.HasPDF,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEmailMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message by id: %w", err)
	}

	return &msg, nil
}

func (r *PostgresRepository) GetMessageAttachments(ctx context.Context, messageID string) ([]domain.EmailAttachment, error) {
	return r.ListEmailAttachmentsByMessageID(ctx, messageID)
}

func defaultRawData(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}

	return raw
}
