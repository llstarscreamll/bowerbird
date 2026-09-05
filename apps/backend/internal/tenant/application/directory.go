package application

import (
	"context"

	"github.com/bowerbird/internal/tenant/api"
	"github.com/bowerbird/internal/tenant/application/ports"
)

type tenantDirectory struct {
	repo ports.TenantRepository
}

func newTenantDirectory(repo ports.TenantRepository) api.Directory {
	return &tenantDirectory{repo: repo}
}

func (d *tenantDirectory) ListAll(ctx context.Context) ([]api.TenantSummary, error) {
	orgs, err := d.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]api.TenantSummary, 0, len(orgs))
	for _, org := range orgs {
		items = append(items, api.TenantSummary{
			ID:     org.ID,
			Name:   org.Name,
			Slug:   org.Slug,
			Status: org.Status,
		})
	}
	return items, nil
}

func (d *tenantDirectory) ExistsByID(ctx context.Context, id string) (bool, error) {
	return d.repo.ExistsByID(ctx, id)
}

var _ api.Directory = (*tenantDirectory)(nil)
