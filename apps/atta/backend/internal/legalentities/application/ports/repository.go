package ports

import (
	"context"

	"github.com/atta/internal/legalentities/domain"
)

type LegalEntityRepository interface {
	Count(ctx context.Context) (int, error)
	List(ctx context.Context) ([]domain.LegalEntity, error)
	GetByID(ctx context.Context, id string) (*domain.LegalEntity, error)
	Create(ctx context.Context, entity domain.LegalEntity) error
	Update(ctx context.Context, entity domain.LegalEntity) error
}

type RegistrationPublisher interface {
	PublishRegistered(ctx context.Context) error
}
