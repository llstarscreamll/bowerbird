package sweeper

import (
	"context"
	"errors"
	"testing"

	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLister struct {
	slugs []string
	err   error
}

func (f fakeLister) ListActiveTenantSlugs(context.Context) ([]string, error) {
	return f.slugs, f.err
}

func TestHandlerIsPlatform(t *testing.T) {
	assert.Equal(t, jobs.ScopePlatform, (&Handler{}).Scope())
}

func TestHandlerAcksWhenNoTenants(t *testing.T) {
	h := &Handler{tenants: fakeLister{}, retention: defaultOutboxRetention}
	err := h.Handle(context.Background(), jobs.JobMessage{MessageID: "j1", JobType: h.JobType()})
	require.NoError(t, err)
}

func TestHandlerReturnsListerError(t *testing.T) {
	h := &Handler{tenants: fakeLister{err: errors.New("db down")}, retention: defaultOutboxRetention}
	err := h.Handle(context.Background(), jobs.JobMessage{MessageID: "j1"})
	require.EqualError(t, err, "db down")
}

func TestNewHandlerRequiresDeps(t *testing.T) {
	assert.Panics(t, func() { NewHandler(nil, fakeLister{}, 0) })
	assert.Panics(t, func() { NewHandler(&database.Registry{}, nil, 0) })
}
