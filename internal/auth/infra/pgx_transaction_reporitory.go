package infra

import (
	"context"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxTransactionRepository struct {
	pool *pgxpool.Pool
}

func (r *PgxTransactionRepository) UpsertMany(ctx context.Context, transactions []domain.Transaction) error {
	placeHolders := make([]string, 0, len(transactions))
	values := make([]interface{}, 0, len(transactions)*8)

	for i, v := range transactions {
		placeHolders = append(placeHolders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*8+1, i*8+2, i*8+3, i*8+4, i*8+5, i*8+6, i*8+7, i*8+8, i*8+9, i*8+10))
		values = append(values, v.ID, v.WalletID, v.UserID, v.Origin, v.Type, v.Amount, v.UserDescription, v.SystemDescription, v.ProcessedAt, v.CreatedAt)
	}

	query := fmt.Sprintf(`
	INSERT INTO transactions (id, wallet_id, user_id, origin, reference, "type", amount, user_description, system_description, processed_at, created_at)
	VALUES %s
	ON CONFLICT (wallet_id, reference) DO NOTHING
	`, strings.Join(placeHolders, ", "))

	_, err := r.pool.Exec(
		ctx,
		query,
		values...,
	)

	return err
}

func (r *PgxTransactionRepository) FindByWalletID(ctx context.Context, walletID string) ([]domain.Transaction, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, wallet_id, user_id, origin, reference, "type", amount, user_description, system_description, processed_at, created_at
		FROM transactions
		WHERE wallet_id = $1
		ORDER BY processed_at DESC
		LIMIT 100`,
		walletID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	transactions := make([]domain.Transaction, 0)

	for rows.Next() {
		t := domain.Transaction{}

		err := rows.Scan(&t.ID, &t.WalletID, &t.UserID, &t.Origin, &t.Reference, &t.Type, &t.Amount, &t.UserDescription, &t.SystemDescription, &t.ProcessedAt, &t.CreatedAt)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, t)
	}

	return transactions, nil
}

func NewPgxTransactionRepository(pool *pgxpool.Pool) *PgxTransactionRepository {
	return &PgxTransactionRepository{pool: pool}
}
