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
	assert.Equal(t, CreationSourceInvoice, item.CreationSource)
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
	alias, err := NewSupplierSKUAlias("A1", "ITEM-1", "P1", "  SKU  ", AliasSourceInvoice, now)
	require.NoError(t, err)
	assert.Equal(t, AliasSchemeSupplierSKU, alias.Scheme)
	assert.Equal(t, "SKU", alias.Value)
	assert.Equal(t, AliasSourceInvoice, alias.Source)
	require.NotNil(t, alias.PartyID)
	assert.Equal(t, "P1", *alias.PartyID)
	assert.True(t, alias.PointsTo("ITEM-1"))
	assert.False(t, alias.PointsTo("OTHER"))

	_, err = NewSupplierSKUAlias("A2", "ITEM-1", "", "SKU", AliasSourceManual, now)
	assert.ErrorIs(t, err, ErrMissingAliasParty)
}

func TestNewGTINAlias(t *testing.T) {
	now := time.Now().UTC()
	alias, err := NewGTINAlias("A1", "ITEM-1", "7701234567890", AliasSourceManual, now)
	require.NoError(t, err)
	assert.Equal(t, AliasSchemeGTIN, alias.Scheme)
	assert.Equal(t, "7701234567890", alias.Value)
	assert.Nil(t, alias.PartyID)

	_, err = NewGTINAlias("A2", "ITEM-1", "MGND3LA/A", AliasSourceManual, now)
	assert.ErrorIs(t, err, ErrInvalidGTIN)
}

func TestParseGTINAndSellerUsable(t *testing.T) {
	_, ok := ClassifyGTIN("7701234567890")
	assert.True(t, ok)
	_, ok = ClassifyGTIN(" 7701 234567890 ")
	assert.True(t, ok)
	_, ok = ClassifyGTIN("MGND3LA/A")
	assert.False(t, ok)
	_, ok = ClassifyGTIN("123")
	assert.False(t, ok)

	for _, raw := range []string{"12345678", "123456789012", "7701234567890", "12345678901234"} {
		_, ok := ClassifyGTIN(raw)
		assert.True(t, ok, raw)
	}

	assert.True(t, ParseSellerSKU("ABC-1").Usable())
	assert.True(t, SellerSKUUsable("ABC-1"))
	assert.False(t, ParseSellerSKU("1").Usable())
	assert.False(t, SellerSKUUsable("1"))
	assert.False(t, SellerSKUUsable("01"))
	assert.False(t, SellerSKUUsable("001"))
	assert.False(t, SellerSKUUsable("n/a"))
	assert.False(t, SellerSKUUsable("na"))
	assert.False(t, SellerSKUUsable("serv"))
	assert.False(t, SellerSKUUsable("servicio"))
	assert.False(t, SellerSKUUsable("item"))
	assert.False(t, SellerSKUUsable("  "))
	assert.False(t, SellerSKUUsable("ab"))
}

func TestHardHitsAgree(t *testing.T) {
	item, conflict, ids := HardHits{SellerItemID: "A"}.Agree()
	assert.Equal(t, "A", item)
	assert.False(t, conflict)
	assert.Equal(t, []string{"A"}, ids)

	item, conflict, ids = HardHits{BuyerItemID: "A", SellerItemID: "B"}.Agree()
	assert.Empty(t, item)
	assert.True(t, conflict)
	assert.Equal(t, []string{"A", "B"}, ids)

	item, conflict, ids = HardHits{BuyerItemID: "A", GTINItemID: "A", SellerItemID: "A"}.Agree()
	assert.Equal(t, "A", item)
	assert.False(t, conflict)
	assert.Equal(t, []string{"A"}, ids)

	item, conflict, ids = HardHits{}.Agree()
	assert.Empty(t, item)
	assert.False(t, conflict)
	assert.Nil(t, ids)
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
	assert.False(t, CanMintProvisional("P", "1"))
	conflict := LinkedByHardConflict([]string{"A", "B"})
	assert.Equal(t, LinkStatusSuggested, conflict.Status)
	assert.Equal(t, SuggestionReasonHardConflict, conflict.Suggestions[0].Reason)
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

func TestUnionAliasesReassignsAndDropsCollidingTuple(t *testing.T) {
	party := "P1"
	held := []Alias{{ID: "H1", ItemID: "A", Scheme: AliasSchemeSupplierSKU, Value: "SKU-1", PartyID: &party, Source: AliasSourceManual}}
	incoming := []Alias{
		{ID: "I1", ItemID: "B", Scheme: AliasSchemeSupplierSKU, Value: "SKU-1", PartyID: &party, Source: AliasSourceInvoice},
		{ID: "I2", ItemID: "B", Scheme: AliasSchemeGTIN, Value: "7701234567890", Source: AliasSourceInvoice},
	}
	reassign, deleteIDs, err := UnionAliases("A", held, incoming)
	require.NoError(t, err)
	require.Equal(t, []string{"I1"}, deleteIDs)
	require.Len(t, reassign, 1)
	assert.Equal(t, "A", reassign[0].ItemID)
	assert.Equal(t, "I2", reassign[0].ID)
	assert.Equal(t, AliasSourceInvoice, reassign[0].Source)
}

func TestPickMergeNameAndKind(t *testing.T) {
	mint := Item{Name: "SKU", Kind: KindUnknown, CreationSource: CreationSourceInvoice}
	master := Item{Name: "Widget largo", Kind: KindGoods, CreationSource: CreationSourceManual}
	assert.Equal(t, "Widget largo", PickMergeName(mint, []Item{master}))
	assert.Equal(t, "Kept", PickMergeName(Item{Name: "Kept", CreationSource: CreationSourceManual}, []Item{master}))
	assert.Equal(t, KindGoods, PickMergeKind(mint, []Item{master}))
	assert.Equal(t, KindService, PickMergeKind(Item{Kind: KindService}, []Item{master}))
}
