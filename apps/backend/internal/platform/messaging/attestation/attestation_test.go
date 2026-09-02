package attestation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignAndVerify(t *testing.T) {
	v := NewVerifier("test-secret")
	sig := v.Sign("msg-1", "acme", "SmokeJob")
	require.NoError(t, v.Verify("msg-1", "acme", "SmokeJob", sig))
}

func TestVerifyRejectsTamperedTenant(t *testing.T) {
	v := NewVerifier("test-secret")
	sig := v.Sign("msg-1", "acme", "SmokeJob")
	require.ErrorIs(t, v.Verify("msg-1", "victim", "SmokeJob", sig), ErrInvalidAttestation)
}

func TestVerifyRejectsMissingSignature(t *testing.T) {
	v := NewVerifier("test-secret")
	require.ErrorIs(t, v.Verify("msg-1", "acme", "SmokeJob", ""), ErrMissingAttestation)
}
