package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"testing"
	"time"

	"llstarscreamll/bowerbird/internal/common/infra/postgresql/tests"

	"github.com/stretchr/testify/assert"
)

func TestPgxMailMessageRepositoryUpsertMany(t *testing.T) {
	var db = postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()

	testsCases := []struct {
		name          string
		scenarioRows  []map[string]interface{}
		input         []domain.MailMessage
		expectedError error
		expectedRows  []map[string]interface{}
	}{
		{
			"should insert a new mail message",
			[]map[string]interface{}{
				{
					"id":          "01JGCZXZEC0000000000000001",
					"external_id": "external-id-1",
					"user_id":     "01JGCZXZEC00000000000000U1",
					"from":        "test@mail.com",
					"to":          "john@doe.com",
					"subject":     "Test subject",
					"body":        "Test body",
					"received_at": time.Date(2025, time.January, 6, 0, 0, 0, 0, time.UTC),
				},
			},
			[]domain.MailMessage{
				{
					ID:         "01JGCZXZEC0000000000000001",
					ExternalID: "external-id-1",
					UserID:     "01JGCZXZEC00000000000000U1",
					From:       "test@mail.com",
					To:         "john@doe.com",
					Subject:    "Test subject",
					Body:       "Test body",
					ReceivedAt: time.Date(2025, time.January, 6, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:         "01JGCZXZEC0000000000000002",
					ExternalID: "external-id-2",
					UserID:     "01JGCZXZEC00000000000000U1",
					From:       "another-test@mail.com",
					To:         "john-the-hacker@doe.com",
					Subject:    "Test subject II",
					Body:       "Test body II",
					ReceivedAt: time.Date(2025, time.January, 10, 0, 0, 0, 0, time.UTC),
				},
			},
			nil,
			[]map[string]interface{}{
				{
					"id":          "01JGCZXZEC0000000000000001",
					"external_id": "external-id-1",
					"user_id":     "01JGCZXZEC00000000000000U1",
					"from":        "test@mail.com",
					"to":          "john@doe.com",
					"subject":     "Test subject",
					"body":        "Test body",
					"received_at": time.Date(2025, time.January, 6, 0, 0, 0, 0, time.UTC),
				},
				{
					"id":          "01JGCZXZEC0000000000000002",
					"external_id": "external-id-2",
					"user_id":     "01JGCZXZEC00000000000000U1",
					"from":        "another-test@mail.com",
					"to":          "john-the-hacker@doe.com",
					"subject":     "Test subject II",
					"body":        "Test body II",
					"received_at": time.Date(2025, time.January, 10, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tests.CleanUpTables(db, []string{"mail_messages"})
			tests.WriteScenarioRows(db, "mail_messages", tc.scenarioRows)

			repo := NewPgxMailMessageRepository(db)
			err := repo.UpsertMany(ctx, tc.input)

			assert.Equal(t, tc.expectedError, err)
			tests.AssertDatabaseHasRows(t, db, "mail_messages", tc.expectedRows)
		})
	}
}
