package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context) error {
	cfg, err := config.Load(ctx)
	if err != nil {
		return err
	}

	directURL := cfg.DirectDatabaseURL()

	log.Println("control-plane migrations starting")
	if err := database.RunMigrations(directURL, "migrations/controlplane"); err != nil {
		return fmt.Errorf("control-plane migrations: %w", err)
	}
	log.Println("control-plane migrations completed")

	log.Println("tenant migrations starting")
	if err := runTenantMigrations(ctx, directURL); err != nil {
		return fmt.Errorf("tenant migrations: %w", err)
	}
	log.Println("tenant migrations completed")
	return nil
}

func runTenantMigrations(ctx context.Context, controlPlaneURL string) error {
	conn, err := pgx.Connect(ctx, controlPlaneURL)
	if err != nil {
		return fmt.Errorf("connect to control plane: %w", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, "SELECT db_name FROM tenants WHERE status = 'active'")
	if err != nil {
		log.Printf("could not query tenants: %v", err)
		return nil
	}
	defer rows.Close()

	var tenantDBs []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return fmt.Errorf("scan tenant row: %w", err)
		}
		tenantDBs = append(tenantDBs, dbName)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(tenantDBs) == 0 {
		log.Println("no active tenants found to migrate")
		return nil
	}

	for _, dbName := range tenantDBs {
		log.Printf("migrating tenant database %s", dbName)
		if err := database.RunMigrations(buildTenantURL(controlPlaneURL, dbName), "migrations/tenant"); err != nil {
			return fmt.Errorf("migrate tenant %s: %w", dbName, err)
		}
	}
	return nil
}

func buildTenantURL(baseURL, dbName string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	u.Path = "/" + dbName
	return u.String()
}
