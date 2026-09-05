package queries

import (
	"context"

	"github.com/bowerbird/internal/secrets/application/ports"
	"github.com/bowerbird/internal/secrets/domain"
)

type ListSecretsQuery struct {
	repo ports.SecretRepository
}

func NewListSecretsQuery(repo ports.SecretRepository) *ListSecretsQuery {
	if repo == nil {
		panic("secret repository is required")
	}
	return &ListSecretsQuery{repo: repo}
}

func (q *ListSecretsQuery) Execute(ctx context.Context, purpose string) ([]domain.Secret, error) {
	secrets, err := q.repo.List(ctx, domain.NormalizePurpose(purpose))
	if err != nil {
		return nil, err
	}
	for i := range secrets {
		secrets[i].Ciphertext = nil
	}
	return secrets, nil
}
