package commands

import (
	"context"
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memMerge struct {
	items   *memItems
	aliases *memAliases
}

func (m *memMerge) ApplyMerge(ctx context.Context, in ports.MergePersistence) error {
	m.items.items[in.Survivor.ID] = in.Survivor
	for _, item := range in.Merged {
		m.items.items[item.ID] = item
	}
	drop := map[string]struct{}{}
	for _, id := range in.DeleteAliasIDs {
		drop[id] = struct{}{}
	}
	for key, alias := range m.aliases.byKey {
		if _, ok := drop[alias.ID]; ok {
			delete(m.aliases.byKey, key)
			continue
		}
	}
	for _, alias := range in.ReassignAliases {
		party := ""
		if alias.PartyID != nil {
			party = *alias.PartyID
		}
		m.aliases.byKey[aliasKey(alias.Scheme, party, alias.Value)] = alias
	}
	return nil
}

type memLinks struct {
	relinked [][]string
}

func (m *memLinks) RelinkItems(ctx context.Context, fromIDs []string, toID string) error {
	cp := append([]string{toID}, fromIDs...)
	m.relinked = append(m.relinked, cp)
	return nil
}
func (m *memLinks) HardConflictPairs(ctx context.Context) ([]ports.ItemIDPair, error) {
	return nil, nil
}
func (m *memLinks) CountLinks(ctx context.Context, itemIDs []string) (map[string]int, error) {
	return map[string]int{}, nil
}

func testItem(id, name, code string, now time.Time) domain.Item {
	return domain.Item{
		ID:             id,
		Name:           name,
		Kind:           domain.KindGoods,
		Status:         domain.StatusConfirmed,
		CreationSource: domain.CreationSourceManual,
		InternalCode:   code,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestMergeItemsUnionsAliases(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{
		"A": testItem("A", "Widget", "INT-A", now),
		"B": testItem("B", "Widget B", "INT-B", now),
	}}
	partyA, partyB := "P1", "P2"
	aliases := &memAliases{byKey: map[string]domain.Alias{
		"supplier_sku|P1|SKU-A": {ID: "AL1", ItemID: "A", Scheme: domain.AliasSchemeSupplierSKU, Value: "SKU-A", PartyID: &partyA, Source: domain.AliasSourceManual},
		"supplier_sku|P2|SKU-B": {ID: "AL2", ItemID: "B", Scheme: domain.AliasSchemeSupplierSKU, Value: "SKU-B", PartyID: &partyB, Source: domain.AliasSourceInvoice},
		"gtin||7701234567890":   {ID: "AL3", ItemID: "B", Scheme: domain.AliasSchemeGTIN, Value: "7701234567890", Source: domain.AliasSourceInvoice},
	}}
	links := &memLinks{}
	cmd := NewMergeItemsCommand(items, aliases, &memMerge{items: items, aliases: aliases}, links)
	cmd.now = func() time.Time { return now }
	name := "Widget"
	code := "INT-A"
	err := cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B"}, Name: &name, InternalCode: &code})
	require.NoError(t, err)

	assert.Equal(t, domain.StatusMerged, items.items["B"].Status)
	assert.Equal(t, "A", items.items["B"].MergedIntoID)
	assert.Equal(t, "", items.items["B"].InternalCode)
	assert.Equal(t, "Widget", items.items["A"].Name)

	byItem := map[string][]domain.Alias{}
	for _, alias := range aliases.byKey {
		byItem[alias.ItemID] = append(byItem[alias.ItemID], alias)
	}
	require.Len(t, byItem["B"], 0)
	require.Len(t, byItem["A"], 3)
	values := map[string]string{}
	for _, alias := range byItem["A"] {
		values[alias.Scheme+"|"+alias.Value] = alias.Source
	}
	assert.Equal(t, domain.AliasSourceManual, values["supplier_sku|SKU-A"])
	assert.Equal(t, domain.AliasSourceInvoice, values["supplier_sku|SKU-B"])
	assert.Equal(t, domain.AliasSourceInvoice, values["gtin|7701234567890"])
	require.Len(t, links.relinked, 1)
	assert.Equal(t, []string{"A", "B"}, links.relinked[0])
}

func TestMergeItemsDropsCollidingAliasTuple(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{
		"A": testItem("A", "Same", "INT-A", now),
		"B": {ID: "B", Name: "Same B", Kind: domain.KindGoods, Status: domain.StatusProvisional, CreationSource: domain.CreationSourceInvoice, CreatedAt: now, UpdatedAt: now},
	}}
	party := "P1"
	aliases := &memAliases{byKey: map[string]domain.Alias{
		"supplier_sku|P1|SKU-1":     {ID: "AL1", ItemID: "A", Scheme: domain.AliasSchemeSupplierSKU, Value: "SKU-1", PartyID: &party, Source: domain.AliasSourceManual},
		"supplier_sku|P1|SKU-1-dup": {ID: "AL2", ItemID: "B", Scheme: domain.AliasSchemeSupplierSKU, Value: "SKU-1", PartyID: &party, Source: domain.AliasSourceInvoice},
	}}
	cmd := NewMergeItemsCommand(items, aliases, &memMerge{items: items, aliases: aliases}, &memLinks{})
	cmd.now = func() time.Time { return now }
	require.NoError(t, cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B"}}))
	_, still := aliases.byKey["supplier_sku|P1|SKU-1-dup"]
	assert.False(t, still)
	kept := aliases.byKey["supplier_sku|P1|SKU-1"]
	assert.Equal(t, "A", kept.ItemID)
	assert.Equal(t, domain.AliasSourceManual, kept.Source)
}

func TestMergeItemsInheritsSourceInternalCode(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{
		"A": {ID: "A", Name: "Mint", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreationSource: domain.CreationSourceInvoice, CreatedAt: now, UpdatedAt: now},
		"B": testItem("B", "Master", "INT-B", now),
	}}
	aliases := &memAliases{byKey: map[string]domain.Alias{}}
	cmd := NewMergeItemsCommand(items, aliases, &memMerge{items: items, aliases: aliases}, &memLinks{})
	cmd.now = func() time.Time { return now }
	require.NoError(t, cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B"}}))
	assert.Equal(t, "INT-B", items.items["A"].InternalCode)
}

func TestMergeItemsKeepsSurvivorInternalCode(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{
		"A": testItem("A", "One", "INT-A", now),
		"B": testItem("B", "Two", "INT-B", now),
	}}
	aliases := &memAliases{byKey: map[string]domain.Alias{}}
	cmd := NewMergeItemsCommand(items, aliases, &memMerge{items: items, aliases: aliases}, &memLinks{})
	cmd.now = func() time.Time { return now }
	require.NoError(t, cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B"}}))
	assert.Equal(t, "INT-A", items.items["A"].InternalCode)
	assert.Equal(t, "", items.items["B"].InternalCode)
}

func TestMergeItemsRequiresInternalCodeChoiceWhenSurvivorHasNone(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{
		"A": {ID: "A", Name: "Mint", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreationSource: domain.CreationSourceInvoice, CreatedAt: now, UpdatedAt: now},
		"B": testItem("B", "Two", "INT-B", now),
		"C": testItem("C", "Three", "INT-C", now),
	}}
	cmd := NewMergeItemsCommand(items, &memAliases{}, &memMerge{items: items, aliases: &memAliases{}}, &memLinks{})
	err := cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B", "C"}})
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeValidation, appErr.Code)
}

