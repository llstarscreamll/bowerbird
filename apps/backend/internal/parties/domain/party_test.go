package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvisionalSupplier(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	party, err := NewProvisionalSupplier("P1", " 900123 ", "  Acme  ", now)
	require.NoError(t, err)
	assert.Equal(t, "900123", party.TaxID)
	assert.Equal(t, "Acme", party.Name)
	assert.True(t, party.HasRole(RoleSupplier))
	assert.Equal(t, StatusProvisional, party.Status)

	fallback, err := NewProvisionalSupplier("P2", "901", "  ", now)
	require.NoError(t, err)
	assert.Equal(t, "901", fallback.Name)

	_, err = NewProvisionalSupplier("P3", "", "Acme", now)
	assert.ErrorIs(t, err, ErrMissingTaxID)
}

func TestEnsureSupplierRoleAndRename(t *testing.T) {
	now := time.Now().UTC()
	party, err := NewProvisionalSupplier("P1", "900", "Acme", now)
	require.NoError(t, err)

	assert.False(t, party.EnsureSupplierRole(now.Add(time.Minute)))
	party.Roles = nil
	assert.True(t, party.EnsureSupplierRole(now.Add(time.Hour)))
	assert.True(t, party.HasRole(RoleSupplier))

	err = party.Rename("  New Name ", now.Add(2*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "New Name", party.Name)

	err = party.Rename("  ", now)
	assert.ErrorIs(t, err, ErrMissingPartyName)
}
