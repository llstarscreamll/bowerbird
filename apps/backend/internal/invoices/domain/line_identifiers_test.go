package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLineIdentifiers(t *testing.T) {
	ids := NewLineIdentifiers("INT-9", "MGND3LA/A", "7701234567890")
	assert.Equal(t, "INT-9", ids.BuyerCode)
	assert.Equal(t, "MGND3LA/A", ids.SellerSKU)
	assert.Equal(t, "7701234567890", ids.GTIN)

	ids = NewLineIdentifiers("", "MGND3LA/A", "MGND3LA/A")
	assert.Equal(t, "MGND3LA/A", ids.SellerSKU)
	assert.Empty(t, ids.GTIN)

	ids = NewLineIdentifiers("", "", "ABC-1")
	assert.Equal(t, "ABC-1", ids.SellerSKU)
	assert.Empty(t, ids.GTIN)

	ids = NewLineIdentifiers("", "", "7701234567890")
	assert.Empty(t, ids.SellerSKU)
	assert.Equal(t, "7701234567890", ids.GTIN)
}

func TestClassifyGTIN(t *testing.T) {
	gtin, ok := ClassifyGTIN(" 7701 234567890 ")
	assert.True(t, ok)
	assert.Equal(t, "7701234567890", gtin)
	_, ok = ClassifyGTIN("MGND3LA/A")
	assert.False(t, ok)
	_, ok = ClassifyGTIN("123")
	assert.False(t, ok)
}

func TestFromCollapsedCode(t *testing.T) {
	ids := FromCollapsedCode("7701234567890")
	assert.Equal(t, "7701234567890", ids.GTIN)
	assert.Empty(t, ids.SellerSKU)

	ids = FromCollapsedCode("SKU-1")
	assert.Equal(t, "SKU-1", ids.SellerSKU)
	assert.Empty(t, ids.GTIN)
}

func TestLineLinkUnlock(t *testing.T) {
	itemID := "ITEM-1"
	locked := LineLink{ItemID: &itemID, Status: LinkStatusLinked, Method: LinkMethodManual, Locked: true}
	next, err := locked.Unlock()
	require.NoError(t, err)
	assert.False(t, next.Locked)
	assert.Equal(t, "ITEM-1", *next.ItemID)
	assert.Equal(t, LinkStatusLinked, next.Status)

	_, err = next.Unlock()
	assert.ErrorIs(t, err, ErrLineLinkNotLocked)

	unlocked, err := locked.ApplyManualDecision(ActionUnlock, "", false)
	require.NoError(t, err)
	assert.False(t, unlocked.Locked)
	assert.Equal(t, "ITEM-1", *unlocked.ItemID)
}

func TestLineForDecisionBelongsToInvoice(t *testing.T) {
	line := LineForDecision{InvoiceHeaderID: "INV-1", BuyerCode: "B", SellerSKU: "S", GTIN: "G"}
	assert.True(t, line.BelongsToInvoice("INV-1"))
	assert.False(t, line.BelongsToInvoice("OTHER"))
}
