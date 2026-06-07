package ports

import (
	"context"

	"github.com/bowerbird/internal/organization/domain"
)

type Provisioner interface {
	CreateDatabase(ctx context.Context, dbName string) error
	MigrateDatabase(ctx context.Context, dbName string) error
	SeedOwner(ctx context.Context, dbName string, owner domain.OwnerData) error
}
