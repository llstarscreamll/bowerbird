package tenants

import (
	"context"

	"github.com/bowerbird/internal/entitlements/application"
	orgdomain "github.com/bowerbird/internal/organization/domain"
)

type Source interface {
	ListAll(ctx context.Context) ([]orgdomain.Organization, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
}

type Directory struct {
	source Source
}

func NewDirectory(source Source) *Directory {
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
