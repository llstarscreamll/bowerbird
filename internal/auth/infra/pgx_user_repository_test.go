package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql/tests"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpsert(t *testing.T) {
	// ToDo: get connection url from env var
	var db = postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()

	testCases := []struct {
		testCase     string
		scenarioRows []map[string]any
		user         domain.User
		expectedRows []map[string]any
	}{
		{
			"should insert a new user",
			[]map[string]any{},
			testUser,
			[]map[string]any{
				{"id": testUser.ID, "first_name": testUser.GivenName, "last_name": testUser.FamilyName, "email": testUser.Email, "photo_url": testUser.PictureUrl},
			},
		},
		{
			"should not to throw an error upserting an existing user email",
			[]map[string]any{
				{"id": "01JGCZXZEC0000000000000000", "first_name": testUser.GivenName, "last_name": testUser.FamilyName, "email": testUser.Email, "photo_url": testUser.PictureUrl},
			},
			testUser,
			[]map[string]any{
				{"id": "01JGCZXZEC0000000000000000", "first_name": testUser.GivenName, "last_name": testUser.FamilyName, "email": testUser.Email, "photo_url": testUser.PictureUrl},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCase, func(t *testing.T) {
			ctx := context.Background()
			tests.CleanUpTables(db, []string{"users"})
			tests.WriteScenarioRows(db, "users", tc.scenarioRows)

			repo := NewPgxUserRepository(db)
			err := repo.Upsert(ctx, tc.user)

			assert.Nil(t, err)
			tests.AssertDatabaseHasRows(t, db, "users", tc.expectedRows)
		})
	}
}
