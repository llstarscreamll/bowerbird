package commands

import (
	"context"
	"testing"
	"time"

	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddItemAlias_ManualSource(t *testing.T) {
	items := &memItems{items: map[string]domain.Item{"ITEM-1": {ID: "ITEM-1", Name: "Widget"}}}
	aliases := &memAliases{}
	cmd := NewAddItemAliasCommand(items, aliases)
	cmd.now = func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }

	err := cmd.Execute(context.Background(), AddItemAliasInput{
		ItemID:  "ITEM-1",
		AliasID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Scheme:  domain.AliasSchemeSupplierSKU,
		Value:   "ABC-1",
		PartyID: "P1",
	})
	require.NoError(t, err)
	got, err := aliases.FindBySchemePartyValue(context.Background(), domain.AliasSchemeSupplierSKU, "P1", "ABC-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, domain.AliasSourceManual, got.Source)
}

func TestAddItemAlias_ConflictIncludesOwner(t *testing.T) {
	items := &memItems{items: map[string]domain.Item{
		"ITEM-1": {ID: "ITEM-1"},
		"ITEM-J": {ID: "ITEM-J"},
	}}
	party := "P1"
	aliases := &memAliases{byKey: map[string]domain.Alias{
		aliasKey(domain.AliasSchemeSupplierSKU, "P1", "ABC-1"): {
			ID: "A-OLD", ItemID: "ITEM-J", Scheme: domain.AliasSchemeSupplierSKU, Value: "ABC-1", PartyID: &party,
		},
	}}
	cmd := NewAddItemAliasCommand(items, aliases)
	err := cmd.Execute(context.Background(), AddItemAliasInput{
		ItemID:  "ITEM-1",
		AliasID: "01ARZ3NDEKTSV4RRFFQ69G5FAW",
		Scheme:  domain.AliasSchemeSupplierSKU,
		Value:   "ABC-1",
		PartyID: "P1",
	})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeConflict, appErr.Code)
	assert.Equal(t, "ITEM-J", appErr.Meta["item_id"])
}

func TestRemoveItemAlias(t *testing.T) {
	items := &memItems{items: map[string]domain.Item{"ITEM-1": {ID: "ITEM-1"}}}
	party := "P1"
	aliases := &memAliases{byKey: map[string]domain.Alias{
		aliasKey(domain.AliasSchemeSupplierSKU, "P1", "ABC-1"): {
			ID: "A1", ItemID: "ITEM-1", Scheme: domain.AliasSchemeSupplierSKU, Value: "ABC-1", PartyID: &party,
		},
	}}
	cmd := NewRemoveItemAliasCommand(items, aliases)
	err := cmd.Execute(context.Background(), "ITEM-1", "A1")
	require.NoError(t, err)
	got, err := aliases.FindBySchemePartyValue(context.Background(), domain.AliasSchemeSupplierSKU, "P1", "ABC-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRememberDecision_AttachesAliasesAndMemory(t *testing.T) {
	items := &memItems{items: map[string]domain.Item{"ITEM-1": {ID: "ITEM-1", Name: "Widget"}}}
	aliases := &memAliases{}
	store := &catalogStore{items: items, aliases: aliases}
	cmd := NewRememberDecisionCommand(items, aliases, store)
	n := 0
	cmd.newID = func() string {
		n++
		return "ID-" + string(rune('A'+n-1))
	}
	err := cmd.Execute(context.Background(), RememberDecisionInput{
		PartyID:     "P1",
		SellerSKU:   "ABC-1",
		GTIN:        "7701234567890",
		Description: "Widget",
		Action:      domain.MemoryActionLink,
		ItemID:      "ITEM-1",
	})
	require.NoError(t, err)
	sku, err := aliases.FindBySchemePartyValue(context.Background(), domain.AliasSchemeSupplierSKU, "P1", "ABC-1")
	require.NoError(t, err)
	require.NotNil(t, sku)
	assert.Equal(t, domain.AliasSourceInvoice, sku.Source)
	gtin, err := aliases.FindBySchemePartyValue(context.Background(), domain.AliasSchemeGTIN, "", "7701234567890")
	require.NoError(t, err)
	require.NotNil(t, gtin)
}

func TestRememberDecision_ConflictOwner(t *testing.T) {
	items := &memItems{items: map[string]domain.Item{"ITEM-1": {ID: "ITEM-1"}}}
	aliases := &memAliases{byKey: map[string]domain.Alias{
		aliasKey(domain.AliasSchemeGTIN, "", "7701234567890"): {ItemID: "ITEM-J", Scheme: domain.AliasSchemeGTIN, Value: "7701234567890"},
	}}
	store := &catalogStore{items: items, aliases: aliases}
	cmd := NewRememberDecisionCommand(items, aliases, store)
	err := cmd.Execute(context.Background(), RememberDecisionInput{
		PartyID:   "P1",
		SellerSKU: "ABC-1",
		GTIN:      "7701234567890",
		Action:    domain.MemoryActionLink,
		ItemID:    "ITEM-1",
	})
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "ITEM-J", appErr.Meta["item_id"])
}
