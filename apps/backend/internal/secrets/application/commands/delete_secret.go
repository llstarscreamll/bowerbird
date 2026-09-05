package commands

import (
	"context"
	"time"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/secrets/application/ports"
	"github.com/bowerbird/internal/secrets/domain"
)

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
	return cmd.repo.AppendAudit(ctx, domain.AuditEvent{
		ID:          id.NewULID(),
		SecretID:    &secretID,
		Purpose:     existing.Purpose,
		Action:      domain.AuditActionDelete,
		ActorUserID: actorUserID,
		CreatedAt:   time.Now().UTC(),
	})
}
