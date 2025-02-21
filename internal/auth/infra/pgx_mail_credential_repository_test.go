package infra

import (
	"context"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/auth/testdata"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql/tests"

	"github.com/stretchr/testify/assert"
)

func TestPgxMailCredentialRepositorySave(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()

	testCases := []struct {
		name         string
		scenarioRows []map[string]any
		input        struct {
			ID             string
			userID         string
			walletID       string
			mailProvider   string
			mailAddress    string
			accessToken    string
			refreshToken   string
			expirationDate time.Time
		}
		expectedError error
		expectedRows  []map[string]any
	}{
		{
			"should store a new mail credential",
			[]map[string]any{},
			struct {
				ID             string
				userID         string
				walletID       string
				mailProvider   string
				mailAddress    string
				accessToken    string
				refreshToken   string
				expirationDate time.Time
			}{
				"01JJ3A890N0000000000000000",
				testdata.TestUser.ID,
				"01JMJHAD350000000000000000",
				"google",
				"john@doe.com",
				"some-access-token",
				"some-refresh-token",
				time.Date(2025, time.January, 6, 0, 0, 0, 0, time.UTC),
			},
			nil,
			[]map[string]any{
				{
					"id":            "01JJ3A890N0000000000000000",
					"user_id":       testdata.TestUser.ID,
					"wallet_id":     "01JMJHAD350000000000000000",
					"mail_provider": "google",
					"mail_address":  "john@doe.com",
					"access_token":  "some-access-token",
					"refresh_token": "some-refresh-token",
					"expires_at":    time.Date(2025, time.January, 6, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tests.CleanUpTables(db, []string{"mail_credentials"})
			tests.WriteScenarioRows(db, "mail_credentials", tc.scenarioRows)

			repo := NewPgxMailCredentialRepository(db)
			err := repo.Save(ctx, tc.input.ID, tc.input.userID, tc.input.walletID, tc.input.mailProvider, tc.input.mailAddress, tc.input.accessToken, tc.input.refreshToken, tc.input.expirationDate)

			assert.Equal(t, tc.expectedError, err)
			tests.AssertDatabaseHasRows(t, db, "mail_credentials", tc.expectedRows)
		})
	}

}
