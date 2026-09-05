package commands

import (
	"context"
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/domain"
)

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
