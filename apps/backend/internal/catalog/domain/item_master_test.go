package domain_test

import (
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInternalCodeAndItemKind(t *testing.T) {
	code, err := domain.ParseInternalCode("  ABC-1  ")
	require.NoError(t, err)
	assert.Equal(t, "ABC-1", code.String())
	assert.True(t, code.Equals(code))

	_, err = domain.ParseInternalCode("   ")
	assert.ErrorIs(t, err, domain.ErrMissingInternalCode)

	kind, err := domain.ParseItemKind("goods")
	require.NoError(t, err)
	assert.Equal(t, domain.KindGoods, kind.String())

	_, err = domain.ParseItemKind("widget")
	assert.ErrorIs(t, err, domain.ErrInvalidItemKind)
}

func TestNewManualItemRequiresInternalCode(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	kind, err := domain.ParseItemKind(domain.KindService)
	require.NoError(t, err)
	code, err := domain.ParseInternalCode("SRV-01")
	require.NoError(t, err)

	item, err := domain.NewManualItem("01ITEM", "Consulting", kind, code, now)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusConfirmed, item.Status)
	assert.Equal(t, domain.CreationSourceManual, item.CreationSource)
	assert.Equal(t, "SRV-01", item.InternalCode)
	assert.True(t, item.IsConfirmed())
	assert.False(t, item.IsProvisional())
}

func TestItemConfirmAndAssignInternalCode(t *testing.T) {
	now := time.Now().UTC()
	item, err := domain.NewProvisionalItem("01P", "Widget", "W-1", now)
	require.NoError(t, err)

	err = item.Confirm(nil, now)
	assert.ErrorIs(t, err, domain.ErrConfirmRequiresCode)

	code, err := domain.ParseInternalCode("W-1")
	require.NoError(t, err)
	require.NoError(t, item.Confirm(&code, now))
	assert.Equal(t, "W-1", item.InternalCode)
	assert.True(t, item.IsConfirmed())

	err = item.Confirm(&code, now)
	assert.ErrorIs(t, err, domain.ErrItemAlreadyConfirmed)

	other, err := domain.ParseInternalCode("OTHER")
	require.NoError(t, err)
	err = item.AssignInternalCode(other, now)
	assert.ErrorIs(t, err, domain.ErrInternalCodeImmutable)

	prov, err := domain.NewProvisionalItem("01P2", "Gadget", "", now)
	require.NoError(t, err)
	require.NoError(t, prov.AssignInternalCode(code, now))
	assert.Equal(t, "W-1", prov.InternalCode)

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

	code, err := domain.ParseInternalCode("W-1")
	require.NoError(t, err)
	require.NoError(t, prov.Confirm(&code, now))

	_, err = prov.InterpretMasterStatusChange(domain.StatusProvisional)
	assert.ErrorIs(t, err, domain.ErrCannotRevertToProvisional)
}
