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

var currentTime = time.Now()

func TestUpsertManyShouldPersistManyRecordsAtOnce(t *testing.T) {
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

func TestUpsertManyShouldUpdateCategoryAndCategorySetterWhenCategorySetterOnStorageIsNull(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()
	tests.CleanUpTables(db, []string{"transactions"})

	tests.WriteScenarioRows(db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "test system description",
			"reference":          currentTime.Format("20060102") + "/test/test system description/-10.000000/0",
			"category_id":        nil,
			"category_setter_id": nil,
			"processed_at":       currentTime,
		},
	})

	tests.AssertDatabaseHasRows(t, db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"reference":          currentTime.Format("20060102") + "/test/test system description/-10.000000/0",
			"category_id":        nil,
			"category_setter_id": nil,
		},
	})

	input := []domain.Transaction{
		{
			ID:                "000000000000000000000000T2",
			WalletID:          "000000000000000000000000W1",
			UserID:            "000000000000000000000000U1",
			Origin:            "test",
			Type:              "expense",
			Amount:            -10.00,
			UserDescription:   "",
			SystemDescription: "test system description",
			CategoryID:        "000000000000000000000000C1",
			CategorySetterID:  "00000000000000000000000000",
			ProcessedAt:       currentTime,
			CreatedAt:         currentTime,
		},
	}

	repo := NewPgxTransactionRepository(db)
	err := repo.UpsertMany(context.Background(), input)

	assert.NoError(t, err)
	tests.AssertDatabaseHasRows(t, db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"category_id":        "000000000000000000000000C1",
			"category_setter_id": "00000000000000000000000000",
		},
	})
}

func TestUpsertManyShouldUpdateCategoryAndCategorySetterWhenCategorySetterOnStorageIsEmptyString(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()
	tests.CleanUpTables(db, []string{"transactions"})

	tests.WriteScenarioRows(db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "test system description",
			"reference":          currentTime.Format("20060102") + "/test/test system description/-10.000000/0",
			"category_id":        "",
			"category_setter_id": "",
			"processed_at":       currentTime,
		},
	})

	input := []domain.Transaction{
		{
			ID:                "000000000000000000000000T2",
			WalletID:          "000000000000000000000000W1",
			UserID:            "000000000000000000000000U1",
			Origin:            "test",
			Type:              "expense",
			Amount:            -10.00,
			UserDescription:   "",
			SystemDescription: "test system description",
			CategoryID:        "000000000000000000000000C1",
			CategorySetterID:  "00000000000000000000000000",
			ProcessedAt:       currentTime,
			CreatedAt:         currentTime,
		},
	}

	repo := NewPgxTransactionRepository(db)
	err := repo.UpsertMany(context.Background(), input)

	assert.NoError(t, err)
	tests.AssertDatabaseHasRows(t, db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "test system description",
			"category_id":        "000000000000000000000000C1",
			"category_setter_id": "00000000000000000000000000",
		},
	})
}

func TestUpsertManyShouldUpdateCategoryAndCategorySetterWhenCategorySetterOnStorageIsEmptyUlid(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()
	tests.CleanUpTables(db, []string{"transactions"})

	tests.WriteScenarioRows(db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "test system description",
			"reference":          currentTime.Format("20060102") + "/test/test system description/-10.000000/0",
			"category_id":        "",
			"category_setter_id": "00000000000000000000000000",
			"processed_at":       currentTime,
		},
	})

	input := []domain.Transaction{
		{
			ID:                "000000000000000000000000T2",
			WalletID:          "000000000000000000000000W1",
			UserID:            "000000000000000000000000U1",
			Origin:            "test",
			Type:              "expense",
			Amount:            -10.00,
			UserDescription:   "",
			SystemDescription: "test system description",
			CategoryID:        "000000000000000000000000C1",
			CategorySetterID:  "00000000000000000000000000",
			ProcessedAt:       currentTime,
			CreatedAt:         currentTime,
		},
	}

	repo := NewPgxTransactionRepository(db)
	err := repo.UpsertMany(context.Background(), input)

	assert.NoError(t, err)
	tests.AssertDatabaseHasRows(t, db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "test system description",
			"category_id":        "000000000000000000000000C1",
			"category_setter_id": "00000000000000000000000000",
		},
	})
}

func TestUpsertManyShouldNotUpdateCategoryAndCategorySetterWhenCategorySetterOnStorageIsANonEmptyUlid(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()
	tests.CleanUpTables(db, []string{"transactions"})

	tests.WriteScenarioRows(db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "test system description",
			"reference":          currentTime.Format("20060102") + "/test/test system description/-10.000000/0",
			"category_id":        "000000000000000000000000C1",
			"category_setter_id": "000000000000000000000000U1",
			"processed_at":       currentTime,
		},
	})

	input := []domain.Transaction{
		{
			ID:                "000000000000000000000000T2",
			WalletID:          "000000000000000000000000W1",
			UserID:            "000000000000000000000000U1",
			Origin:            "test",
			Type:              "expense",
			Amount:            -10.00,
			UserDescription:   "",
			SystemDescription: "test system description",
			CategoryID:        "000000000000000000000000C2",
			CategorySetterID:  "00000000000000000000000000",
			ProcessedAt:       currentTime,
			CreatedAt:         currentTime,
		},
	}

	repo := NewPgxTransactionRepository(db)
	err := repo.UpsertMany(context.Background(), input)

	assert.NoError(t, err)
	tests.AssertDatabaseHasRows(t, db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "test system description",
			"category_id":        "000000000000000000000000C1",
			"category_setter_id": "000000000000000000000000U1",
		},
	})
}

func TestUpsertManyShouldUpdateSystemDescriptionWhenTheNewOneIsLongerThanTheOldOne(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()
	tests.CleanUpTables(db, []string{"transactions"})

	tests.WriteScenarioRows(db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"wallet_id":          "000000000000000000000000W1",
			"user_id":            "000000000000000000000000U1",
			"origin":             "test",
			"type":               "expense",
			"amount":             -10.00,
			"user_description":   "",
			"system_description": "COMPANIA DE SERVICIOS PUBLICOS DE",
			"reference":          currentTime.Format("20060102") + "/test/compania de servicios publicos de/-10.000000/0",
			"processed_at":       currentTime,
		},
	})

	input := []domain.Transaction{
		{
			ID:                "000000000000000000000000T2",
			WalletID:          "000000000000000000000000W1",
			UserID:            "000000000000000000000000U1",
			Origin:            "test",
			Type:              "expense",
			Amount:            -10.00,
			SystemDescription: "COMPANIA DE SERVICIOS PUBLICOS DE SOGAMOSO",
			ProcessedAt:       currentTime,
			CreatedAt:         currentTime,
		},
	}

	repo := NewPgxTransactionRepository(db)
	err := repo.UpsertMany(context.Background(), input)

	assert.NoError(t, err)
	tests.AssertDatabaseHasRows(t, db, "transactions", []map[string]any{
		{
			"id":                 "000000000000000000000000T1",
			"system_description": "COMPANIA DE SERVICIOS PUBLICOS DE SOGAMOSO",
		},
	})
}
