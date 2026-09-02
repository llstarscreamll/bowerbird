package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	inboxPorts "github.com/bowerbird/internal/inbox/application/ports"
	"github.com/bowerbird/internal/inbox/domain"
	"github.com/bowerbird/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	registry *database.Registry
}

func NewPostgresRepository(registry *database.Registry) *PostgresRepository {
	return &PostgresRepository{registry: registry}
}

func (r *PostgresRepository) GetSyncCursor(ctx context.Context, connectionID string) (*domain.SyncCursor, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
			SELECT connection_id, last_synced_at, last_error, status, history_id
			FROM inbox_sync_cursors
			WHERE connection_id = $1
		`
	var snapshot domain.SyncCursorSnapshot
	var status string
	err = pool.QueryRow(ctx, query, connectionID).Scan(
		&snapshot.ConnectionID,
		&snapshot.LastSyncedAt,
		&snapshot.LastError,
		&status,
		&snapshot.HistoryID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found is fine, we can create one
		}
		return nil, fmt.Errorf("failed to get sync cursor: %w", err)
	}

	snapshot.Status = domain.SyncCursorStatus(status)
	if !snapshot.Status.IsValid() {
		snapshot.Status = domain.SyncCursorStatusIdle
	}

	return domain.RehydrateSyncCursor(snapshot), nil
}

func (r *PostgresRepository) UpsertSyncCursor(ctx context.Context, cursor *domain.SyncCursor) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	snapshot := cursor.Snapshot()
	query := `
			INSERT INTO inbox_sync_cursors (connection_id, last_synced_at, last_error, status, history_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (connection_id) DO UPDATE SET
				last_synced_at = EXCLUDED.last_synced_at,
				last_error = EXCLUDED.last_error,
				status = EXCLUDED.status,
				history_id = EXCLUDED.history_id
		`
	_, err = pool.Exec(ctx, query, snapshot.ConnectionID, snapshot.LastSyncedAt, snapshot.LastError, snapshot.Status.String(), snapshot.HistoryID)
	if err != nil {
		return fmt.Errorf("failed to upsert sync cursor: %w", err)
	}

	return nil
}

func (r *PostgresRepository) UpsertInboxMessage(ctx context.Context, message *domain.InboxMessage) (bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get tenant db pool: %w", err)
	}
	q := database.QuerierFromContext(ctx, pool)
	snapshot := message.Snapshot()
	query := `
			INSERT INTO email_messages (
				id,
				account_id,
				provider_message_id,
				provider_thread_id,
				subject,
				sender_email,
				snippet,
				to_emails,
				cc_emails,
				bcc_emails,
				folder,
				is_read,
				is_starred,
				is_draft,
				received_at,
				sync_status,
				raw_data,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
			ON CONFLICT (account_id, provider_message_id)
			DO UPDATE SET
				provider_thread_id = EXCLUDED.provider_thread_id,
				subject = EXCLUDED.subject,
				sender_email = EXCLUDED.sender_email,
				snippet = EXCLUDED.snippet,
				to_emails = EXCLUDED.to_emails,
				cc_emails = EXCLUDED.cc_emails,
				bcc_emails = EXCLUDED.bcc_emails,
				folder = EXCLUDED.folder,
				is_read = EXCLUDED.is_read,
				is_starred = EXCLUDED.is_starred,
				is_draft = EXCLUDED.is_draft,
				received_at = EXCLUDED.received_at,
				sync_status = EXCLUDED.sync_status,
				raw_data = EXCLUDED.raw_data,
				updated_at = EXCLUDED.updated_at
			RETURNING id, (xmax = 0) AS inserted
		`

	var inserted bool
	var returnedID string
	err = q.QueryRow(
		ctx,
		query,
		snapshot.ID,
		snapshot.ConnectionID,
		snapshot.ProviderMessageID,
		snapshot.ProviderThreadID,
		snapshot.Subject,
		snapshot.SenderEmail,
		snapshot.Snippet,
		nullableTextArray(snapshot.ToEmails),
		nullableTextArray(snapshot.CcEmails),
		nullableTextArray(snapshot.BccEmails),
		snapshot.Folder.String(),
		snapshot.IsRead,
		snapshot.IsStarred,
		snapshot.IsDraft,
		snapshot.ReceivedAt,
		string(snapshot.SyncStatus),
		defaultRawData(snapshot.RawData),
		snapshot.CreatedAt,
		snapshot.UpdatedAt,
	).Scan(&returnedID, &inserted)
	if err != nil {
		return false, fmt.Errorf("failed to upsert inbox message: %w", err)
	}
	message.ConfirmPersisted(returnedID)

	return inserted, nil
}

func (r *PostgresRepository) UpsertMessageAttachment(ctx context.Context, attachment *domain.MessageAttachment) (bool, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get tenant db pool: %w", err)
	}
	q := database.QuerierFromContext(ctx, pool)

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
	err = q.QueryRow(
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
		return false, fmt.Errorf("failed to upsert message attachment: %w", err)
	}

	return inserted, nil
}

func (r *PostgresRepository) ListMessageViews(ctx context.Context, filter inboxPorts.ListMessagesFilter) (inboxPorts.MessageListPage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return inboxPorts.MessageListPage{}, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args := []any{filter.ViewerUserID}
	conditions := []string{"(c.sharing_policy = 'tenant_all' OR c.owner_user_id = $1)"}
	argPos := 2

	folder := strings.TrimSpace(filter.Folder)
	if folder == "" {
		folder = string(domain.MailFolderInbox)
	}
	if folder == string(domain.MailFolderStarred) {
		conditions = append(conditions, "m.is_starred = TRUE AND m.folder <> 'trash'")
	} else if domain.MailFolder(folder).IsValid() {
		conditions = append(conditions, fmt.Sprintf("m.folder = $%d", argPos))
		args = append(args, folder)
		argPos++
	}

	if filter.AccountID != "" && filter.AccountID != "all" {
		conditions = append(conditions, fmt.Sprintf("c.id = $%d", argPos))
		args = append(args, filter.AccountID)
		argPos++
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		conditions = append(conditions, fmt.Sprintf("(COALESCE(m.subject,'') ILIKE $%d OR COALESCE(m.sender_email,'') ILIKE $%d OR COALESCE(m.snippet,'') ILIKE $%d)", argPos, argPos, argPos))
		args = append(args, "%"+q+"%")
		argPos++
	}

	if filter.OnlyInvoices {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM email_attachments a
			WHERE a.message_id = m.id AND (a.filename ILIKE '%.xml' OR a.filename ILIKE '%.pdf')
		)`)
	}

	where := strings.Join(conditions, " AND ")
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM email_messages m
		JOIN connections c ON m.account_id = c.id
		WHERE %s
	`, where)

	var total int
	if err := pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return inboxPorts.MessageListPage{}, fmt.Errorf("failed to count message views: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT
			m.id,
			c.provider,
			c.id AS account_id,
			c.email_address AS account_email,
			COALESCE(m.subject, '(Sin asunto)') AS subject,
			COALESCE(m.sender_email, 'Desconocido') AS sender,
			COALESCE(m.snippet, NULLIF(m.raw_data->>'snippet', ''), m.raw_data->>'Snippet', '') AS snippet,
			COALESCE(m.to_emails, '{}') AS to_emails,
			COALESCE(m.provider_thread_id, '') AS thread_id,
			m.folder,
			m.is_read,
			m.is_starred,
			m.is_draft,
			COALESCE(m.received_at, m.created_at) AS received_at,
			CASE
				WHEN EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND (a.filename ILIKE '%%.xml' OR a.filename ILIKE '%%.pdf')) THEN 'new'
				ELSE 'skipped'
			END AS processing_status,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%%.xml') AS has_xml,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%%.pdf') AS has_pdf
		FROM email_messages m
		JOIN connections c ON m.account_id = c.id
		WHERE %s
		ORDER BY received_at DESC NULLS LAST
		LIMIT $%d OFFSET $%d
	`, where, argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := pool.Query(ctx, listQuery, args...)
	if err != nil {
		return inboxPorts.MessageListPage{}, fmt.Errorf("failed to list message views: %w", err)
	}
	defer rows.Close()

	messages := make([]inboxPorts.MessageListView, 0)
	for rows.Next() {
		var msg inboxPorts.MessageListView
		if err := rows.Scan(
			&msg.ID,
			&msg.Provider,
			&msg.AccountID,
			&msg.AccountEmail,
			&msg.Subject,
			&msg.Sender,
			&msg.Snippet,
			&msg.ToEmails,
			&msg.ThreadID,
			&msg.Folder,
			&msg.IsRead,
			&msg.IsStarred,
			&msg.IsDraft,
			&msg.ReceivedAt,
			&msg.ProcessingStatus,
			&msg.HasXML,
			&msg.HasPDF,
		); err != nil {
			return inboxPorts.MessageListPage{}, fmt.Errorf("failed to scan message list view: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return inboxPorts.MessageListPage{}, fmt.Errorf("failed while iterating message list views: %w", err)
	}

	return inboxPorts.MessageListPage{Items: messages, Total: total, Limit: limit, Offset: offset}, nil
}

func (r *PostgresRepository) GetMessageViewByID(ctx context.Context, messageID, viewerUserID string) (*inboxPorts.MessageDetailView, error) {
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
			COALESCE(m.snippet, NULLIF(m.raw_data->>'snippet', ''), m.raw_data->>'Snippet', '') AS snippet,
			COALESCE(m.to_emails, '{}'),
			COALESCE(m.cc_emails, '{}'),
			COALESCE(m.bcc_emails, '{}'),
			COALESCE(m.provider_thread_id, ''),
			m.folder,
			m.is_read,
			m.is_starred,
			m.is_draft,
			COALESCE(
				NULLIF(m.raw_data->>'plain_text_body', ''),
				NULLIF(m.raw_data->>'PlainTextBody', ''),
				COALESCE(m.snippet, ''),
				''
			) AS body_text,
			m.raw_data,
			COALESCE(m.received_at, m.created_at) AS received_at,
			CASE
				WHEN EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND (a.filename ILIKE '%.xml' OR a.filename ILIKE '%.pdf')) THEN 'new'
				ELSE 'skipped'
			END AS processing_status,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%.xml') AS has_xml,
			EXISTS(SELECT 1 FROM email_attachments a WHERE a.message_id = m.id AND a.filename ILIKE '%.pdf') AS has_pdf
		FROM email_messages m
		JOIN connections c ON m.account_id = c.id
		WHERE m.id = $1 AND (c.sharing_policy = 'tenant_all' OR c.owner_user_id = $2)
	`

	var msg inboxPorts.MessageDetailView
	err = pool.QueryRow(ctx, query, messageID, viewerUserID).Scan(
		&msg.ID,
		&msg.Provider,
		&msg.AccountID,
		&msg.AccountEmail,
		&msg.Subject,
		&msg.Sender,
		&msg.Snippet,
		&msg.ToEmails,
		&msg.CcEmails,
		&msg.BccEmails,
		&msg.ThreadID,
		&msg.Folder,
		&msg.IsRead,
		&msg.IsStarred,
		&msg.IsDraft,
		&msg.BodyText,
		&msg.RawData,
		&msg.ReceivedAt,
		&msg.ProcessingStatus,
		&msg.HasXML,
		&msg.HasPDF,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInboxMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message detail view by id: %w", err)
	}

	attachments, err := r.ListMessageAttachments(ctx, messageID)
	if err != nil {
		return nil, err
	}
	for _, att := range attachments {
		mime := ""
		if att.MimeType != nil {
			mime = *att.MimeType
		}
		size := int64(0)
		if att.SizeBytes != nil {
			size = *att.SizeBytes
		}
		msg.Attachments = append(msg.Attachments, inboxPorts.AttachmentView{
			ID:        att.ID,
			Filename:  att.Filename,
			MimeType:  mime,
			SizeBytes: size,
		})
	}

	return &msg, nil
}

func (r *PostgresRepository) GetInboxMessageByID(ctx context.Context, messageID string) (*domain.InboxMessage, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		SELECT id, account_id, provider_message_id, provider_thread_id, subject, sender_email,
			snippet, to_emails, cc_emails, bcc_emails, folder, is_read, is_starred, is_draft,
			received_at, sync_status, raw_data, created_at, updated_at
		FROM email_messages
		WHERE id = $1
	`
	var snapshot domain.InboxMessageSnapshot
	var status string
	var folder string
	err = pool.QueryRow(ctx, query, messageID).Scan(
		&snapshot.ID,
		&snapshot.ConnectionID,
		&snapshot.ProviderMessageID,
		&snapshot.ProviderThreadID,
		&snapshot.Subject,
		&snapshot.SenderEmail,
		&snapshot.Snippet,
		&snapshot.ToEmails,
		&snapshot.CcEmails,
		&snapshot.BccEmails,
		&folder,
		&snapshot.IsRead,
		&snapshot.IsStarred,
		&snapshot.IsDraft,
		&snapshot.ReceivedAt,
		&status,
		&snapshot.RawData,
		&snapshot.CreatedAt,
		&snapshot.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInboxMessageNotFound
		}
		return nil, fmt.Errorf("failed to get inbox message: %w", err)
	}
	snapshot.Folder = domain.MailFolder(folder)
	snapshot.SyncStatus = domain.MessageSyncStatus(status)
	return domain.RehydrateInboxMessage(snapshot), nil
}

