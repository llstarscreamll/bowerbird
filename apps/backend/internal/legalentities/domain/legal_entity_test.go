package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaxID(t *testing.T) {
	id, err := ParseTaxID("900.123.456-7")
	require.NoError(t, err)
	assert.Equal(t, "9001234567", id.String())

	_, err = ParseTaxID("  --  ")
	assert.ErrorIs(t, err, ErrMissingTaxID)
}

func TestParseScheme(t *testing.T) {
	scheme, err := ParseScheme("31")
	require.NoError(t, err)
	assert.Equal(t, SchemeNIT, scheme.String())

	_, err = ParseScheme("99")
	assert.ErrorIs(t, err, ErrInvalidScheme)
}

func TestTaxIDMatches(t *testing.T) {
	id, err := ParseTaxID("900123")
	require.NoError(t, err)
	assert.True(t, id.Matches("900.123"))
	assert.False(t, id.Matches("900124"))
	assert.False(t, id.Matches(""))
}

func TestNewLegalEntity(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	tax, err := ParseTaxID("900.1")
	require.NoError(t, err)
	scheme, err := ParseScheme("31")
	require.NoError(t, err)

	entity, err := NewLegalEntity("LE1", tax, scheme, " Acme SAS ", now)
	require.NoError(t, err)
	assert.Equal(t, "9001", entity.TaxID)
	assert.Equal(t, "Acme SAS", entity.LegalName)
	assert.Equal(t, SchemeNIT, entity.SchemeID)
	assert.True(t, RequiresRegistrationNotice(entity.PullEvents()))
	assert.True(t, entity.ReceivesAs("900.1"))
	assert.False(t, entity.ReceivesAs("9002"))
}

func TestChangeIdentityRecordsEvent(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	tax, _ := ParseTaxID("9001")
	scheme, _ := ParseScheme("31")
	entity, err := NewLegalEntity("LE1", tax, scheme, "Acme", now)
	require.NoError(t, err)
	_ = entity.PullEvents()

	next, err := ParseTaxID("9002")
	require.NoError(t, err)
	assert.True(t, entity.ChangeIdentity(next, scheme, now))
	assert.True(t, RequiresRegistrationNotice(entity.PullEvents()))
}

func TestRenameDoesNotRecordIdentityEvent(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	tax, _ := ParseTaxID("9001")
	scheme, _ := ParseScheme("31")
	entity, err := NewLegalEntity("LE1", tax, scheme, "Acme", now)
	require.NoError(t, err)
	_ = entity.PullEvents()

	require.NoError(t, entity.Rename("Acme SAS", now))
	assert.False(t, RequiresRegistrationNotice(entity.PullEvents()))
}

func TestAssertRegistrable(t *testing.T) {
	require.NoError(t, AssertRegistrable(0))
	assert.ErrorIs(t, AssertRegistrable(1), ErrAlreadyRegistered)
}
