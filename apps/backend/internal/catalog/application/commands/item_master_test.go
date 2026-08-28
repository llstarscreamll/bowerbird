package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type memWrite struct {
	items   map[string]domain.Item
	aliases map[string]domain.Alias
	fail    bool
}

func (m *memWrite) CreateItemWithAlias(ctx context.Context, item domain.Item, alias domain.Alias) error {
	if m.fail {
		return appErrors.New(appErrors.CodeInternal, "write failed")
	}
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	if m.aliases == nil {
		m.aliases = map[string]domain.Alias{}
	}
	if _, ok := m.items[item.ID]; ok {
		return appErrors.New(appErrors.CodeConflict, "duplicate item")
	}
	for _, a := range m.aliases {
		if a.Scheme == domain.AliasSchemeInternalSKU && a.Value == alias.Value {
			return appErrors.New(appErrors.CodeConflict, "duplicate sku")
		}
	}
	m.items[item.ID] = item
	m.aliases[alias.ID] = alias
	return nil
}

func (m *memWrite) UpdateItemWithOptionalAlias(ctx context.Context, item domain.Item, alias *domain.Alias) error {
	if m.fail {
		return appErrors.New(appErrors.CodeInternal, "write failed")
	}
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	m.items[item.ID] = item
	if alias != nil {
		if m.aliases == nil {
			m.aliases = map[string]domain.Alias{}
		}
		m.aliases[alias.ID] = *alias
	}
	return nil
}

func (m *memWrite) CreateItem(ctx context.Context, item domain.Item) error {
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	m.items[item.ID] = item
	return nil
}
func (m *memWrite) UpdateItem(ctx context.Context, item domain.Item) error {
	m.items[item.ID] = item
	return nil
}
func (m *memWrite) GetItemByID(ctx context.Context, id string) (*domain.Item, error) {
	if item, ok := m.items[id]; ok {
		cp := item
		return &cp, nil
	}
	return nil, nil
}
func (m *memWrite) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *memWrite) ListItems(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	return nil, nil
}
func (m *memWrite) FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error) {
	return nil, nil
}
func (m *memWrite) CreateAlias(ctx context.Context, alias domain.Alias) error {
	if m.aliases == nil {
		m.aliases = map[string]domain.Alias{}
	}
	m.aliases[alias.ID] = alias
	return nil
}
func (m *memWrite) FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error) {
	return nil, nil
}
func (m *memWrite) ListInternalSKUsByItemIDs(ctx context.Context, itemIDs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, a := range m.aliases {
		if a.Scheme != domain.AliasSchemeInternalSKU {
			continue
		}
		for _, id := range itemIDs {
			if a.ItemID == id {
				out[id] = a.Value
			}
		}
	}
	return out, nil
}

func TestCreateItemCommand(t *testing.T) {
	t.Parallel()
	store := &memWrite{}
	cmd := NewCreateItemCommand(store, store)
	cmd.now = func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }
	cmd.newID = func() string { return "01ARZ3NDEKTSV4RRFFQ69G5FB0" }

	id := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	err := cmd.Execute(context.Background(), CreateItemInput{
		ID: id, Name: "Widget", Kind: domain.KindGoods, InternalSKU: "SKU-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	item := store.items[id]
	if item.Status != domain.StatusConfirmed {
		t.Fatalf("status=%s", item.Status)
	}
	if len(store.aliases) != 1 {
		t.Fatalf("expected alias")
	}

	err = cmd.Execute(context.Background(), CreateItemInput{
		ID: id, Name: "Dup", Kind: domain.KindGoods, InternalSKU: "SKU-2",
	})
	var appErr *appErrors.AppError
	if err == nil || !errors.As(err, &appErr) || appErr.Code != appErrors.CodeConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestUpdateItemCommandConfirmRequiresSKU(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	id := "01ARZ3NDEKTSV4RRFFQ69G5FC0"
	store := &memWrite{items: map[string]domain.Item{
		id: {ID: id, Name: "Prov", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
	}}
	cmd := NewUpdateItemCommand(store, store, store)
	cmd.now = func() time.Time { return now }
	cmd.newID = func() string { return "01ARZ3NDEKTSV4RRFFQ69G5FB1" }

	status := domain.StatusConfirmed
	err := cmd.Execute(context.Background(), UpdateItemInput{ID: id, Status: &status})
	if err == nil {
		t.Fatal("expected validation error")
	}

	sku := "SKU-P"
	err = cmd.Execute(context.Background(), UpdateItemInput{ID: id, Status: &status, InternalSKU: &sku})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if store.items[id].Status != domain.StatusConfirmed {
		t.Fatalf("not confirmed")
	}
	if len(store.aliases) != 1 {
		t.Fatalf("expected sku alias")
	}

	other := "SKU-OTHER"
	err = cmd.Execute(context.Background(), UpdateItemInput{ID: id, InternalSKU: &other})
	if err == nil {
		t.Fatal("expected immutable sku error")
	}
}

func TestUpdateItemCommandProvisionalStatusNoOpWhenAlreadyProvisional(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	id := "01ARZ3NDEKTSV4RRFFQ69G5FD0"
	store := &memWrite{items: map[string]domain.Item{
		id: {ID: id, Name: "Prov", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
	}}
	cmd := NewUpdateItemCommand(store, store, store)
	cmd.now = func() time.Time { return now }

	status := domain.StatusProvisional
	name := "Prov Renamed"
	err := cmd.Execute(context.Background(), UpdateItemInput{ID: id, Name: &name, Status: &status})
	if err != nil {
		t.Fatalf("expected no-op provisional status, got %v", err)
	}
	if store.items[id].Name != "Prov Renamed" {
		t.Fatalf("name not updated")
	}
	if store.items[id].Status != domain.StatusProvisional {
		t.Fatalf("status changed unexpectedly")
	}
}
