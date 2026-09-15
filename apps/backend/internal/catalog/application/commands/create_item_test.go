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
	items map[string]domain.Item
	fail  bool
}

func (m *memWrite) CreateItem(ctx context.Context, item domain.Item) error {
	if m.fail {
		return appErrors.New(appErrors.CodeInternal, "write failed")
	}
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	if _, ok := m.items[item.ID]; ok {
		return appErrors.New(appErrors.CodeConflict, "a catalog item with this id already exists")
	}
	if item.InternalCode != "" {
		for _, existing := range m.items {
			if existing.InternalCode == item.InternalCode {
				return appErrors.New(appErrors.CodeConflict, "an item with this internal code already exists")
			}
		}
	}
	m.items[item.ID] = item
	return nil
}
func (m *memWrite) UpdateItem(ctx context.Context, item domain.Item) error {
	if m.fail {
		return appErrors.New(appErrors.CodeInternal, "write failed")
	}
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
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
func (m *memWrite) GetItemsByIDs(ctx context.Context, ids []string) ([]domain.Item, error) {
	return nil, nil
}
func (m *memWrite) ListItems(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	return nil, nil
}
func (m *memWrite) FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error) {
	return nil, nil
}

func TestCreateItemCommand(t *testing.T) {
	t.Parallel()
	store := &memWrite{}
	cmd := NewCreateItemCommand(store)
	cmd.now = func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }

	id := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	err := cmd.Execute(context.Background(), CreateItemInput{
		ID: id, Name: "Widget", Kind: domain.KindGoods, InternalCode: "CODE-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	item := store.items[id]
	if item.Status != domain.StatusConfirmed {
		t.Fatalf("status=%s", item.Status)
	}
	if item.InternalCode != "CODE-1" {
		t.Fatalf("internal_code=%s", item.InternalCode)
	}

	err = cmd.Execute(context.Background(), CreateItemInput{
		ID: id, Name: "Dup", Kind: domain.KindGoods, InternalCode: "CODE-2",
	})
	var appErr *appErrors.AppError
	if err == nil || !errors.As(err, &appErr) || appErr.Code != appErrors.CodeConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}
