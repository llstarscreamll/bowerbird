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
	"github.com/jackc/pgx/v5/pgtype"
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
			columns = append(columns, "\""+k+"\"")
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

	actualRows, err := pgx.CollectRows(dbRows, pgx.RowToMap)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, len(expectedRecords), len(actualRows), "Mismatched database rows count, expected %d, got %d", len(expectedRecords), len(actualRows))

	for _, expected := range expectedRecords {
		expectedRecordFound := false
		columnsNotFount := make(map[string]interface{})

		for _, actualRow := range actualRows {
			equalColumns := 0
			columnsNotFount = make(map[string]interface{})

			fmt.Printf("Comparing DB rows:\n%#v\n%#v\n", expected, actualRow)

			for expectedColumnName := range expected {

				_, ok := actualRow[expectedColumnName]
				if !ok {
					columnsNotFount[expectedColumnName] = expected[expectedColumnName]
					continue
				}

				expectedTime, ok1 := expected[expectedColumnName].(time.Time)
				resultTime, ok2 := actualRow[expectedColumnName].(time.Time)
				if ok1 && ok2 {
					if expectedTime.Equal(resultTime) {
						equalColumns++
						continue
					}
				}

				expectedFloat, ok1 := expected[expectedColumnName].(float64)
				if ok1 {
					actualFloat, ok := actualRow[expectedColumnName].(pgtype.Numeric)
					if ok {
						float, _ := actualFloat.Float64Value()
						if expectedFloat == float.Float64 {
							equalColumns++
							continue
						}
					}
				}

				if expected[expectedColumnName] == actualRow[expectedColumnName] {
					equalColumns++
					continue
				}

				columnsNotFount[expectedColumnName] = expected[expectedColumnName]
			}

			expectedRecordFound = equalColumns == len(slices.Collect(maps.Keys(expected)))
			if expectedRecordFound {
				columnsNotFount = make(map[string]interface{})
				break
			}
		}

		assert.True(t, expectedRecordFound, "Expected row not found in DB, here is what was expected:\n%#v\nHere is what was found:\n%#v\nConflicting columns:\n%#v", expected, actualRows, columnsNotFount)
	}
}
