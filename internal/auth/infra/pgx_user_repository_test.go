package infra

import (
	"context"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql/tests"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPgxUserRepositoryUpsert(t *testing.T) {
	// ToDo: get connection url from env var
	var db = postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()

	testCases := []struct {
		testCase      string
		scenarioRows  []map[string]any
		user          domain.User
		expectedID    string
		expectedError error
		expectedRows  []map[string]any
	}{
		{
			"should insert a new user",
			[]map[string]any{},
			testUser,
			testUser.ID, nil,
			[]map[string]any{
				{"id": testUser.ID, "first_name": testUser.GivenName, "last_name": testUser.FamilyName, "email": testUser.Email, "photo_url": testUser.PictureUrl},
			},
		},
		{
			"should not to throw an error upserting an already existing user email",
			[]map[string]any{
				{"id": "01JGCZXZEC0000000000000001", "first_name": testUser.GivenName, "last_name": testUser.FamilyName, "email": testUser.Email, "photo_url": testUser.PictureUrl},
			},
			testUser,
			"01JGCZXZEC0000000000000001", nil,
			[]map[string]any{
				{"id": "01JGCZXZEC0000000000000001", "first_name": testUser.GivenName, "last_name": testUser.FamilyName, "email": testUser.Email, "photo_url": testUser.PictureUrl},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCase, func(t *testing.T) {
			ctx := context.Background()
			tests.CleanUpTables(db, []string{"users"})
			tests.WriteScenarioRows(db, "users", tc.scenarioRows)

			repo := NewPgxUserRepository(db)
			ID, err := repo.Upsert(ctx, tc.user)

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedID, ID)
			tests.AssertDatabaseHasRows(t, db, "users", tc.expectedRows)
		})
	}
}

func TestPgxUserRepositoryGetByID(t *testing.T) {
	// ToDo: get connection url from env var
	var db = postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()

	testCases := []struct {
		testCase       string
		scenarioRows   []map[string]any
		input          string
		expectedResult domain.User
		expectedError  error
	}{
		{
			"should return a user by given ID when it does exist",
			[]map[string]any{
				{"id": testUser.ID, "first_name": testUser.GivenName, "last_name": testUser.FamilyName, "email": testUser.Email, "photo_url": testUser.PictureUrl},
			},
			testUser.ID,
			testUser,
			nil,
		},
		{
			"should return empty user when given user ID does not exists",
			[]map[string]any{},
			testUser.ID,
			domain.User{},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCase, func(t *testing.T) {
			ctx := context.Background()
			tests.CleanUpTables(db, []string{"users"})
			tests.WriteScenarioRows(db, "users", tc.scenarioRows)

			repo := NewPgxUserRepository(db)
			result, err := repo.GetByID(ctx, tc.input)

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
