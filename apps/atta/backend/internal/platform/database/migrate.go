package database

import (
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes database migrations from the specified folder against the target DB.
func RunMigrations(dbURL string, migrationsPath string) error {
	log.Printf("Running migrations from %s", migrationsPath)

	m, err := migrate.New(fmt.Sprintf("file://%s", migrationsPath), dbURL)
	if err != nil {
		return fmt.Errorf("failed to init migrate: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrate up: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Printf("No new migrations to apply for %s", migrationsPath)
	} else {
		log.Printf("Migrations applied successfully from %s", migrationsPath)
	}

	return nil
}
