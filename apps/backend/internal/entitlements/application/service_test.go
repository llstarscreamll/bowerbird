package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/bowerbird/internal/entitlements/application"
	"github.com/bowerbird/internal/entitlements/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	byTenant map[string][]domain.Entitlement
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byTenant: map[string][]domain.Entitlement{}}
}

func (r *fakeRepo) ResolveTenantID(ctx context.Context, idOrSlug string) (string, error) {
	return idOrSlug, nil
}

func (r *fakeRepo) ListByTenant(ctx context.Context, tenantID string) ([]domain.Entitlement, error) {
	return r.byTenant[tenantID], nil
}

func (r *fakeRepo) Upsert(ctx context.Context, entitlement domain.Entitlement) error {
	items := r.byTenant[entitlement.TenantID]
	for i, existing := range items {
		if existing.FeatureKey == entitlement.FeatureKey {
			items[i] = entitlement
			r.byTenant[entitlement.TenantID] = items
			return nil
		}
	}
	r.byTenant[entitlement.TenantID] = append(items, entitlement)
	return nil
}

func (r *fakeRepo) Delete(ctx context.Context, tenantID, featureKey string) error {
	items := r.byTenant[tenantID]
	kept := items[:0]
	for _, item := range items {
		if item.FeatureKey != featureKey {
			kept = append(kept, item)
		}
	}
	r.byTenant[tenantID] = kept
	return nil
}

func (r *fakeRepo) ReplaceTenantFeatures(ctx context.Context, tenantID string, entitlements []domain.Entitlement) error {
	r.byTenant[tenantID] = entitlements
	return nil
}

func TestEvaluate_TrialExpiredDoesNotGrantMail(t *testing.T) {
	now := time.Now().UTC()
	expired := now.Add(-time.Hour)
	repo := newFakeRepo()
	repo.byTenant["t1"] = []domain.Entitlement{
		{FeatureKey: domain.FeatureMailInbox, Status: domain.StatusTrial, StartsAt: now.Add(-48 * time.Hour), EndsAt: &expired},
		{FeatureKey: domain.FeatureInvoicingCaptureFromEmail, Status: domain.StatusActive, StartsAt: now.Add(-48 * time.Hour)},
	}
	svc := application.NewService(repo)

	access, err := svc.Evaluate(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, []string{domain.FeatureInvoicingCaptureFromEmail}, access.Features)
}

func TestRequire_SendForbiddenWhenMissing(t *testing.T) {
	now := time.Now().UTC()
	repo := newFakeRepo()
	repo.byTenant["t1"] = []domain.Entitlement{
		{FeatureKey: domain.FeatureMailInbox, Status: domain.StatusActive, StartsAt: now.Add(-time.Hour)},
	}
	svc := application.NewService(repo)
	ctx := tenant.WithTenantID(context.Background(), "t1")

	err := svc.Require(ctx, domain.FeatureMailSend)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeForbidden, appErr.Code)
	assert.Equal(t, domain.FeatureMailSend, appErr.Meta["feature_key"])
}

func TestRequireAny_AllowsIngestWithoutMailInbox(t *testing.T) {
	now := time.Now().UTC()
	repo := newFakeRepo()
	repo.byTenant["t1"] = []domain.Entitlement{
		{FeatureKey: domain.FeatureInvoicingCaptureFromEmail, Status: domain.StatusActive, StartsAt: now.Add(-time.Hour)},
	}
	svc := application.NewService(repo)
	ctx := tenant.WithTenantID(context.Background(), "t1")

	require.NoError(t, svc.RequireAny(ctx, domain.FeatureMailInbox, domain.FeatureInvoicingCaptureFromEmail))
	require.Error(t, svc.Require(ctx, domain.FeatureMailInbox))
}

func TestApplyDefaultPack_ExcludesSend(t *testing.T) {
	repo := newFakeRepo()
	svc := application.NewService(repo)
	require.NoError(t, svc.ApplyDefaultPack(context.Background(), "t1", "op-1"))

	access, err := svc.Evaluate(context.Background(), "t1")
	require.NoError(t, err)
	assert.Contains(t, access.Features, domain.FeatureMailInbox)
	assert.Contains(t, access.Features, domain.FeatureInvoicingCaptureFromEmail)
	assert.NotContains(t, access.Features, domain.FeatureMailSend)
}

func TestSetAccess_RevokeMailRemovesSend(t *testing.T) {
	repo := newFakeRepo()
	svc := application.NewService(repo)
	ctx := context.Background()
	require.NoError(t, svc.ApplyDefaultPack(ctx, "t1", "op-1"))
	require.NoError(t, svc.SetAccess(ctx, "t1", application.SetAccessInput{Feature: domain.FeatureMailSend, Enabled: true, ActorID: "op-1"}))
	require.NoError(t, svc.SetAccess(ctx, "t1", application.SetAccessInput{Product: domain.ProductMail, Enabled: false, ActorID: "op-1"}))

	access, err := svc.Evaluate(ctx, "t1")
	require.NoError(t, err)
	assert.NotContains(t, access.Features, domain.FeatureMailInbox)
	assert.NotContains(t, access.Features, domain.FeatureMailSend)
	assert.Contains(t, access.Features, domain.FeatureInvoicingCaptureFromEmail)
}
