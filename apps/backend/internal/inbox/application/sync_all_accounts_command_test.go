package application_test

import (
	"context"
	"errors"
	"testing"

	connectionsApp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	inboxApp "github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncAllConnectionsCommand_DispatchesJobPerActiveAccount(t *testing.T) {
	connectionsService := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{
			{ID: "acc-1", Provider: "gmail"},
			{ID: "acc-2", Provider: "outlook"},
		},
	}
	jobDispatcher := &fakeSyncAccountJobDispatcher{}
	cmd := inboxApp.NewSyncAllAccountsCommand(connectionsService, jobDispatcher)

	ctx := tenant.WithTenantID(context.Background(), "tenant-a")
	err := cmd.Execute(ctx, "user-1")

	require.NoError(t, err)
	require.Len(t, jobDispatcher.jobs, 2)
	assert.Equal(t, "tenant-a", jobDispatcher.jobs[0].TenantID)
	assert.Equal(t, "acc-1", jobDispatcher.jobs[0].AccountID)
	assert.Equal(t, "tenant-a", jobDispatcher.jobs[1].TenantID)
	assert.Equal(t, "acc-2", jobDispatcher.jobs[1].AccountID)
}

func TestSyncAllConnectionsCommand_NoActiveAccountsDoesNothing(t *testing.T) {
	connectionsService := &fakeConnectionsInternalService{}
	jobDispatcher := &fakeSyncAccountJobDispatcher{}
	cmd := inboxApp.NewSyncAllAccountsCommand(connectionsService, jobDispatcher)

	ctx := tenant.WithTenantID(context.Background(), "tenant-a")
	err := cmd.Execute(ctx, "user-1")

	require.NoError(t, err)
	assert.Len(t, jobDispatcher.jobs, 0)
}

func TestSyncAllConnectionsCommand_ReturnsDispatchErrors(t *testing.T) {
	connectionsService := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{
			{ID: "acc-1", Provider: "gmail"},
			{ID: "acc-2", Provider: "gmail"},
		},
	}
	jobDispatcher := &fakeSyncAccountJobDispatcher{failAccountID: "acc-1"}
	cmd := inboxApp.NewSyncAllAccountsCommand(connectionsService, jobDispatcher)

	ctx := tenant.WithTenantID(context.Background(), "tenant-a")
	err := cmd.Execute(ctx, "user-1")

	require.Error(t, err)
	assert.Len(t, jobDispatcher.jobs, 2)
}

func TestSyncAllConnectionsCommand_SkipsPrivateAccountsFromOtherUsers(t *testing.T) {
	connectionsService := &fakeConnectionsInternalService{
		activeConnections: []connectionsApp.ConnectionInfo{
			{ID: "acc-private-me", Provider: "gmail", SharingPolicy: "private", OwnerUserID: "user-1"},
			{ID: "acc-private-other", Provider: "gmail", SharingPolicy: "private", OwnerUserID: "user-2"},
			{ID: "acc-shared", Provider: "gmail", SharingPolicy: "organization", OwnerUserID: "user-2"},
		},
	}
	jobDispatcher := &fakeSyncAccountJobDispatcher{}
	cmd := inboxApp.NewSyncAllAccountsCommand(connectionsService, jobDispatcher)

	ctx := tenant.WithTenantID(context.Background(), "tenant-a")
	err := cmd.Execute(ctx, "user-1")

	require.NoError(t, err)
	require.Len(t, jobDispatcher.jobs, 2)
	assert.Equal(t, "acc-private-me", jobDispatcher.jobs[0].AccountID)
	assert.Equal(t, "acc-shared", jobDispatcher.jobs[1].AccountID)
}

type fakeSyncAccountJobDispatcher struct {
	jobs          []inboxApp.SyncAccountJob
	failAccountID string
}

func (f *fakeSyncAccountJobDispatcher) DispatchSyncAccount(ctx context.Context, job inboxApp.SyncAccountJob) error {
	f.jobs = append(f.jobs, job)
	if job.AccountID == f.failAccountID {
		return errors.New("dispatch failed")
	}
	return nil
}
