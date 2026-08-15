package ports

import "context"

type FeatureChecker interface {
	Require(ctx context.Context, featureKey string) error
	RequireAny(ctx context.Context, featureKeys ...string) error
	Has(ctx context.Context, featureKey string) (bool, error)
}
