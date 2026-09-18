package application

import "context"

type OperatorDirectory interface {
	IsPlatformOperator(ctx context.Context, userID string) (bool, error)
}

type TenantSummary struct {
	ID     string
	Name   string
	Slug   string
	Status string
}

type TenantDirectory interface {
	ListTenants(ctx context.Context) ([]TenantSummary, error)
	TenantExists(ctx context.Context, tenantID string) (bool, error)
}
