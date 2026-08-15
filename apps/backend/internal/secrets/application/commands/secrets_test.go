package commands_test

import (
	"context"
	"testing"
	"time"

	platformCrypto "github.com/bowerbird/internal/platform/crypto"
	"github.com/bowerbird/internal/secrets/application/commands"
	"github.com/bowerbird/internal/secrets/application/queries"
	"github.com/bowerbird/internal/secrets/domain"
	"github.com/stretchr/testify/require"
)

type memorySecretRepo struct {
	byID map[string]domain.Secret
}

func newMemorySecretRepo() *memorySecretRepo {
	return &memorySecretRepo{byID: map[string]domain.Secret{}}
}

func (r *memorySecretRepo) List(ctx context.Context, purpose string) ([]domain.Secret, error) {
	out := make([]domain.Secret, 0)
	for _, secret := range r.byID {
		if purpose == "" || secret.Purpose == purpose {
			copySecret := secret
			copySecret.Ciphertext = nil
			out = append(out, copySecret)
		}
	}
	return out, nil
}

func (r *memorySecretRepo) GetByID(ctx context.Context, id string) (*domain.Secret, error) {
	secret, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	copySecret := secret
	return &copySecret, nil
}

func (r *memorySecretRepo) Create(ctx context.Context, secret domain.Secret) error {
	r.byID[secret.ID] = secret
	return nil
}

func (r *memorySecretRepo) Update(ctx context.Context, secret domain.Secret) error {
	r.byID[secret.ID] = secret
	return nil
}

func (r *memorySecretRepo) Delete(ctx context.Context, id string) error {
	delete(r.byID, id)
	return nil
}

func (r *memorySecretRepo) ListCiphertextsByPurpose(ctx context.Context, purpose string) ([]domain.Secret, error) {
	out := make([]domain.Secret, 0)
	for _, secret := range r.byID {
		if secret.Purpose == purpose {
			out = append(out, secret)
		}
	}
	return out, nil
}

func (r *memorySecretRepo) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	secret, ok := r.byID[id]
	if !ok {
		return nil
	}
	secret.LastUsedAt = &usedAt
	r.byID[id] = secret
	return nil
}

func (r *memorySecretRepo) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	return nil
}

func TestCreateAndResolveSecret(t *testing.T) {
	cipher, err := platformCrypto.NewAESCipherFromBase64Key("Ym93ZXJiaXJkLWxvY2FsLXNlY3JldHMta2V5LTMyYiE=")
	require.NoError(t, err)

	repo := newMemorySecretRepo()
	create := commands.NewCreateSecretCommand(repo, cipher)
	resolve := queries.NewResolveByPurposeQuery(repo, cipher)

	created, err := create.Execute(context.Background(), commands.CreateSecretInput{
		Purpose:     domain.PurposeInvoicingDocumentPassword,
		Label:       "NIT Acme",
		Value:       "900123456",
		ActorUserID: "user-1",
	})
	require.NoError(t, err)
	require.Equal(t, 1, created.Version)
	require.Nil(t, created.Ciphertext)

	resolved, err := resolve.Execute(context.Background(), domain.PurposeInvoicingDocumentPassword)
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	require.Equal(t, "900123456", resolved[0].Value)

	update := commands.NewUpdateSecretCommand(repo, cipher)
	value := "900999888"
	updated, err := update.Execute(context.Background(), commands.UpdateSecretInput{
		ID:          created.ID,
		Value:       &value,
		ActorUserID: "user-1",
	})
	require.NoError(t, err)
	require.Equal(t, 2, updated.Version)

	resolved, err = resolve.Execute(context.Background(), domain.PurposeInvoicingDocumentPassword)
	require.NoError(t, err)
	require.Equal(t, "900999888", resolved[0].Value)
}

func TestCreateRejectsUnknownPurpose(t *testing.T) {
	cipher, err := platformCrypto.NewAESCipherFromBase64Key("Ym93ZXJiaXJkLWxvY2FsLXNlY3JldHMta2V5LTMyYiE=")
	require.NoError(t, err)
	create := commands.NewCreateSecretCommand(newMemorySecretRepo(), cipher)
	_, err = create.Execute(context.Background(), commands.CreateSecretInput{
		Purpose:     "unknown.purpose",
		Label:       "x",
		Value:       "y",
		ActorUserID: "user-1",
	})
	require.Error(t, err)
}
