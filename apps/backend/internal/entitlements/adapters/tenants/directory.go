package tenants

import (
	"context"

	"github.com/bowerbird/internal/entitlements/application"
	tenantapi "github.com/bowerbird/internal/tenant/api"
)

type Directory struct {
	source tenantapi.Directory
}

func NewDirectory(source tenantapi.Directory) *Directory {
	if source == nil {
		panic("tenant directory is required")
	}
	return &Directory{source: source}
}

func (d *Directory) ListTenants(ctx context.Context) ([]application.TenantSummary, error) {
	orgs, err := d.source.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]application.TenantSummary, 0, len(orgs))
	for _, org := range orgs {
		items = append(items, application.TenantSummary{
			ID:     org.ID,
			Name:   org.Name,
			Slug:   org.Slug,
			Status: org.Status,
		})
	}
	return items, nil
}

func (d *Directory) TenantExists(ctx context.Context, tenantID string) (bool, error) {
	return d.source.ExistsByID(ctx, tenantID)
}
