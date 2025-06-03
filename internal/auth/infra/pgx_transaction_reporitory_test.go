package infra

import (
	"context"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql/tests"

	"github.com/stretchr/testify/assert"
)

func TestUpsertManyShouldReturnNilWhenDataPersistedIsOk(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()
	tests.CleanUpTables(db, []string{"transactions"})

	transactions := []domain.Transaction{
		{
			ID:                "000000000000000000000000T1",
			WalletID:          "000000000000000000000000W1",
			UserID:            "000000000000000000000000U1",
			Origin:            "test",
			Type:              "expense",
			Amount:            -100,
			UserDescription:   "",
			SystemDescription: "test system description",
			CategoryID:        "000000000000000000000000C1",
			CategorySetterID:  "00000000000000000000000000",
			ProcessedAt:       time.Now(),
			CreatedAt:         time.Now(),
		},
		{
			// no category specified
			ID:                "000000000000000000000000T2",
			WalletID:          "000000000000000000000000W1",
			UserID:            "000000000000000000000000U1",
			Origin:            "test",
			Type:              "expense",
			Amount:            -50,
			SystemDescription: "another test",
		},
	}

	repo := NewPgxTransactionRepository(db)
	err := repo.UpsertMany(context.Background(), transactions)

	assert.NoError(t, err)
	tests.AssertDatabaseHasRows(t, db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -100.00,
			"user_description":   "",
			"system_description": "test system description",
			"category_id":        "000000000000000000000000C1",
			"category_setter_id": "00000000000000000000000000",
		},
		{
			"id":                 "000000000000000000000000T2",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -50.00,
			"system_description": "another test",
			// default values
			"user_description":   "",
			"category_id":        nil,
			"category_setter_id": nil,
		},
	})
}
