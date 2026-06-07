package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/bowerbird/internal/identity/application/ports"
)

type DeleteAccountCommand struct {
	repo ports.Repository
}

func NewDeleteAccountCommand(repo ports.Repository) *DeleteAccountCommand {
	return &DeleteAccountCommand{repo: repo}
}

func (cmd *DeleteAccountCommand) Execute(ctx context.Context, userID string) error {
	memberships, membershipsErr := cmd.repo.FindTenantMemberships(ctx, userID)

	err := cmd.repo.SoftDeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	var syncErr error
	if membershipsErr != nil {
		syncErr = errors.Join(syncErr, fmt.Errorf("list tenant memberships for cleanup: %w", membershipsErr))
	}
	for _, m := range memberships {
		dbName, err := cmd.repo.GetTenantDBName(ctx, m.TenantID)
		if err != nil {
			syncErr = errors.Join(syncErr, fmt.Errorf("resolve tenant db for %s: %w", m.TenantID, err))
			continue
		}
		if dbName != "" {
			if err := cmd.repo.SoftDeleteTenantUserProfile(ctx, dbName, userID); err != nil {
				syncErr = errors.Join(syncErr, fmt.Errorf("soft delete tenant profile for %s: %w", m.TenantID, err))
			}
		}
	}
	if syncErr != nil {
		return fmt.Errorf("account deleted in control plane with partial tenant profile cleanup: %w", syncErr)
	}

	return nil
}
