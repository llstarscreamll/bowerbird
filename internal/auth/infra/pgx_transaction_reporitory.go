package infra

import (
	"context"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxTransactionRepository struct {
	pool *pgxpool.Pool
}

// ToDo: handle the case when there are more than 100 transactions
func (r *PgxTransactionRepository) UpsertMany(ctx context.Context, transactions []domain.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	seen := make(map[string]domain.Transaction)
	deduped := make([]domain.Transaction, 0)

	for _, t := range transactions {
		key := t.WalletID + "|" + t.Reference()
		if _, exists := seen[key]; !exists {
			seen[key] = t
		}
	}

	for _, t := range seen {
		deduped = append(deduped, t)
	}

	placeHolders := make([]string, 0, len(deduped))
	values := make([]interface{}, 0, len(deduped)*13)

	for i, v := range deduped {
		placeHolders = append(placeHolders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*13+1, i*13+2, i*13+3, i*13+4, i*13+5, i*13+6, i*13+7, i*13+8, i*13+9, i*13+10, i*13+11, i*13+12, i*13+13))

		var categoryID interface{} = v.CategoryID
		if v.CategoryID == "" {
			categoryID = nil
		}

		var categorySetterID interface{} = v.CategorySetterID
		if v.CategorySetterID == "" {
			categorySetterID = nil
		}

		values = append(values, v.ID, v.WalletID, v.UserID, v.Origin, v.Reference(), v.Type, v.Amount, v.UserDescription, v.SystemDescription, v.ProcessedAt, v.CreatedAt, categoryID, categorySetterID)
	}

	query := fmt.Sprintf(`
	INSERT INTO transactions (id, wallet_id, user_id, origin, reference, "type", amount, user_description, system_description, processed_at, created_at, category_id, category_setter_id)
	VALUES %s
	ON CONFLICT (wallet_id, reference) DO UPDATE
	SET origin = EXCLUDED.origin,
		system_description = CASE
			WHEN EXCLUDED.system_description > transactions.system_description THEN EXCLUDED.system_description
			ELSE transactions.system_description
		END,
		category_id = CASE 
			WHEN transactions.category_setter_id IS NULL OR transactions.category_setter_id = '' OR transactions.category_setter_id = '00000000000000000000000000'
			THEN EXCLUDED.category_id
			ELSE transactions.category_id
		END,
		category_setter_id = CASE
			WHEN transactions.category_setter_id IS NULL OR transactions.category_setter_id = '' OR transactions.category_setter_id = '00000000000000000000000000'
			THEN EXCLUDED.category_setter_id
			ELSE transactions.category_setter_id
		END
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
		`SELECT t.id,
				t.wallet_id,
		 		t.user_id,
		 		t.origin,
		 		t."type",
		 		t.amount,
		 		t.user_description,
		 		t.system_description,
		 		t.processed_at,
		 		t.created_at,
		 		COALESCE(c.id, '') as category_id,
		 		COALESCE(c.name, '') as category_name,
		 		COALESCE(c.color, '') as category_color,
				COALESCE(c.icon, '') as category_icon
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.wallet_id = $1
		ORDER BY t.processed_at DESC
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

		err := rows.Scan(&t.ID,
			&t.WalletID,
			&t.UserID,
			&t.Origin,
			&t.Type,
			&t.Amount,
			&t.UserDescription,
			&t.SystemDescription,
			&t.ProcessedAt,
			&t.CreatedAt,
			&t.CategoryID,
			&t.CategoryName,
			&t.CategoryColor,
			&t.CategoryIcon)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, t)
	}

	return transactions, nil
}

func (r *PgxTransactionRepository) GetByWalletIDAndID(ctx context.Context, walletID, transactionID string) (domain.Transaction, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT transactions.id,
				transactions.wallet_id,
				transactions.user_id,
				CONCAT(users.first_name, ' ', users.last_name) as name,
				transactions.origin,
				transactions."type",
				transactions.amount,
				transactions.user_description,
				transactions.system_description,
				transactions.processed_at,
				transactions.created_at,
				COALESCE(categories.name, '') as category_name,
				COALESCE(categories.color, '') as category_color,
				COALESCE(categories.icon, '') as category_icon,
				COALESCE(categories.id, '') as category_id
		FROM transactions
		INNER JOIN users ON transactions.user_id = users.id
		LEFT JOIN categories ON transactions.category_id = categories.id
		WHERE transactions.wallet_id = $1 AND transactions.id = $2`,
		walletID, transactionID,
	)

	var t domain.Transaction
	err := row.Scan(&t.ID,
		&t.WalletID,
		&t.UserID,
		&t.UserName,
		&t.Origin,
		&t.Type,
		&t.Amount,
		&t.UserDescription,
		&t.SystemDescription,
		&t.ProcessedAt,
		&t.CreatedAt,
		&t.CategoryName,
		&t.CategoryColor,
		&t.CategoryIcon,
		&t.CategoryID,
	)

	if err != nil {
		return domain.Transaction{}, err
	}

	return t, nil
}

func (r *PgxTransactionRepository) Update(ctx context.Context, transaction domain.Transaction) error {
	_, err := r.pool.Exec(
		ctx,
		`UPDATE transactions SET category_id = $1, category_setter_id = $2 WHERE id = $3 AND wallet_id = $4`,
		transaction.CategoryID, transaction.CategorySetterID, transaction.ID, transaction.WalletID,
	)

	return err
}

func (r *PgxTransactionRepository) GetMetrics(ctx context.Context, walletID string, from, to time.Time) (domain.Metrics, error) {
	row := r.pool.QueryRow(
		ctx,
		`SELECT SUM(CASE WHEN "type" = 'income' THEN amount ELSE 0 END) as total_income,
				SUM(CASE WHEN "type" = 'expense' THEN amount ELSE 0 END) as total_expense
		FROM transactions
		WHERE wallet_id = $1 AND processed_at BETWEEN $2 AND $3`,
		walletID, from, to,
	)

	var metrics domain.Metrics
	err := row.Scan(&metrics.TotalIncome, &metrics.TotalExpense)
	if err != nil {
		return domain.Metrics{}, err
	}

	metrics.From = from
	metrics.To = to

	return metrics, nil
}

func NewPgxTransactionRepository(pool *pgxpool.Pool) *PgxTransactionRepository {
	return &PgxTransactionRepository{pool: pool}
}
