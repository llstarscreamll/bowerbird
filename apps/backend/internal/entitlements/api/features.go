package api

import (
	"context"

	"github.com/bowerbird/internal/entitlements/domain"
)

const (
	FeatureInvoicingWorkspace        = domain.FeatureInvoicingWorkspace
	FeatureInvoicingCaptureFromEmail = domain.FeatureInvoicingCaptureFromEmail
	FeatureMailInbox                 = domain.FeatureMailInbox
	FeatureMailOrganize              = domain.FeatureMailOrganize
	FeatureMailSend                  = domain.FeatureMailSend
)

// Features is the entitlements Open Host Service for feature gates.
type Features interface {
	Require(ctx context.Context, featureKey string) error
	RequireAny(ctx context.Context, featureKeys ...string) error
	Has(ctx context.Context, featureKey string) (bool, error)
}

// DefaultPack grants the default entitlement pack when a tenant is created.
type DefaultPack interface {
	ApplyDefaultPack(ctx context.Context, tenantID, actorUserID string) error
}
