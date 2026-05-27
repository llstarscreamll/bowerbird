package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

var ErrActiveConnectionNotFound = errors.New("active connection not found")

type TriggerSyncUseCase struct {
	repo               domain.Repository
	connectionsService application.InternalService
	syncAccounts       *SyncAccountsUseCase
}

func NewTriggerSyncUseCase(repo domain.Repository, connectionsService application.InternalService, syncAccounts *SyncAccountsUseCase) *TriggerSyncUseCase {
	return &TriggerSyncUseCase{
		repo:               repo,
		connectionsService: connectionsService,
		syncAccounts:       syncAccounts,
	}
}

func (u *TriggerSyncUseCase) Execute(ctx context.Context, accountID *string) error {
	tenantSlug, err := tenant.TenantSlugFromContext(ctx)
	if err != nil {
		return err
	}

	accounts, err := u.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return fmt.Errorf("list active connections: %w", err)
	}

	// Filter accounts if accountID is specified
	var targetAccounts []application.ConnectionInfo
	if accountID != nil && *accountID != "" && *accountID != "all" {
		for _, acc := range accounts {
			if acc.ID == *accountID {
				targetAccounts = append(targetAccounts, acc)
				break
			}
		}
		if len(targetAccounts) == 0 {
			return fmt.Errorf("%w: %s", ErrActiveConnectionNotFound, *accountID)
		}
	} else {
		targetAccounts = accounts
	}

	// Trigger sync for each target account in background
	for _, acc := range targetAccounts {
		// Only trigger if not already syncing
		cursor, err := u.repo.GetSyncCursor(ctx, acc.ID)
		if err != nil {
			slog.Error("failed to get sync cursor", "account_id", acc.ID, "error", err)
			continue
		}

		if cursor == nil {
			cursor = &domain.InboxSyncCursor{
				ConnectionID: acc.ID,
				Status:       domain.InboxSyncStatusIdle,
			}
		}

		if cursor.Status == domain.InboxSyncStatusSyncing {
			// Already syncing, skip
			continue
		}

		// Mark as syncing before launching background job
		cursor.Status = domain.InboxSyncStatusSyncing
		if err := u.repo.UpsertSyncCursor(ctx, cursor); err != nil {
			slog.Error("failed to mark account as syncing", "account_id", acc.ID, "error", err)
			continue
		}

		// Run in background (context without cancellation from request)
		go func(acc application.ConnectionInfo, c *domain.InboxSyncCursor) {
			bgCtx := tenant.WithTenantSlug(context.Background(), tenantSlug)
			result := &SyncAccountsResult{}
			err := u.syncAccounts.SyncSingleAccountWithResult(bgCtx, tenantSlug, acc, result)
			if err != nil {
				slog.Error("background sync failed", "account_id", acc.ID, "error", err)
				// syncAccounts.SyncSingleAccountWithResult already updates cursor on error
			}
		}(acc, cursor)
	}

	return nil
}
