package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bowerbird/internal/secrets/application/ports"
)

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
