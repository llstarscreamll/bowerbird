package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaxID(t *testing.T) {
	taxID, err := ParseTaxID(" 900123 ")
	require.NoError(t, err)
	assert.Equal(t, "900123", taxID.String())

	_, err = ParseTaxID("  ")
	assert.ErrorIs(t, err, ErrMissingTaxID)
}

func TestParsePartyRoles(t *testing.T) {
	roles, err := ParsePartyRoles([]string{"supplier", "customer"})
	require.NoError(t, err)
	assert.Equal(t, []string{"customer", "supplier"}, roles.Strings())

	_, err = ParsePartyRoles(nil)
	assert.ErrorIs(t, err, ErrMissingRoles)

	_, err = ParsePartyRoles([]string{"invalid"})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

func TestNewProvisionalSupplier(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	taxID, err := ParseTaxID("900123")
	require.NoError(t, err)
	party := NewProvisionalSupplier("P1", taxID, "  Acme  ", now)
	assert.Equal(t, "900123", party.TaxID)
	assert.Equal(t, "Acme", party.Name)
	assert.True(t, party.HasRole(RoleSupplier))
	assert.Equal(t, StatusProvisional, party.Status)
	assert.Equal(t, CreationSourceInvoice, party.CreationSource)

	taxID2, err := ParseTaxID("901")
	require.NoError(t, err)
	fallback := NewProvisionalSupplier("P2", taxID2, "  ", now)
	assert.Equal(t, "901", fallback.Name)
}

func TestNewConfirmedParty(t *testing.T) {
	now := time.Now().UTC()
	taxID, err := ParseTaxID("900")
	require.NoError(t, err)
	roles, err := ParsePartyRoles([]string{RoleCustomer})
	require.NoError(t, err)

	party, err := NewConfirmedParty("P1", taxID, "Acme", roles, now)
	require.NoError(t, err)
	assert.Equal(t, StatusConfirmed, party.Status)
	assert.Equal(t, CreationSourceManual, party.CreationSource)
	assert.True(t, party.HasRole(RoleCustomer))

	_, err = NewConfirmedParty("P2", taxID, "  ", roles, now)
	assert.ErrorIs(t, err, ErrMissingPartyName)
}

func TestEnsureSupplierRoleAndRename(t *testing.T) {
	now := time.Now().UTC()
	taxID, err := ParseTaxID("900")
	require.NoError(t, err)
	party := NewProvisionalSupplier("P1", taxID, "Acme", now)

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

func TestUpdateProfile(t *testing.T) {
	now := time.Now().UTC()
	taxID, err := ParseTaxID("900")
	require.NoError(t, err)
	roles, err := ParsePartyRoles([]string{RoleSupplier})
	require.NoError(t, err)
	party, err := NewConfirmedParty("P1", taxID, "Acme", roles, now)
	require.NoError(t, err)

	newRoles, err := ParsePartyRoles([]string{RoleSupplier, RoleCustomer})
	require.NoError(t, err)
	name := "New Acme"
	changed, err := party.UpdateProfile(&name, &newRoles, now.Add(time.Hour))
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, "New Acme", party.Name)
	assert.True(t, party.HasRole(RoleCustomer))
}
