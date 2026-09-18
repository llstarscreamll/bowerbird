package domain_test

import (
	"testing"
	"time"

	"github.com/atta/internal/catalog/domain"
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
	assert.False(t, domain.InternalCode{}.Assigned())

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
	got, ok := item.ParsedInternalCode()
	require.True(t, ok)
	assert.Equal(t, "SRV-01", got.String())
	assert.True(t, item.IsConfirmed())
	assert.False(t, item.IsProvisional())
}

func TestNewImportedItem(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	kind, err := domain.ParseImportKind("bien")
	require.NoError(t, err)
	code, err := domain.ParseInternalCode("IMP-1")
	require.NoError(t, err)
	item, err := domain.NewImportedItem("01IMPITEM00000000000000000", "Tornillo", kind, code, now)
	require.NoError(t, err)
	assert.Equal(t, domain.CreationSourceImport, item.CreationSource)
	assert.Equal(t, domain.KindGoods, item.Kind)
	assert.True(t, item.IsConfirmed())
}

func TestApplyImportDoesNotMutateIdentity(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	kind, err := domain.ParseItemKind(domain.KindGoods)
	require.NoError(t, err)
	code, err := domain.ParseInternalCode("SKU-1")
	require.NoError(t, err)
	item, err := domain.NewManualItem("01M", "Old", kind, code, now)
	require.NoError(t, err)

	next, err := domain.ParseItemKind(domain.KindService)
	require.NoError(t, err)
	changed, err := item.ApplyImport("New", next, now)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, "New", item.Name)
	assert.Equal(t, domain.KindService, item.Kind)
	assert.Equal(t, domain.CreationSourceManual, item.CreationSource)
	assert.Equal(t, "SKU-1", item.InternalCode)

	changed, err = item.ApplyImport("New", next, now)
	require.NoError(t, err)
	assert.False(t, changed)
}

func TestApplyImportConfirmsProvisional(t *testing.T) {
	now := time.Now().UTC()
	item, err := domain.NewProvisionalItem("01P", "Widget", "W-1", now)
	require.NoError(t, err)
	code, err := domain.ParseInternalCode("W-1")
	require.NoError(t, err)
	require.NoError(t, item.AssignInternalCode(code, now))

	kind, err := domain.ParseItemKind(domain.KindGoods)
	require.NoError(t, err)
	changed, err := item.ApplyImport("Widget", kind, now)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.True(t, item.IsConfirmed())
	assert.Equal(t, domain.CreationSourceInvoice, item.CreationSource)
}

func TestParseImportKind(t *testing.T) {
	kind, err := domain.ParseImportKind("")
	require.NoError(t, err)
	assert.Equal(t, domain.KindUnknown, kind.String())
	_, err = domain.ParseImportKind("widget")
	assert.ErrorIs(t, err, domain.ErrInvalidItemKind)
}

func TestItemConfirmAndAssignInternalCode(t *testing.T) {
	now := time.Now().UTC()
	item, err := domain.NewProvisionalItem("01P", "Widget", "W-1", now)
	require.NoError(t, err)

	err = item.Confirm(domain.InternalCode{}, now)
	assert.ErrorIs(t, err, domain.ErrConfirmRequiresCode)

	code, err := domain.ParseInternalCode("W-1")
	require.NoError(t, err)
	require.NoError(t, item.Confirm(code, now))
	got, ok := item.ParsedInternalCode()
	require.True(t, ok)
	assert.Equal(t, "W-1", got.String())
	assert.True(t, item.IsConfirmed())

	err = item.Confirm(code, now)
	assert.ErrorIs(t, err, domain.ErrItemAlreadyConfirmed)

	other, err := domain.ParseInternalCode("OTHER")
	require.NoError(t, err)
	err = item.AssignInternalCode(other, now)
	assert.ErrorIs(t, err, domain.ErrInternalCodeImmutable)

	prov, err := domain.NewProvisionalItem("01P2", "Gadget", "", now)
	require.NoError(t, err)
	require.NoError(t, prov.AssignInternalCode(code, now))
	assigned, ok := prov.ParsedInternalCode()
	require.True(t, ok)
	assert.Equal(t, "W-1", assigned.String())
	require.NoError(t, prov.Confirm(domain.InternalCode{}, now))
	assert.True(t, prov.IsConfirmed())

	require.NoError(t, prov.Rename("Gadget Pro", now))
	assert.Equal(t, "Gadget Pro", prov.Name)
	kind, err := domain.ParseItemKind(domain.KindAsset)
	require.NoError(t, err)
	require.NoError(t, prov.ChangeKind(kind, now))
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
	require.NoError(t, prov.Confirm(code, now))

	_, err = prov.InterpretMasterStatusChange(domain.StatusProvisional)
	assert.ErrorIs(t, err, domain.ErrCannotRevertToProvisional)
}

func TestMergeInto(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	kind, err := domain.ParseItemKind(domain.KindGoods)
	require.NoError(t, err)
	code, err := domain.ParseInternalCode("SKU-M")
	require.NoError(t, err)
	item, err := domain.NewManualItem("01SRC", "Widget", kind, code, now)
	require.NoError(t, err)

	require.NoError(t, item.MergeInto("01DST", now))
	assert.True(t, item.IsMerged())
	assert.Equal(t, "01DST", item.MergedIntoID)
	assert.Empty(t, item.InternalCode)
	require.NoError(t, item.MergeInto("01DST", now))
	assert.ErrorIs(t, item.MergeInto("01OTHER", now), domain.ErrItemAlreadyMerged)
	assert.ErrorIs(t, item.MergeInto("01SRC", now), domain.ErrCannotMergeIntoSelf)
}
