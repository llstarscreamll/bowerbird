package queries

import (
	"context"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/secrets/application/ports"
	"github.com/bowerbird/internal/secrets/domain"
)

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
