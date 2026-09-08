package application_test

import (
	"context"
	"errors"
	"testing"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	entitlementsapi "github.com/bowerbird/internal/entitlements/api"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTenantLister struct {
	slugs []string
	err   error
}

func (f fakeTenantLister) ListActiveTenantSlugs(context.Context) ([]string, error) {
	return f.slugs, f.err
}

type fakeFeatures struct {
	forbidden map[string]bool
}

func (f *fakeFeatures) Require(ctx context.Context, featureKey string) error {
	return f.RequireAny(ctx, featureKey)
}

func (f *fakeFeatures) RequireAny(ctx context.Context, _ ...string) error {
	slug, _ := tenant.TenantIDFromContext(ctx)
	if f.forbidden[slug] {
		return appErrors.New(appErrors.CodeForbidden, "feature not available")
	}
	return nil
}

func (f *fakeFeatures) Has(ctx context.Context, featureKey string) (bool, error) {
	err := f.RequireAny(ctx, featureKey)
	return err == nil, nil
}

func TestPlatformSyncAllAccountsFansOutPerTenant(t *testing.T) {
	dispatcher := &fakeSyncAccountJobDispatcher{}
	cmd := inboxCommands.NewPlatformSyncAllAccountsCommand(
		inboxCommands.NewSyncAllAccountsCommand(&fakeConnectionsInternalService{
			activeConnections: []connectionsapi.ConnectionInfo{{ID: "acc-1", Provider: "gmail"}},
		}, dispatcher),
		&fakeFeatures{},
		fakeTenantLister{slugs: []string{"acme", "beta"}},
	)

	require.NoError(t, cmd.Execute(context.Background()))
	require.Len(t, dispatcher.jobs, 2)
	assert.Equal(t, "acme", dispatcher.jobs[0].TenantID)
	assert.Equal(t, "beta", dispatcher.jobs[1].TenantID)
}

func TestPlatformSyncAllAccountsChecksEntitlementsPerTenant(t *testing.T) {
	dispatcher := &fakeSyncAccountJobDispatcher{}
	cmd := inboxCommands.NewPlatformSyncAllAccountsCommand(
		inboxCommands.NewSyncAllAccountsCommand(&fakeConnectionsInternalService{
			activeConnections: []connectionsapi.ConnectionInfo{{ID: "acc-1", Provider: "gmail"}},
		}, dispatcher),
		&fakeFeatures{forbidden: map[string]bool{"beta": true}},
		fakeTenantLister{slugs: []string{"acme", "beta"}},
	)

	require.NoError(t, cmd.Execute(context.Background()))
	require.Len(t, dispatcher.jobs, 1)
	assert.Equal(t, "acme", dispatcher.jobs[0].TenantID)
}

func TestPlatformSyncAllAccountsAcksPartialTenantFailure(t *testing.T) {
	dispatcher := &fakeSyncAccountJobDispatcher{}
	cmd := inboxCommands.NewPlatformSyncAllAccountsCommand(
		inboxCommands.NewSyncAllAccountsCommand(&failingConnectionsLister{
			failTenant:        "beta",
			activeConnections: []connectionsapi.ConnectionInfo{{ID: "acc-1", Provider: "gmail"}},
		}, dispatcher),
		&fakeFeatures{},
		fakeTenantLister{slugs: []string{"acme", "beta"}},
	)

	require.NoError(t, cmd.Execute(context.Background()))
	require.Len(t, dispatcher.jobs, 1)
	assert.Equal(t, "acme", dispatcher.jobs[0].TenantID)
}

func TestPlatformSyncAllAccountsReturnsErrorWhenAllTenantsFail(t *testing.T) {
	dispatcher := &fakeSyncAccountJobDispatcher{failAll: true}
	cmd := inboxCommands.NewPlatformSyncAllAccountsCommand(
		inboxCommands.NewSyncAllAccountsCommand(&fakeConnectionsInternalService{
			activeConnections: []connectionsapi.ConnectionInfo{{ID: "acc-1", Provider: "gmail"}},
		}, dispatcher),
		&fakeFeatures{},
		fakeTenantLister{slugs: []string{"acme", "beta"}},
	)

	require.Error(t, cmd.Execute(context.Background()))
}

func TestPlatformSyncAllAccountsReturnsListerError(t *testing.T) {
	cmd := inboxCommands.NewPlatformSyncAllAccountsCommand(
		inboxCommands.NewSyncAllAccountsCommand(&fakeConnectionsInternalService{}, &fakeSyncAccountJobDispatcher{}),
		&fakeFeatures{},
		fakeTenantLister{err: errors.New("db down")},
	)
	require.EqualError(t, cmd.Execute(context.Background()), "db down")
}

type failingConnectionsLister struct {
	failTenant        string
	activeConnections []connectionsapi.ConnectionInfo
}

func (f *failingConnectionsLister) GetActiveConnections(ctx context.Context) ([]connectionsapi.ConnectionInfo, error) {
	slug, _ := tenant.TenantIDFromContext(ctx)
	if slug == f.failTenant {
		return nil, errors.New("list accounts failed")
	}
	return f.activeConnections, nil
}

func (f *failingConnectionsLister) DecryptCredentials(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (f *failingConnectionsLister) MarkRequiresReconnect(context.Context, string, string) error {
	return nil
}

func (f *failingConnectionsLister) GetSharingPolicy(context.Context, string) (string, error) {
	return "", nil
}

var _ entitlementsapi.Features = (*fakeFeatures)(nil)
