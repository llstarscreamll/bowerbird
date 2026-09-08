package commands

import (
	"context"
	"errors"
	"log/slog"

	entitlementsapi "github.com/bowerbird/internal/entitlements/api"
	"github.com/bowerbird/internal/inbox/application/ports"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/tenant"
)

type PlatformSyncAllAccountsCommand struct {
	perTenant *SyncAllAccountsCommand
	features  entitlementsapi.Features
	tenants   ports.ActiveTenantLister
	logger    *slog.Logger
}

func NewPlatformSyncAllAccountsCommand(
	perTenant *SyncAllAccountsCommand,
	features entitlementsapi.Features,
	tenants ports.ActiveTenantLister,
) *PlatformSyncAllAccountsCommand {
	if perTenant == nil {
		panic("sync all accounts command is required")
	}
	if features == nil {
		panic("feature checker is required")
	}
	if tenants == nil {
		panic("tenant lister is required")
	}
	return &PlatformSyncAllAccountsCommand{
		perTenant: perTenant,
		features:  features,
		tenants:   tenants,
		logger:    slog.Default(),
	}
}

func (c *PlatformSyncAllAccountsCommand) Execute(ctx context.Context) error {
	slugs, err := c.tenants.ListActiveTenantSlugs(ctx)
	if err != nil {
		return err
	}
	if len(slugs) == 0 {
		c.logger.Info("inbox sync-all: no active tenants")
		return nil
	}

	var runErr error
	ok := 0
	for _, slug := range slugs {
		if err := c.syncTenant(ctx, slug); err != nil {
			c.logger.Error("inbox sync-all tenant failed", "tenant_slug", slug, "error", err)
			runErr = errors.Join(runErr, err)
			continue
		}
		ok++
	}
	if ok > 0 {
		return nil
	}
	return runErr
}

func (c *PlatformSyncAllAccountsCommand) syncTenant(ctx context.Context, slug string) error {
	tenantCtx := tenant.WithTenantID(ctx, slug)
	if err := c.features.RequireAny(tenantCtx, entitlementsapi.FeatureMailInbox, entitlementsapi.FeatureInvoicingCaptureFromEmail); err != nil {
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) && appErr.Code == appErrors.CodeForbidden {
			c.logger.Info("skipping inbox sync-all: feature not available", "tenant_slug", slug)
			return nil
		}
		return err
	}
	return c.perTenant.ExecuteScheduled(tenantCtx)
}
