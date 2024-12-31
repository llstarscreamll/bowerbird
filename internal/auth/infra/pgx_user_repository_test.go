package infra

import (
	"context"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"log"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestUpsert(t *testing.T) {
	// ToDo: get connection url from env var
	db := postgresql.CreatePgxConnectionPool(context.Background(), "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable")
	defer db.Close()

	testCases := []struct {
		testCase     string
		scenarioRows []struct{ id, first_name, last_name, email, photo_url string }
		user         domain.User
		expectedRows []struct{ id, first_name, last_name, email, photo_url string }
	}{
		{
			"should insert a new user",
			[]struct{ id, first_name, last_name, email, photo_url string }{},
			testUser,
			[]struct{ id, first_name, last_name, email, photo_url string }{
				{testUser.ID, testUser.GivenName, testUser.FamilyName, testUser.Email, testUser.PictureUrl},
			},
		},
		{
			"should not to throw an error upserting an existing user email",
			[]struct{ id, first_name, last_name, email, photo_url string }{
				{"01JGCZXZEC0000000000000000", testUser.GivenName, testUser.FamilyName, testUser.Email, testUser.PictureUrl},
			},
			testUser,
			[]struct{ id, first_name, last_name, email, photo_url string }{
				{"01JGCZXZEC0000000000000000", testUser.GivenName, testUser.FamilyName, testUser.Email, testUser.PictureUrl},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCase, func(t *testing.T) {
			ctx := context.Background()
			cleanUpTables(db, []string{"users"})
			writeScenarioData(db, tc.scenarioRows)

			repo := NewPgxUserRepository(db)
			err := repo.Upsert(ctx, tc.user)

			assert.Nil(t, err)
			assertDatabaseHasRecords(t, db, tc.expectedRows)
		})
	}
}

func cleanUpTables(db *pgxpool.Pool, tables []string) {
	for _, t := range tables {
		db.Exec(context.Background(), fmt.Sprintf("DELETE FROM %s", t))
	}
}

func writeScenarioData(db *pgxpool.Pool, rows []struct{ id, first_name, last_name, email, photo_url string }) {
	for _, r := range rows {
		_, err := db.Exec(context.Background(), "INSERT INTO users (id, first_name, last_name, email, photo_url) VALUES ($1, $2, $3, $4, $5)", r.id, r.first_name, r.last_name, r.email, r.photo_url)
		if err != nil {
			panic(err)
		}
	}
}

func assertDatabaseHasRecords(t *testing.T, db *pgxpool.Pool, expectedRows []struct{ id, first_name, last_name, email, photo_url string }) {
	countResult, err := db.Query(context.Background(), "SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}

	var results []struct {
		id, first_name, last_name, email, photo_url string
		created_at, updated_at                      time.Time
	}

	for countResult.Next() {
		var r struct {
			id, first_name, last_name, email, photo_url string
			created_at, updated_at                      time.Time
		}
		err := countResult.Scan(&r.id, &r.first_name, &r.last_name, &r.email, &r.photo_url, &r.created_at, &r.updated_at)
		if err != nil {
			log.Fatal(err)
		}

		results = append(results, r)
	}

	assert.Equal(t, len(expectedRows), len(results), "Mismatched database rows count, expected %d, got %d", len(expectedRows), len(results))

	expectedRowFound := slices.ContainsFunc(results, func(result struct {
		id, first_name, last_name, email, photo_url string
		created_at, updated_at                      time.Time
	}) bool {
		return slices.ContainsFunc(expectedRows, func(expected struct {
			id, first_name, last_name, email, photo_url string
		}) bool {
			return expected.id == result.id && expected.first_name == result.first_name && expected.last_name == result.last_name && expected.email == result.email && expected.photo_url == result.photo_url
		})
	})

	assert.True(t, expectedRowFound, "Expected row not found in database")
}
