package commands

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/platform/tenant"
)

const importPurgeBatchSize = 1000

type PurgeStaleCatalogImportsCommand struct {
	imports ports.ImportRepository
	now     func() time.Time
}

func NewPurgeStaleCatalogImportsCommand(imports ports.ImportRepository) *PurgeStaleCatalogImportsCommand {
	if imports == nil {
		panic("import repository is required")
	}
	return &PurgeStaleCatalogImportsCommand{imports: imports, now: time.Now}
}

func (cmd *PurgeStaleCatalogImportsCommand) Execute(ctx context.Context) (int64, error) {
	cutoff := cmd.now().UTC().AddDate(-1, 0, 0)
	return cmd.imports.PurgeStaleImports(ctx, cutoff, importPurgeBatchSize)
}

type PlatformPurgeStaleCatalogImportsCommand struct {
	perTenant *PurgeStaleCatalogImportsCommand
	tenants   ports.ActiveTenantLister
	logger    *slog.Logger
}

func NewPlatformPurgeStaleCatalogImportsCommand(
	perTenant *PurgeStaleCatalogImportsCommand,
	tenants ports.ActiveTenantLister,
) *PlatformPurgeStaleCatalogImportsCommand {
	if perTenant == nil {
		panic("purge command is required")
	}
	if tenants == nil {
		panic("tenant lister is required")
	}
	return &PlatformPurgeStaleCatalogImportsCommand{
		perTenant: perTenant,
		tenants:   tenants,
		logger:    slog.Default(),
	}
}

func (c *PlatformPurgeStaleCatalogImportsCommand) Execute(ctx context.Context) error {
	slugs, err := c.tenants.ListActiveTenantSlugs(ctx)
	if err != nil {
		return err
	}
	if len(slugs) == 0 {
		c.logger.Info("catalog import purge: no active tenants")
		return nil
	}
	var runErr error
	ok := 0
	for _, slug := range slugs {
		if _, err := c.perTenant.Execute(tenant.WithTenantID(ctx, slug)); err != nil {
			c.logger.Error("catalog import purge tenant failed", "tenant_slug", slug, "error", err)
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
