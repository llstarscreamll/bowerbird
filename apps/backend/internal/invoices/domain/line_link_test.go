package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyManualDecision(t *testing.T) {
	current := LineLink{Status: LinkStatusUnmatched}

	linked, err := ApplyManualDecision(current, MemoryActionLink, "ITEM-1", true)
	require.NoError(t, err)
	assert.True(t, linked.IsLinked())
	assert.Equal(t, LinkMethodManual, linked.Method)
	assert.True(t, linked.Locked)
	require.NotNil(t, linked.ItemID)
	assert.Equal(t, "ITEM-1", *linked.ItemID)

	_, err = ApplyManualDecision(current, MemoryActionLink, "", true)
	assert.ErrorIs(t, err, ErrItemIDRequired)

	locked := LineLink{Status: LinkStatusLinked, Locked: true}
	_, err = ApplyManualDecision(locked, MemoryActionLink, "ITEM-2", true)
	assert.ErrorIs(t, err, ErrLineLinkLocked)

	rejected, err := ApplyManualDecision(current, MemoryActionNeverMatch, "", true)
	require.NoError(t, err)
	assert.Equal(t, LinkStatusRejected, rejected.Status)
	assert.Nil(t, rejected.ItemID)

	_, err = ApplyManualDecision(current, "nope", "", false)
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestRememberedItemID(t *testing.T) {
	linked := "ITEM-1"
	assert.Equal(t, &linked, RememberedItemID(MemoryActionLink, &linked, "IGNORED"))
	assert.Nil(t, RememberedItemID(MemoryActionNeverMatch, nil, ""))

	blocked := RememberedItemID(MemoryActionNeverMatch, nil, "BLOCKED")
	require.NotNil(t, blocked)
	assert.Equal(t, "BLOCKED", *blocked)
}

func TestRecalculateLinkingStatus(t *testing.T) {
	assert.Equal(t, LinkingStatusPending, RecalculateLinkingStatus([]string{LinkStatusLinked, LinkStatusUnmatched}))
	assert.Equal(t, LinkingStatusLinked, RecalculateLinkingStatus([]string{LinkStatusLinked, LinkStatusRejected}))
}

func TestLineLinkFromRecord(t *testing.T) {
	link := LineLinkFromRecord("ITEM-1", LinkStatusLinked, LinkMethodManual, true, nil)
	assert.True(t, link.IsLinked())
	require.NotNil(t, link.ItemID)
	assert.Equal(t, "ITEM-1", *link.ItemID)

	itemID, status, method, locked, suggestions := link.PersistFields()
	require.NotNil(t, itemID)
	assert.Equal(t, "ITEM-1", *itemID)
	assert.Equal(t, LinkStatusLinked, status)
	assert.Equal(t, LinkMethodManual, method)
	assert.True(t, locked)
	assert.Equal(t, emptySuggestions, suggestions)
}
