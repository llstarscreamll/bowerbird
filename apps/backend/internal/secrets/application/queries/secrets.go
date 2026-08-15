package queries

import (
	"context"

	appErrors "github.com/bowerbird/internal/platform/errors"
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

type ResolveByPurposeQuery struct {
	repo   ports.SecretRepository
	cipher ports.SecretCipher
}

func NewResolveByPurposeQuery(repo ports.SecretRepository, cipher ports.SecretCipher) *ResolveByPurposeQuery {
	if repo == nil {
		panic("secret repository is required")
	}
	if cipher == nil {
		panic("secret cipher is required")
	}
	return &ResolveByPurposeQuery{repo: repo, cipher: cipher}
}

func (q *ResolveByPurposeQuery) Execute(ctx context.Context, purpose string) ([]domain.ResolvedSecret, error) {
	purpose = domain.NormalizePurpose(purpose)
	if purpose == "" {
		return nil, appErrors.New(appErrors.CodeValidation, "purpose is required")
	}

	rows, err := q.repo.ListCiphertextsByPurpose(ctx, purpose)
	if err != nil {
		return nil, err
	}

	out := make([]domain.ResolvedSecret, 0, len(rows))
	for _, row := range rows {
		plaintext, err := q.cipher.Decrypt(row.Ciphertext)
		if err != nil {
			return nil, appErrors.Wrap(err, appErrors.CodeInternal, "failed to decrypt secret")
		}
		out = append(out, domain.ResolvedSecret{
			ID:    row.ID,
			Value: string(plaintext),
		})
	}
	return out, nil
}
