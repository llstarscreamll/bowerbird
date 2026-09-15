package postgres

import (
	"context"
	"fmt"

	inboxapi "github.com/bowerbird/internal/inbox/api"
)

func (r *PostgresRepository) ListExtractionCandidates(ctx context.Context, cursor string, limit int) (inboxapi.ExtractionCandidatePage, error) {
	if limit <= 0 {
		limit = 50
	}
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return inboxapi.ExtractionCandidatePage{}, fmt.Errorf("get tenant db pool: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT m.id
		FROM email_messages m
		WHERE ($1 = '' OR m.id > $1)
		  AND EXISTS (
			SELECT 1 FROM email_attachments a
			WHERE a.message_id = m.id
			  AND (
				LOWER(a.filename) LIKE '%.xml'
				OR LOWER(a.filename) LIKE '%.pdf'
				OR LOWER(a.filename) LIKE '%.zip'
				OR COALESCE(a.mime_type, '') ILIKE '%xml%'
				OR COALESCE(a.mime_type, '') ILIKE '%pdf%'
				OR COALESCE(a.mime_type, '') ILIKE '%zip%'
			  )
		  )
		ORDER BY m.id ASC
		LIMIT $2
	`, cursor, limit)
	if err != nil {
		return inboxapi.ExtractionCandidatePage{}, fmt.Errorf("list extraction candidate ids: %w", err)
	}
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return inboxapi.ExtractionCandidatePage{}, fmt.Errorf("scan candidate id: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return inboxapi.ExtractionCandidatePage{}, err
	}
	if len(ids) == 0 {
		return inboxapi.ExtractionCandidatePage{}, nil
	}

	attRows, err := pool.Query(ctx, `
		SELECT m.id, COALESCE(m.subject, ''), COALESCE(m.snippet, ''),
		       a.s3_key, a.filename, COALESCE(a.mime_type, '')
		FROM email_messages m
		JOIN email_attachments a ON a.message_id = m.id
		WHERE m.id = ANY($1)
		ORDER BY m.id ASC, a.id ASC
	`, ids)
	if err != nil {
		return inboxapi.ExtractionCandidatePage{}, fmt.Errorf("list extraction candidate attachments: %w", err)
	}
	defer attRows.Close()

	byID := make(map[string]*inboxapi.ExtractionCandidate, len(ids))
	order := make([]string, 0, len(ids))
	for attRows.Next() {
		var messageID, subject, snippet, s3Key, filename, mimeType string
		if err := attRows.Scan(&messageID, &subject, &snippet, &s3Key, &filename, &mimeType); err != nil {
			return inboxapi.ExtractionCandidatePage{}, fmt.Errorf("scan candidate attachment: %w", err)
		}
		item, ok := byID[messageID]
		if !ok {
			item = &inboxapi.ExtractionCandidate{MessageID: messageID, Subject: subject, Snippet: snippet}
			byID[messageID] = item
			order = append(order, messageID)
		}
		item.Attachments = append(item.Attachments, inboxapi.AttachmentRef{
			S3Key:    s3Key,
			Filename: filename,
			MimeType: mimeType,
		})
	}
	if err := attRows.Err(); err != nil {
		return inboxapi.ExtractionCandidatePage{}, err
	}

	page := inboxapi.ExtractionCandidatePage{Items: make([]inboxapi.ExtractionCandidate, 0, len(order))}
	for _, id := range order {
		page.Items = append(page.Items, *byID[id])
	}
	if len(ids) == limit {
		page.NextCursor = ids[len(ids)-1]
	}
	return page, nil
}
