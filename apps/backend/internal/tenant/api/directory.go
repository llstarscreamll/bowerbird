package api

import (
	"context"

	"github.com/bowerbird/internal/tenant/domain"
)

const StatusActive = domain.StatusActive

type TenantSummary struct {
	ID     string
	Name   string
	Slug   string
	Status string
}

// Directory is the tenant Open Host Service for control-plane listings.
type Directory interface {
	ListAll(ctx context.Context) ([]TenantSummary, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
}
