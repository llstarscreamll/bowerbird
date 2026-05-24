package domain

import "context"

type OwnerData struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	AvatarURL string
}

// Provisioner is responsible for the physical infrastructure provisioning of a new organization.
type Provisioner interface {
	// CreateDatabase creates a new physical database.
	CreateDatabase(ctx context.Context, dbName string) error

	// MigrateDatabase applies the latest business schemas to the organization's database.
	MigrateDatabase(ctx context.Context, dbName string) error

	// SeedOwner inserts the initial user into the tenant DB and assigns the admin role.
	SeedOwner(ctx context.Context, dbName string, owner OwnerData) error
}
