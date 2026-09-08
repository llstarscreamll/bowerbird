package jobs

import (
	"context"
	"testing"

	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubHandler struct {
	jobType string
	scope   Scope
	tenant  string
	handled string
}

func (h *stubHandler) JobType() string { return h.jobType }
func (h *stubHandler) Scope() Scope    { return h.scope }
func (h *stubHandler) Handle(ctx context.Context, msg JobMessage) error {
	h.handled = msg.MessageID
	h.tenant, _ = tenant.TenantIDFromContext(ctx)
	return nil
}

func TestHandleJobTenantInjectsContext(t *testing.T) {
	handler := &stubHandler{jobType: "TenantJob"}
	verifier := attestation.NewVerifier("secret")
	router := NewRouter(verifier, handler)

	err := router.HandleJob(context.Background(), JobMessage{
		MessageID:         "m1",
		JobType:           "TenantJob",
		TenantSlug:        "acme",
		TenantAttestation: verifier.Sign("m1", "acme", "TenantJob"),
	})
	require.NoError(t, err)
	assert.Equal(t, "m1", handler.handled)
	assert.Equal(t, "acme", handler.tenant)
}

func TestHandleJobTenantRejectsEmptySlug(t *testing.T) {
	handler := &stubHandler{jobType: "TenantJob"}
	verifier := attestation.NewVerifier("secret")
	router := NewRouter(verifier, handler)

	err := router.HandleJob(context.Background(), JobMessage{
		MessageID:         "m1",
		JobType:           "TenantJob",
		TenantAttestation: verifier.Sign("m1", attestation.PlatformSubject, "TenantJob"),
	})
	require.ErrorIs(t, err, tenant.ErrNoTenantIdInContext)
	assert.Empty(t, handler.handled)
}

func TestHandleJobTenantRejectsPlatformSubject(t *testing.T) {
	handler := &stubHandler{jobType: "TenantJob"}
	verifier := attestation.NewVerifier("secret")
	router := NewRouter(verifier, handler)

	err := router.HandleJob(context.Background(), JobMessage{
		MessageID:         "m1",
		JobType:           "TenantJob",
		TenantSlug:        attestation.PlatformSubject,
		TenantAttestation: verifier.Sign("m1", attestation.PlatformSubject, "TenantJob"),
	})
	require.ErrorIs(t, err, ErrJobScopeMismatch)
	assert.Empty(t, handler.handled)
}

func TestHandleJobPlatformOmitsTenantContext(t *testing.T) {
	handler := &stubHandler{jobType: "PlatformJob", scope: ScopePlatform}
	verifier := attestation.NewVerifier("secret")
	router := NewRouter(verifier, handler)

	err := router.HandleJob(context.Background(), JobMessage{
		MessageID:         "m1",
		JobType:           "PlatformJob",
		TenantAttestation: verifier.Sign("m1", "", "PlatformJob"),
	})
	require.NoError(t, err)
	assert.Equal(t, "m1", handler.handled)
	assert.Empty(t, handler.tenant)
}

func TestHandleJobPlatformRejectsTenantSlug(t *testing.T) {
	handler := &stubHandler{jobType: "PlatformJob", scope: ScopePlatform}
	verifier := attestation.NewVerifier("secret")
	router := NewRouter(verifier, handler)

	err := router.HandleJob(context.Background(), JobMessage{
		MessageID:         "m1",
		JobType:           "PlatformJob",
		TenantSlug:        "acme",
		TenantAttestation: verifier.Sign("m1", "acme", "PlatformJob"),
	})
	require.ErrorIs(t, err, ErrJobScopeMismatch)
	assert.Empty(t, handler.handled)
}

func TestHandleJobPlatformRejectsInvalidAttestation(t *testing.T) {
	handler := &stubHandler{jobType: "PlatformJob", scope: ScopePlatform}
	verifier := attestation.NewVerifier("secret")
	router := NewRouter(verifier, handler)

	err := router.HandleJob(context.Background(), JobMessage{
		MessageID:         "m1",
		JobType:           "PlatformJob",
		TenantAttestation: verifier.Sign("m1", "acme", "PlatformJob"),
	})
	require.ErrorIs(t, err, attestation.ErrInvalidAttestation)
	assert.Empty(t, handler.handled)
}
