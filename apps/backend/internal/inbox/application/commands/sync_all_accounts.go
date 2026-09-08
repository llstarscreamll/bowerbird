package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	"github.com/bowerbird/internal/platform/tenant"
)

type SyncAccountJob struct {
	TenantID  string
	AccountID string
	Provider  string
}

type SyncAccountJobDispatcher interface {
	DispatchSyncAccount(ctx context.Context, job SyncAccountJob) error
}

type SyncAllAccountsCommand struct {
	connectionsService connectionsapi.InternalService
	jobDispatcher      SyncAccountJobDispatcher
	logger             *slog.Logger
}

func NewSyncAllAccountsCommand(
	connectionsService connectionsapi.InternalService,
	jobDispatcher SyncAccountJobDispatcher,
) *SyncAllAccountsCommand {
	if connectionsService == nil {
		panic("connections service is required")
	}
	if jobDispatcher == nil {
		panic("sync account job dispatcher is required")
	}
	return &SyncAllAccountsCommand{
		connectionsService: connectionsService,
		jobDispatcher:      jobDispatcher,
		logger:             slog.Default(),
	}
}

func (c *SyncAllAccountsCommand) Execute(ctx context.Context, requestorUserID string) error {
	_, err := c.dispatch(ctx, func(account connectionsapi.ConnectionInfo) bool {
		return account.VisibleTo(requestorUserID)
	})
	return err
}

func (c *SyncAllAccountsCommand) ExecuteScheduled(ctx context.Context) error {
	dispatched, err := c.dispatch(ctx, func(connectionsapi.ConnectionInfo) bool { return true })
	if dispatched > 0 {
		return nil
	}
	return err
}

func (c *SyncAllAccountsCommand) dispatch(ctx context.Context, include func(connectionsapi.ConnectionInfo) bool) (int, error) {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return 0, err
	}

	accounts, err := c.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return 0, fmt.Errorf("list active accounts: %w", err)
	}

	if len(accounts) == 0 {
		c.logger.Info("no active accounts found for sync", "tenant_id", tenantID)
		return 0, nil
	}

	var dispatchErr error
	dispatched := 0
	for _, account := range accounts {
		if !include(account) {
			continue
		}

		err := c.jobDispatcher.DispatchSyncAccount(ctx, SyncAccountJob{
			TenantID:  tenantID,
			AccountID: account.ID,
			Provider:  account.Provider,
		})
		if err != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("dispatch account %s: %w", account.ID, err))
			c.logger.Error("failed to dispatch sync account job", "tenant_slug", tenantID, "account_id", account.ID, "error", err)
			continue
		}

		dispatched++
		c.logger.Info("dispatched sync job for account", "tenant_id", tenantID, "account_id", account.ID)
	}

	return dispatched, dispatchErr
}
