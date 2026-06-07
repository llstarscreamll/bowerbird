package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	connections "github.com/bowerbird/internal/connections/application"
	"github.com/bowerbird/internal/connections/domain"
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
	connectionsService connections.InternalService
	jobDispatcher      SyncAccountJobDispatcher
	logger             *slog.Logger
}

func NewSyncAllAccountsCommand(
	connectionsService connections.InternalService,
	jobDispatcher SyncAccountJobDispatcher,
) *SyncAllAccountsCommand {
	return &SyncAllAccountsCommand{
		connectionsService: connectionsService,
		jobDispatcher:      jobDispatcher,
		logger:             slog.Default(),
	}
}

func (c *SyncAllAccountsCommand) Execute(ctx context.Context, requestorUserID string) error {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	accounts, err := c.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return fmt.Errorf("list active accounts: %w", err)
	}

	if len(accounts) == 0 {
		c.logger.Info("no active accounts found for sync", "tenant_id", tenantID)
		return nil
	}

	fmt.Println("accounts to sync", len(accounts))

	var dispatchErr error
	for _, account := range accounts {
		if account.SharingPolicy == domain.SharingPolicyPrivate && account.OwnerUserID != requestorUserID {
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
		}

		c.logger.Info("dispatched sync job for account", "tenant_id", tenantID, "account_id", account.ID)
	}

	if dispatchErr != nil {
		return dispatchErr
	}

	return nil
}

type InlineSyncAccountJobDispatcher struct {
	command *SyncAccountCommand
}

func NewInlineSyncAccountJobDispatcher(command *SyncAccountCommand) *InlineSyncAccountJobDispatcher {
	return &InlineSyncAccountJobDispatcher{command: command}
}

func (d *InlineSyncAccountJobDispatcher) DispatchSyncAccount(ctx context.Context, job SyncAccountJob) error {
	if d.command == nil {
		return errors.New("sync account command is nil")
	}

	jobCtx := tenant.WithTenantID(ctx, job.TenantID)
	return d.command.Execute(jobCtx, SyncAccountCommandInput{AccountID: job.AccountID})
}
