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

type UpdateSecretInput struct {
	ID          string
	Label       *string
	Description *string
	Value       *string
	ActorUserID string
}

type UpdateSecretCommand struct {
	repo   ports.SecretRepository
	cipher ports.SecretCipher
}

func NewUpdateSecretCommand(repo ports.SecretRepository, cipher ports.SecretCipher) *UpdateSecretCommand {
	if repo == nil {
		panic("secret repository is required")
	}
	if cipher == nil {
		panic("secret cipher is required")
	}
	return &UpdateSecretCommand{repo: repo, cipher: cipher}
}

func (cmd *UpdateSecretCommand) Execute(ctx context.Context, input UpdateSecretInput) (*domain.Secret, error) {
	if input.ID == "" {
		return nil, appErrors.New(appErrors.CodeValidation, "secret id is required")
	}
	if input.ActorUserID == "" {
		return nil, appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	existing, err := cmd.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "secret not found")
	}

	rotated := false
	if input.Label != nil {
		label := domain.NormalizeLabel(*input.Label)
		if label == "" {
			return nil, appErrors.New(appErrors.CodeValidation, "label is required")
		}
		existing.Label = label
	}
	if input.Description != nil {
		existing.Description = strings.TrimSpace(*input.Description)
	}
	if input.Value != nil {
		value := strings.TrimSpace(*input.Value)
		if value == "" {
			return nil, appErrors.New(appErrors.CodeValidation, "value is required")
		}
		ciphertext, err := cmd.cipher.Encrypt([]byte(value))
		if err != nil {
			return nil, appErrors.Wrap(err, appErrors.CodeInternal, "failed to encrypt secret")
		}
		existing.Ciphertext = ciphertext
		existing.Version++
		rotated = true
	}

	existing.UpdatedBy = input.ActorUserID
	existing.UpdatedAt = time.Now().UTC()

	if err := cmd.repo.Update(ctx, *existing); err != nil {
		return nil, err
	}

	if rotated || input.Label != nil || input.Description != nil {
		secretID := existing.ID
		auditAction := domain.AuditActionUpdate
		if rotated {
			auditAction = domain.AuditActionRotate
		}
		_ = cmd.repo.AppendAudit(ctx, domain.AuditEvent{
			ID:          id.NewULID(),
			SecretID:    &secretID,
			Purpose:     existing.Purpose,
			Action:      auditAction,
			ActorUserID: input.ActorUserID,
			CreatedAt:   existing.UpdatedAt,
		})
	}

	existing.Ciphertext = nil
	return existing, nil
}
