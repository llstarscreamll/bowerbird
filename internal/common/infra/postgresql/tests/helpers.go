package tests

import (
	"context"
	"fmt"
	"log"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func CleanUpTables(db *pgxpool.Pool, tables []string) {
	for _, t := range tables {
		db.Exec(context.Background(), fmt.Sprintf("DELETE FROM %s", t))
	}
}

func WriteScenarioRows(db *pgxpool.Pool, tableName string, rows []map[string]any) {
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
			log.Fatal(err)
		}
	}
}

func AssertDatabaseHasRows(t *testing.T, db *pgxpool.Pool, tableName string, expectedRecords []map[string]any) {
	dbRows, err := db.Query(context.Background(), fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		log.Fatal(err)
	}

	results, err := pgx.CollectRows(dbRows, pgx.RowToMap)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, len(expectedRecords), len(results), "Mismatched database rows count, expected %d, got %d", len(expectedRecords), len(results))

	for _, expected := range expectedRecords {
		expectedRecordFound := false
		for _, result := range results {

			equalColumns := 0
			for k := range expected {
				fmt.Printf("Comparing DB column %s: %v, %v\n", k, expected[k], result[k])

				expectedTime, ok1 := expected[k].(time.Time)
				resultTime, ok2 := result[k].(time.Time)
				if ok1 && ok2 {
					if expectedTime.Equal(resultTime) {
						equalColumns++
						continue
					}
				}

				if expected[k] == result[k] {
					equalColumns++
				}
			}

			expectedRecordFound = equalColumns == len(slices.Collect(maps.Keys(expected)))
			if expectedRecordFound {
				break
			}
		}
		assert.True(t, expectedRecordFound, "Expected row not found in database: %v", expected)
	}
}
