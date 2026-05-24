package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/config"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
)

func main() {
	target := flag.String("target", "all", "Migration target: 'controlplane', 'tenants', or 'all'")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *target == "controlplane" || *target == "all" {
		log.Println("--- Starting Control Plane Migrations ---")
		err := database.RunMigrations(cfg.DatabaseURL, "migrations/controlplane")
		if err != nil {
			log.Fatalf("Control plane migration failed: %v", err)
		}
		log.Println("--- Control Plane Migrations Completed ---")
	}

	if *target == "tenants" || *target == "all" {
		log.Println("--- Starting Tenant Migrations ---")
		if err := runTenantMigrations(ctx, cfg.DatabaseURL); err != nil {
			log.Fatalf("Tenant migrations failed: %v", err)
		}
		log.Println("--- Tenant Migrations Completed ---")
	}
}

func runTenantMigrations(ctx context.Context, controlPlaneURL string) error {
	// Connect to control plane to get list of active tenants
	conn, err := pgx.Connect(ctx, controlPlaneURL)
	if err != nil {
		return fmt.Errorf("connect to control plane: %w", err)
	}
	defer conn.Close(ctx)

	// In a real scenario you would parse the connection string and replace the database name.
	// We'll use a placeholder URL generator for this example.
	// Assuming URL is like: postgres://user:pass@host:5432/bowerbird?sslmode=disable
	// A proper implementation uses standard libraries to parse the URL and substitute the Path.

	rows, err := conn.Query(ctx, "SELECT db_name FROM tenants WHERE status = 'active'")
	if err != nil {
		// If the tenants table doesn't exist yet, that means we haven't run control plane migrations,
		// or there are no tenants. We can just return.
		log.Printf("Could not query tenants (is the control plane migrated?): %v", err)
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

	if len(tenantDBs) == 0 {
		log.Println("No active tenants found to migrate.")
		return nil
	}

	// This is a naive URL replacement. We assume the base URL ends with the control plane DB name or we can extract it.
	// For production, parse `controlPlaneURL` using `net/url` and replace `url.Path`.
	for _, dbName := range tenantDBs {
		log.Printf("Migrating tenant database: %s", dbName)
		tenantURL := buildTenantURL(controlPlaneURL, dbName)

		err := database.RunMigrations(tenantURL, "migrations/tenant")
		if err != nil {
			log.Printf("ERROR migrating tenant %s: %v", dbName, err)
			// Decide whether to fail fast or continue with other tenants. Usually, continue and report at the end.
		}
	}

	return nil
}

func buildTenantURL(baseURL string, dbName string) string {
	// Parse the connection string
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL // Fallback, though likely to fail
	}

	// Replace the path (database name)
	u.Path = "/" + dbName
	return u.String()
}
