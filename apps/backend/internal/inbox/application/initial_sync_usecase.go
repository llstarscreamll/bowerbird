package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	connectionsapp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

type InitialSyncUseCase struct {
	repo               domain.Repository
	connectionsService connectionsapp.InternalService
	syncAccounts       *SyncAccountsUseCase
	logger             *slog.Logger
	now                func() time.Time
}

func NewInitialSyncUseCase(
	repo domain.Repository,
	connectionsService connectionsapp.InternalService,
	syncAccounts *SyncAccountsUseCase,
) *InitialSyncUseCase {
	return &InitialSyncUseCase{
		repo:               repo,
		connectionsService: connectionsService,
		syncAccounts:       syncAccounts,
		logger:             slog.Default(),
		now:                time.Now,
	}
}

func (u *InitialSyncUseCase) Process(ctx context.Context, tenantSlug string, connectionID string, provider string) error {
	u.logger.Info("starting initial sync for connection", "connection_id", connectionID, "tenant_slug", tenantSlug)

	exists, err := u.hasActiveConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to validate active connection: %w", err)
	}
	if !exists {
		u.logger.Warn("skipping initial sync for missing or inactive connection", "connection_id", connectionID, "tenant_slug", tenantSlug)
		return nil
	}

	// Create or update cursor to start from 2 months ago
	twoMonthsAgo := u.now().UTC().AddDate(0, -2, 0)
	cursor, err := u.repo.GetSyncCursor(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("failed to get sync cursor: %w", err)
	}

	if cursor == nil {
		cursor = &domain.InboxSyncCursor{
			ConnectionID: connectionID,
			Status:       domain.InboxSyncStatusIdle,
			LastSyncedAt: &twoMonthsAgo,
		}
	} else {
		cursor.LastSyncedAt = &twoMonthsAgo
	}

	err = u.repo.UpsertSyncCursor(ctx, cursor)
	if err != nil {
		if isForeignKeyViolation(err) {
			u.logger.Warn("skipping initial sync because connection no longer exists", "connection_id", connectionID, "tenant_slug", tenantSlug)
			return nil
		}
		return fmt.Errorf("failed to upsert initial sync cursor: %w", err)
	}

	if u.syncAccounts != nil {
		u.logger.Info("syncing single account", "connection_id", connectionID)
		_, err = u.syncAccounts.SyncSingleAccount(ctx, connectionID)
		if err != nil {
			u.logger.Error("sync single account failed", "connection_id", connectionID, "error", err)
			return fmt.Errorf("sync single account failed: %w", err)
		}
	}

	return nil
}

func (u *InitialSyncUseCase) hasActiveConnection(ctx context.Context, connectionID string) (bool, error) {
	if u.connectionsService == nil {
		return true, nil
	}

	connections, err := u.connectionsService.GetActiveConnections(ctx)
	if err != nil {
		return false, err
	}

	for _, connection := range connections {
		if connection.ID == connectionID {
			return true, nil
		}
	}

	return false, nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23503"
}
