package commands

import (
	"context"
	"fmt"

	"github.com/bowerbird/internal/identity/application/ports"
)

type LeaveTenantCommand struct {
	repo ports.Repository
}

func NewLeaveTenantCommand(repo ports.Repository) *LeaveTenantCommand {
	return &LeaveTenantCommand{repo: repo}
}

func (cmd *LeaveTenantCommand) Execute(ctx context.Context, userID, tenantID string) error {
	err := cmd.repo.RemoveTenantMembership(ctx, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to leave tenant: %w", err)
	}

	dbName, err := cmd.repo.GetTenantDBName(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("left tenant in control plane but failed to resolve tenant db: %w", err)
	}
	if dbName != "" {
		if err := cmd.repo.SoftDeleteTenantUserProfile(ctx, dbName, userID); err != nil {
			return fmt.Errorf("left tenant in control plane but failed to update tenant profile: %w", err)
		}
	}

	return nil
}
