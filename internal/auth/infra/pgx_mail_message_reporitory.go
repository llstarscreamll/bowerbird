package infra

import (
	"context"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxMailMessageRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxMailMessageRepository) UpsertMany(ctx context.Context, messages []domain.MailMessage) error {
	if len(messages) == 0 {
		return nil
	}

	placeHolders := make([]string, 0, len(messages))
	values := make([]interface{}, 0, len(messages)*8)

	for i, v := range messages {
		placeHolders = append(placeHolders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*8+1, i*8+2, i*8+3, i*8+4, i*8+5, i*8+6, i*8+7, i*8+8))
		values = append(values, v.ID, v.ExternalID, v.UserID, v.From, v.To, v.Subject, v.Body, v.ReceivedAt)
	}

	query := fmt.Sprintf(`
	INSERT INTO mail_messages (id, external_id, user_id, "from", "to", subject, body, received_at)
	VALUES %s
	ON CONFLICT (external_id, user_id) DO NOTHING
	`, strings.Join(placeHolders, ", "))

	_, err := r.pool.Exec(
		ctx,
		query,
		values...,
	)

	return err
}

func NewPgxMailMessageRepository(pool *pgxpool.Pool) *PgxMailMessageRepository {
	return &PgxMailMessageRepository{pool: pool}
}
