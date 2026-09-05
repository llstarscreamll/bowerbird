package commands

import (
	"context"
	"strings"
	"time"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/secrets/application/ports"
	"github.com/bowerbird/internal/secrets/domain"
)

type CreateSecretInput struct {
	Purpose     string
	Kind        string
	Label       string
	Description string
	Value       string
	ActorUserID string
}

type CreateSecretCommand struct {
	repo   ports.SecretRepository
	cipher ports.SecretCipher
}

func NewCreateSecretCommand(repo ports.SecretRepository, cipher ports.SecretCipher) *CreateSecretCommand {
	if repo == nil {
		panic("secret repository is required")
	}
	if cipher == nil {
		panic("secret cipher is required")
	}
	return &CreateSecretCommand{repo: repo, cipher: cipher}
}

func (cmd *CreateSecretCommand) Execute(ctx context.Context, input CreateSecretInput) (*domain.Secret, error) {
	purpose := domain.NormalizePurpose(input.Purpose)
	label := domain.NormalizeLabel(input.Label)
	value := strings.TrimSpace(input.Value)

	if purpose == "" {
		return nil, appErrors.New(appErrors.CodeValidation, "purpose is required")
	}
	if !domain.IsKnownPurpose(purpose) {
		return nil, appErrors.New(appErrors.CodeValidation, "unknown purpose")
	}
	if label == "" {
		return nil, appErrors.New(appErrors.CodeValidation, "label is required")
	}
	if value == "" {
		return nil, appErrors.New(appErrors.CodeValidation, "value is required")
	}
	if input.ActorUserID == "" {
		return nil, appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = domain.DefaultKindForPurpose(purpose)
	}

	ciphertext, err := cmd.cipher.Encrypt([]byte(value))
	if err != nil {
		return nil, appErrors.Wrap(err, appErrors.CodeInternal, "failed to encrypt secret")
	}

	now := time.Now().UTC()
	secret := domain.Secret{
		ID:          id.NewULID(),
		Purpose:     purpose,
		Kind:        kind,
		Label:       label,
		Description: strings.TrimSpace(input.Description),
		Ciphertext:  ciphertext,
		Version:     1,
		KeyID:       domain.KeyIDLocalAESV1,
		CreatedBy:   input.ActorUserID,
		UpdatedBy:   input.ActorUserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := cmd.repo.Create(ctx, secret); err != nil {
		return nil, err
	}

	secretID := secret.ID
	_ = cmd.repo.AppendAudit(ctx, domain.AuditEvent{
		ID:          id.NewULID(),
		SecretID:    &secretID,
		Purpose:     purpose,
		Action:      domain.AuditActionCreate,
		ActorUserID: input.ActorUserID,
		CreatedAt:   now,
	})

	secret.Ciphertext = nil
	return &secret, nil
}
