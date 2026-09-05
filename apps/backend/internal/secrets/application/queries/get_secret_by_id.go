package queries

import (
	"context"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/secrets/application/ports"
	"github.com/bowerbird/internal/secrets/domain"
)

type GetSecretByIDQuery struct {
	repo ports.SecretRepository
}

func NewGetSecretByIDQuery(repo ports.SecretRepository) *GetSecretByIDQuery {
	if repo == nil {
		panic("secret repository is required")
	}
	return &GetSecretByIDQuery{repo: repo}
}

func (q *GetSecretByIDQuery) Execute(ctx context.Context, id string) (*domain.Secret, error) {
	if id == "" {
		return nil, appErrors.New(appErrors.CodeValidation, "secret id is required")
	}
	secret, err := q.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "secret not found")
	}
	secret.Ciphertext = nil
	return secret, nil
}
