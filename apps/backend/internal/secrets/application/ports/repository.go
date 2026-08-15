package ports

import (
	"context"
	"time"

	"github.com/bowerbird/internal/secrets/domain"
)

type SecretCipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type SecretRepository interface {
	List(ctx context.Context, purpose string) ([]domain.Secret, error)
	GetByID(ctx context.Context, id string) (*domain.Secret, error)
	Create(ctx context.Context, secret domain.Secret) error
	Update(ctx context.Context, secret domain.Secret) error
	Delete(ctx context.Context, id string) error
	ListCiphertextsByPurpose(ctx context.Context, purpose string) ([]domain.Secret, error)
	MarkUsed(ctx context.Context, id string, usedAt time.Time) error
	AppendAudit(ctx context.Context, event domain.AuditEvent) error
}