func (r *PostgresRepository) UpdateInboxMessageFlags(ctx context.Context, message *domain.InboxMessage) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	snapshot := message.Snapshot()
	_, err = pool.Exec(ctx, `
		UPDATE email_messages
		SET folder = $2, is_read = $3, is_starred = $4, is_draft = $5, updated_at = $6
		WHERE id = $1
	`, snapshot.ID, snapshot.Folder.String(), snapshot.IsRead, snapshot.IsStarred, snapshot.IsDraft, snapshot.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update inbox message flags: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetMessageAttachment(ctx context.Context, messageID, attachmentID string) (*domain.MessageAttachment, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	query := `
		SELECT id, message_id, filename, mime_type, size_bytes, sha256, s3_key, raw_data, created_at, updated_at
		FROM email_attachments
		WHERE message_id = $1 AND id = $2
	`
	var att domain.MessageAttachment
	err = pool.QueryRow(ctx, query, messageID, attachmentID).Scan(
		&att.ID,
		&att.MessageID,
		&att.Filename,
		&att.MimeType,
		&att.SizeBytes,
		&att.SHA256,
		&att.S3Key,
		&att.RawData,
		&att.CreatedAt,
		&att.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInboxMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message attachment: %w", err)
	}
	return &att, nil
}

func (r *PostgresRepository) GetMessageAttachmentByMessageAndSHA(ctx context.Context, messageID, sha256 string) (*domain.MessageAttachment, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}
	q := database.QuerierFromContext(ctx, pool)

	query := `
		SELECT id, message_id, filename, mime_type, size_bytes, sha256, s3_key, raw_data, created_at, updated_at
		FROM email_attachments
		WHERE message_id = $1 AND sha256 = $2
	`
	var att domain.MessageAttachment
	err = q.QueryRow(ctx, query, messageID, sha256).Scan(
		&att.ID,
		&att.MessageID,
		&att.Filename,
		&att.MimeType,
		&att.SizeBytes,
		&att.SHA256,
		&att.S3Key,
		&att.RawData,
		&att.CreatedAt,
		&att.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get message attachment by sha: %w", err)
	}
	return &att, nil
}

func (r *PostgresRepository) ListMessageAttachments(ctx context.Context, messageID string) ([]*domain.MessageAttachment, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant db pool: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT id, message_id, filename, mime_type, size_bytes, sha256, s3_key, raw_data, created_at, updated_at
		FROM email_attachments
		WHERE message_id = $1
		ORDER BY filename
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list message attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*domain.MessageAttachment
	for rows.Next() {
		var att domain.MessageAttachment
		if err := rows.Scan(
			&att.ID,
			&att.MessageID,
			&att.Filename,
			&att.MimeType,
			&att.SizeBytes,
			&att.SHA256,
			&att.S3Key,
			&att.RawData,
			&att.CreatedAt,
			&att.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message attachment: %w", err)
		}
		attachments = append(attachments, &att)
	}
	return attachments, rows.Err()
}

func defaultRawData(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}

	return raw
}

func nullableTextArray(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
