package application

import (
	"context"
	"time"

	"github.com/bowerbird/internal/entitlements/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/tenant"
)

type Service struct {
	repo domain.Repository
	now  func() time.Time
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

type EffectiveAccess struct {
	Features []string `json:"features"`
}

func (s *Service) canonicalTenantID(ctx context.Context, tenantID string) (string, error) {
	return s.repo.ResolveTenantID(ctx, tenantID)
}

func (s *Service) Evaluate(ctx context.Context, tenantID string) (EffectiveAccess, error) {
	canonicalID, err := s.canonicalTenantID(ctx, tenantID)
	if err != nil {
		return EffectiveAccess{}, err
	}
	grants, err := s.repo.ListByTenant(ctx, canonicalID)
	if err != nil {
		return EffectiveAccess{}, err
	}

	now := s.now().UTC()
	features := make([]string, 0)
	for _, grant := range grants {
		if grant.IsEffective(now) {
			features = append(features, grant.FeatureKey)
		}
	}
	return EffectiveAccess{Features: features}, nil
}

func (s *Service) Has(ctx context.Context, featureKey string) (bool, error) {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return false, err
	}
	access, err := s.Evaluate(ctx, tenantID)
	if err != nil {
		return false, err
	}
	for _, key := range access.Features {
		if key == featureKey {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) Require(ctx context.Context, featureKey string) error {
	ok, err := s.Has(ctx, featureKey)
	if err != nil {
		return err
	}
	if !ok {
		return forbiddenFeature(featureKey)
	}
	return nil
}

func (s *Service) RequireAny(ctx context.Context, featureKeys ...string) error {
	for _, key := range featureKeys {
		ok, err := s.Has(ctx, key)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	featureKey := ""
	if len(featureKeys) > 0 {
		featureKey = featureKeys[0]
	}
	return forbiddenFeature(featureKey)
}

func (s *Service) Grants(ctx context.Context, tenantID string) ([]domain.Entitlement, error) {
	canonicalID, err := s.canonicalTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListByTenant(ctx, canonicalID)
}

func (s *Service) ApplyDefaultPack(ctx context.Context, tenantID, actorUserID string) error {
	canonicalID, err := s.canonicalTenantID(ctx, tenantID)
	if err != nil {
		return err
	}
	return s.grantKeys(ctx, canonicalID, actorUserID, domain.DefaultPackFeatureKeys(), domain.StatusActive, domain.SourceManual, nil)
}

type SetAccessInput struct {
	Product string
	Feature string
	Enabled bool
	EndsAt  *time.Time
	ActorID string
	Source  string
	AsTrial bool
}

func (s *Service) SetAccess(ctx context.Context, tenantID string, input SetAccessInput) error {
	canonicalID, err := s.canonicalTenantID(ctx, tenantID)
	if err != nil {
		return err
	}
	tenantID = canonicalID
	source := input.Source
	if source == "" {
		source = domain.SourceManual
	}
	status := domain.StatusActive
	if input.AsTrial || input.EndsAt != nil && source == domain.SourceTrial {
		status = domain.StatusTrial
		source = domain.SourceTrial
	}
	if input.EndsAt != nil {
		source = domain.SourceTrial
		status = domain.StatusTrial
	}

	if input.Product != "" {
		if !domain.ProductExists(input.Product) {
			return domain.ErrUnknownProduct
		}
		if !input.Enabled {
			return s.revokeKeys(ctx, tenantID, domain.AllFeatureKeys(input.Product))
		}
		return s.grantKeys(ctx, tenantID, input.ActorID, domain.RequiredFeatureKeys(input.Product), status, source, input.EndsAt)
	}

	if input.Feature == "" || !domain.FeatureExists(input.Feature) {
		return domain.ErrUnknownFeature
	}
	if !input.Enabled {
		return s.repo.Delete(ctx, tenantID, input.Feature)
	}
	return s.grantKeys(ctx, tenantID, input.ActorID, []string{input.Feature}, status, source, input.EndsAt)
}

func (s *Service) grantKeys(ctx context.Context, tenantID, actorUserID string, keys []string, status, source string, endsAt *time.Time) error {
	now := s.now().UTC()
	for _, key := range keys {
		ent := domain.Entitlement{
			ID:         id.NewULID(),
			TenantID:   tenantID,
			FeatureKey: key,
			Status:     status,
			Source:     source,
			StartsAt:   now,
			EndsAt:     endsAt,
			CreatedBy:  actorUserID,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.repo.Upsert(ctx, ent); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) revokeKeys(ctx context.Context, tenantID string, keys []string) error {
	for _, key := range keys {
		if err := s.repo.Delete(ctx, tenantID, key); err != nil {
			return err
		}
	}
	return nil
}

func forbiddenFeature(featureKey string) error {
	return appErrors.New(appErrors.CodeForbidden, "feature is not available").WithMeta("feature_key", featureKey)
}
