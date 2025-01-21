package infra

import (
	"context"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql/tests"

	"github.com/stretchr/testify/assert"
)

func TestPgxUserRepositorySave(t *testing.T) {
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()

	testCases := []struct {
		name         string
		scenarioRows []map[string]any
		input        struct {
			userID         string
			sessionID      string
			expirationDate time.Time
		}
		expectedError error
		expectedRows  []map[string]any
	}{
		{
			"should store a new session",
			[]map[string]any{},
			struct {
				userID         string
				sessionID      string
				expirationDate time.Time
			}{"01JGCZXZEC00000000000000U1", "01JGCZXZEC00000000000000S1", time.Date(2025, time.January, 6, 0, 0, 0, 0, time.UTC)},
			nil,
			[]map[string]any{
				{
					"id":         "01JGCZXZEC00000000000000S1",
					"user_id":    "01JGCZXZEC00000000000000U1",
					"expires_at": time.Date(2025, time.January, 6, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tests.CleanUpTables(db, []string{"sessions"})
			tests.WriteScenarioRows(db, "users", tc.scenarioRows)

			repo := NewPgxSessionRepository(db)
			err := repo.Save(ctx, tc.input.sessionID, tc.input.userID, tc.input.expirationDate)

			assert.Equal(t, tc.expectedError, err)
			tests.AssertDatabaseHasRows(t, db, "sessions", tc.expectedRows)
		})
	}

}
