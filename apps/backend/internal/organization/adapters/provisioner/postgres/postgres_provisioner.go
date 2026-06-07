package postgres

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/bowerbird/internal/organization/domain"
	"github.com/bowerbird/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validDBName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type PostgresProvisioner struct {
	pool          *pgxpool.Pool
	baseURL       string
	migrationsDir string
}

func NewPostgresProvisioner(pool *pgxpool.Pool, baseURL string, migrationsDir string) *PostgresProvisioner {
	return &PostgresProvisioner{
		pool:          pool,
		baseURL:       baseURL,
		migrationsDir: migrationsDir,
	}
}

func (p *PostgresProvisioner) CreateDatabase(ctx context.Context, dbName string) error {
	if !validDBName.MatchString(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	// CREATE DATABASE cannot run inside a transaction block or as a prepared statement easily in pgx
	// We must execute it directly.
	query := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err := p.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("execute create database: %w", err)
	}

	return nil
}

func (p *PostgresProvisioner) MigrateDatabase(ctx context.Context, dbName string) error {
	u, err := url.Parse(p.baseURL)
	if err != nil {
		return fmt.Errorf("parse base url: %w", err)
	}

	// Point the URL to the newly created tenant database
	u.Path = "/" + dbName
	tenantDBURL := u.String()

	err = database.RunMigrations(tenantDBURL, p.migrationsDir)
	if err != nil {
		return fmt.Errorf("run tenant migrations: %w", err)
	}

	return nil
}

func (p *PostgresProvisioner) SeedOwner(ctx context.Context, dbName string, owner domain.OwnerData) error {
	u, err := url.Parse(p.baseURL)
	if err != nil {
		return fmt.Errorf("parse base url: %w", err)
	}

	// Point the URL to the newly created tenant database
	u.Path = "/" + dbName
	tenantDBURL := u.String()

	// Connect specifically to the new tenant's database
	tenantPool, err := pgxpool.New(ctx, tenantDBURL)
	if err != nil {
		return fmt.Errorf("connect to tenant db %s: %w", dbName, err)
	}
	defer tenantPool.Close()

	// Use a transaction to ensure both user creation and role assignment succeed or fail together
	tx, err := tenantPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tenant tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Insert the owner user profile
	// Assuming email is still required in the tenant DB according to the schema
	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, first_name, last_name, picture_url) 
		VALUES ($1, $2, $3, $4, $5)
	`, owner.ID, owner.Email, owner.FirstName, owner.LastName, owner.AvatarURL)
	if err != nil {
		return fmt.Errorf("insert owner user profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tenant tx: %w", err)
	}

	return nil
}
