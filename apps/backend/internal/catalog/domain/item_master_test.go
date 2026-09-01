package domain_test

import (
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInternalSKUAndItemKind(t *testing.T) {
	sku, err := domain.ParseInternalSKU("  ABC-1  ")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1", sku.String())
	assert.True(t, sku.Equals(sku))

	_, err = domain.ParseInternalSKU("   ")
	assert.ErrorIs(t, err, domain.ErrMissingInternalSKU)

	kind, err := domain.ParseItemKind("goods")
	require.NoError(t, err)
	assert.Equal(t, domain.KindGoods, kind.String())

	_, err = domain.ParseItemKind("widget")
	assert.ErrorIs(t, err, domain.ErrInvalidItemKind)
}

func TestNewManualItemAndInternalSKUAlias(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	kind, err := domain.ParseItemKind(domain.KindService)
	require.NoError(t, err)
	sku, err := domain.ParseInternalSKU("SRV-01")
	require.NoError(t, err)

	item, err := domain.NewManualItem("01ITEM", "Consulting", kind, sku, now)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, item.Status)
	assert.Equal(t, domain.CreationSourceManual, item.CreationSource)
	assert.True(t, item.IsConfirmed())
	assert.False(t, item.IsProvisional())

	alias, err := domain.NewInternalSKUAlias("01ALIAS", item.ID, sku, now)
	require.NoError(t, err)
	assert.Equal(t, domain.AliasSchemeInternalSKU, alias.Scheme)
	assert.Nil(t, alias.PartyID)
	assert.Equal(t, "SRV-01", alias.Value)
}

func TestItemConfirmAndAssignInternalSKU(t *testing.T) {
	now := time.Now().UTC()
	item, err := domain.NewProvisionalItem("01P", "Widget", "W-1", now)
	require.NoError(t, err)

	_, _, err = item.Confirm(nil, nil, now)
	assert.ErrorIs(t, err, domain.ErrConfirmRequiresSKU)

	sku, err := domain.ParseInternalSKU("W-1")
	require.NoError(t, err)
	got, assignNew, err := item.Confirm(nil, &sku, now)
	require.NoError(t, err)
	assert.True(t, assignNew)
	assert.Equal(t, "W-1", got.String())
	assert.True(t, item.IsConfirmed())

	_, _, err = item.Confirm(nil, &sku, now)
	assert.ErrorIs(t, err, domain.ErrItemAlreadyConfirmed)

	other, err := domain.ParseInternalSKU("OTHER")
	require.NoError(t, err)
	_, err = item.AssignInternalSKU(&sku, other, now)
	assert.ErrorIs(t, err, domain.ErrInternalSKUImmutable)

	prov, err := domain.NewProvisionalItem("01P2", "Gadget", "", now)
	require.NoError(t, err)
	assign, err := prov.AssignInternalSKU(nil, sku, now)
	require.NoError(t, err)
	assert.True(t, assign)

	require.NoError(t, prov.Rename("Gadget Pro", now))
	assert.Equal(t, "Gadget Pro", prov.Name)
	kind, err := domain.ParseItemKind(domain.KindAsset)
	require.NoError(t, err)
	prov.ChangeKind(kind, now)
	assert.Equal(t, domain.KindAsset, prov.Kind)

	parsedKind, err := prov.ItemKind()
	require.NoError(t, err)
	assert.True(t, parsedKind.Equals(kind))
}

func TestInterpretMasterStatusChange(t *testing.T) {
	now := time.Now().UTC()
	prov, err := domain.NewProvisionalItem("01P", "Widget", "W-1", now)
	require.NoError(t, err)

	confirm, err := prov.InterpretMasterStatusChange(domain.StatusConfirmed)
	require.NoError(t, err)
	assert.True(t, confirm)

	confirm, err = prov.InterpretMasterStatusChange(domain.StatusProvisional)
	require.NoError(t, err)
	assert.False(t, confirm) // already provisional → no-op

	confirm, err = prov.InterpretMasterStatusChange("")
	require.NoError(t, err)
	assert.False(t, confirm)

	_, err = prov.InterpretMasterStatusChange("archived")
	assert.ErrorIs(t, err, domain.ErrInvalidItemStatus)

	sku, err := domain.ParseInternalSKU("W-1")
	require.NoError(t, err)
	_, _, err = prov.Confirm(nil, &sku, now)
	require.NoError(t, err)

	_, err = prov.InterpretMasterStatusChange(domain.StatusProvisional)
	assert.ErrorIs(t, err, domain.ErrCannotRevertToProvisional)
}
