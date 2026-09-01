package ports

import (
	"context"

	"github.com/bowerbird/internal/parties/domain"
)

type PartyRepository interface {
	Create(ctx context.Context, party domain.Party) error
	Update(ctx context.Context, party domain.Party) error
	GetByID(ctx context.Context, id string) (*domain.Party, error)
	GetByTaxID(ctx context.Context, taxID string) (*domain.Party, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Party, error)
}

type ListFilter struct {
	Role           string
	Search         string
	CreationSource string
}
