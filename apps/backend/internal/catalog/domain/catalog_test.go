package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvisionalItem(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	item, err := NewProvisionalItem("I1", "  Widget  ", "SKU", now)
	require.NoError(t, err)
	assert.Equal(t, "Widget", item.Name)
	assert.Equal(t, StatusProvisional, item.Status)
	assert.Equal(t, KindUnknown, item.Kind)
	assert.True(t, item.IsProvisional())

	fallback, err := NewProvisionalItem("I2", "  ", "SKU-2", now)
	require.NoError(t, err)
	assert.Equal(t, "SKU-2", fallback.Name)

	_, err = NewProvisionalItem("I3", "", "", now)
	assert.ErrorIs(t, err, ErrMissingItemName)
}

func TestNewSupplierSKUAlias(t *testing.T) {
	now := time.Now().UTC()
	alias, err := NewSupplierSKUAlias("A1", "ITEM-1", "P1", "  SKU  ", now)
	require.NoError(t, err)
	assert.Equal(t, AliasSchemeSupplierSKU, alias.Scheme)
	assert.Equal(t, "SKU", alias.Value)
	require.NotNil(t, alias.PartyID)
	assert.Equal(t, "P1", *alias.PartyID)
	assert.True(t, alias.PointsTo("ITEM-1"))
	assert.False(t, alias.PointsTo("OTHER"))
}

func TestPreserveLockedLinkAndSoftStatus(t *testing.T) {
	assert.Nil(t, PreserveLockedLink(LineResolutionInput{}))
	preserved := PreserveLockedLink(LineResolutionInput{
		ExistingLocked: true,
		ExistingItemID: "ITEM-1",
	})
	require.NotNil(t, preserved)
	assert.Equal(t, LinkMethodManual, preserved.Method)

	assert.Equal(t, LinkStatusUnmatched, SoftOrUnmatchedStatus(nil))
	assert.Equal(t, LinkStatusSuggested, SoftOrUnmatchedStatus([]Suggestion{{ItemID: "I"}}))
	assert.True(t, CanMintProvisional("P", "SKU"))
	assert.False(t, CanMintProvisional("", "SKU"))
}

func TestNewMatchMemory(t *testing.T) {
	now := time.Now().UTC()
	itemID := "ITEM-1"
	mem, err := NewMatchMemory("M1", "P1", "SKU", "Widget", MemoryActionLink, &itemID, now)
	require.NoError(t, err)
	assert.Equal(t, MemoryActionLink, mem.Action)
	assert.NotEmpty(t, mem.EvidenceKey)

	blocked := "ITEM-BAD"
	never, err := NewMatchMemory("M2", "P1", "SKU", "Widget", MemoryActionNeverMatch, &blocked, now)
	require.NoError(t, err)
	assert.True(t, never.IsNeverMatch())
	assert.Equal(t, "ITEM-BAD", never.LinkedItemID())
}
