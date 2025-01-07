package infra

import (
	"context"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
	"log"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
			cleanUpTables(db, []string{"users"})
			writeScenarioRows(db, "users", tc.scenarioRows)

			repo := NewPgxUserRepository(db)
			err := repo.Upsert(ctx, tc.user)

			assert.Nil(t, err)
			assertDatabaseHasRows(t, db, "users", tc.expectedRows)
		})
	}
}

func cleanUpTables(db *pgxpool.Pool, tables []string) {
	for _, t := range tables {
		db.Exec(context.Background(), fmt.Sprintf("DELETE FROM %s", t))
	}
}

func writeScenarioRows(db *pgxpool.Pool, tableName string, rows []map[string]any) {
	for _, row := range rows {
		var columns []string
		var values []interface{}
		var placeholders []string

		for k, v := range row {
			columns = append(columns, k)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)+1))
			values = append(values, v)
		}

		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "),
		)

		_, err := db.Exec(
			context.Background(),
			query,
			values...,
		)
		if err != nil {
			panic(err)
		}
	}
}

func assertDatabaseHasRows(t *testing.T, db *pgxpool.Pool, tableName string, expectedRecords []map[string]any) {
	dbRows, err := db.Query(context.Background(), fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		log.Fatal(err)
	}

	results, err := pgx.CollectRows(dbRows, pgx.RowToMap)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, len(expectedRecords), len(results), "Mismatched database rows count, expected %d, got %d", len(expectedRecords), len(results))

	for _, expected := range expectedRecords {
		for _, result := range results {

			equalColumns := 0
			for k := range expected {
				if expected[k] == result[k] {
					equalColumns++
				}
			}

			assert.True(t, equalColumns == len(slices.Collect(maps.Keys(expected))), "Expected row not found in database: %v", expected)
		}
	}
}
