package sweeper

import (
	"context"
	"testing"

	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
)

func TestHandlerRequiresTenantSlug(t *testing.T) {
	h := &Handler{retention: defaultOutboxRetention}
	err := h.Handle(context.Background(), jobs.JobMessage{JobType: h.JobType()})
	assert.ErrorIs(t, err, tenant.ErrNoTenantIdInContext)
}