func TestMergeItemsRejectsSingleSourceThatIsSurvivor(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{"A": testItem("A", "One", "INT-A", now)}}
	cmd := NewMergeItemsCommand(items, &memAliases{}, &memMerge{items: items, aliases: &memAliases{}}, &memLinks{})
	err := cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"A"}})
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeValidation, appErr.Code)
}

func TestMergeItemsDefaultsNameAndKindWhenOmitted(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{
		"A": {ID: "A", Name: "SKU", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreationSource: domain.CreationSourceInvoice, CreatedAt: now, UpdatedAt: now},
		"B": testItem("B", "Widget largo", "INT-B", now),
	}}
	cmd := NewMergeItemsCommand(items, &memAliases{byKey: map[string]domain.Alias{}}, &memMerge{items: items, aliases: &memAliases{byKey: map[string]domain.Alias{}}}, &memLinks{})
	cmd.now = func() time.Time { return now }
	require.NoError(t, cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B"}}))
	assert.Equal(t, "Widget largo", items.items["A"].Name)
	assert.Equal(t, domain.KindGoods, items.items["A"].Kind)
	assert.Equal(t, "INT-B", items.items["A"].InternalCode)
}

func TestMergeItemsIsIdempotentWhenAlreadyMerged(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	items := &memItems{items: map[string]domain.Item{
		"A": testItem("A", "Widget", "INT-A", now),
		"B": testItem("B", "Widget B", "INT-B", now),
	}}
	aliases := &memAliases{byKey: map[string]domain.Alias{}}
	cmd := NewMergeItemsCommand(items, aliases, &memMerge{items: items, aliases: aliases}, &memLinks{})
	cmd.now = func() time.Time { return now }
	code := "INT-A"
	require.NoError(t, cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B"}, InternalCode: &code}))
	require.NoError(t, cmd.Execute(context.Background(), MergeItemsInput{SurvivorID: "A", SourceIDs: []string{"B"}, InternalCode: &code}))
	assert.Equal(t, domain.StatusMerged, items.items["B"].Status)
	assert.Equal(t, "A", items.items["B"].MergedIntoID)
}
