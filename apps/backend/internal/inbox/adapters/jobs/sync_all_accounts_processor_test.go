package jobs

import (
	"context"
	"testing"

	connectionsapi "github.com/bowerbird/internal/connections/api"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLister struct {
	slugs []string
}

func (f fakeLister) ListActiveTenantSlugs(context.Context) ([]string, error) {
	return f.slugs, nil
}

type allowAllFeatures struct{}

func (allowAllFeatures) Require(context.Context, string) error       { return nil }
func (allowAllFeatures) RequireAny(context.Context, ...string) error { return nil }
func (allowAllFeatures) Has(context.Context, string) (bool, error)   { return true, nil }

type stubConnections struct{}

func (stubConnections) GetActiveConnections(context.Context) ([]connectionsapi.ConnectionInfo, error) {
	return nil, nil
}
func (stubConnections) DecryptCredentials(context.Context, string) ([]byte, error) { return nil, nil }
func (stubConnections) MarkRequiresReconnect(context.Context, string, string) error {
	return nil
}
func (stubConnections) GetSharingPolicy(context.Context, string) (string, error) { return "", nil }

type stubDispatcher struct{}

func (stubDispatcher) DispatchSyncAccount(context.Context, inboxCommands.SyncAccountJob) error {
	return nil
}

func TestProcessInboxSyncAllAccountsIsPlatform(t *testing.T) {
	h := NewProcessInboxSyncAllAccounts(inboxCommands.NewPlatformSyncAllAccountsCommand(
		inboxCommands.NewSyncAllAccountsCommand(stubConnections{}, stubDispatcher{}),
		allowAllFeatures{},
		fakeLister{},
	))
	assert.Equal(t, platformJobs.ScopePlatform, h.Scope())
}

func TestProcessInboxSyncAllAccountsDelegatesToCommand(t *testing.T) {
	h := NewProcessInboxSyncAllAccounts(inboxCommands.NewPlatformSyncAllAccountsCommand(
		inboxCommands.NewSyncAllAccountsCommand(stubConnections{}, stubDispatcher{}),
		allowAllFeatures{},
		fakeLister{slugs: []string{}},
	))
	require.NoError(t, h.Handle(context.Background(), platformJobs.JobMessage{MessageID: "job-1"}))
}

func TestParentCorrelationFallsBackToMessageID(t *testing.T) {
	assert.Equal(t, "mid", parentCorrelation(platformJobs.JobMessage{MessageID: "mid"}))
	assert.Equal(t, "cid", parentCorrelation(platformJobs.JobMessage{MessageID: "mid", CorrelationID: "cid"}))
}
