package ports

import (
	"context"

	"github.com/bowerbird/internal/connections/domain"
)

type ConnectionRepository interface {
	ListActive(ctx context.Context) ([]*domain.Connection, error)
	GetByID(ctx context.Context, id string) (*domain.Connection, error)
	Upsert(ctx context.Context, conn *domain.Connection) error
}
