package commands

import (
	"context"
	"fmt"
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

type DeleteSecretCommand struct {
	repo ports.SecretRepository
}

func NewDeleteSecretCommand(repo ports.SecretRepository) *DeleteSecretCommand {
	if repo == nil {
		panic("secret repository is required")
	}
	return &DeleteSecretCommand{repo: repo}
}

func (cmd *DeleteSecretCommand) Execute(ctx context.Context, idValue, actorUserID string) error {
	if idValue == "" {
		return appErrors.New(appErrors.CodeValidation, "secret id is required")
	}
	if actorUserID == "" {
		return appErrors.New(appErrors.CodeUnauthorized, "unauthorized")
	}

	existing, err := cmd.repo.GetByID(ctx, idValue)
	if err != nil {
		return err
	}
	if existing == nil {
		return appErrors.New(appErrors.CodeNotFound, "secret not found")
	}

	if err := cmd.repo.Delete(ctx, idValue); err != nil {
		return err
	}

	secretID := existing.ID
	_ = cmd.repo.AppendAudit(ctx, domain.AuditEvent{
		ID:          id.NewULID(),
		SecretID:    &secretID,
		Purpose:     existing.Purpose,
		Action:      domain.AuditActionDelete,
		ActorUserID: actorUserID,
		CreatedAt:   time.Now().UTC(),
	})
	return nil
}

type MarkSecretUsedCommand struct {
	repo ports.SecretRepository
}

func NewMarkSecretUsedCommand(repo ports.SecretRepository) *MarkSecretUsedCommand {
	if repo == nil {
		panic("secret repository is required")
	}
	return &MarkSecretUsedCommand{repo: repo}
}

func (cmd *MarkSecretUsedCommand) Execute(ctx context.Context, secretID string) error {
	if secretID == "" {
		return fmt.Errorf("secret id is required")
	}
	return cmd.repo.MarkUsed(ctx, secretID, time.Now().UTC())
}
