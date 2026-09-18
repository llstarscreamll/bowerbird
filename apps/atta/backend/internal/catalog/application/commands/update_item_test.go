package commands

import (
	"context"
	"testing"
	"time"

	"github.com/atta/internal/catalog/domain"
)

func TestUpdateItemCommandConfirmRequiresCode(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	id := "01ARZ3NDEKTSV4RRFFQ69G5FC0"
	store := &memWrite{items: map[string]domain.Item{
		id: {ID: id, Name: "Prov", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
	}}
	cmd := NewUpdateItemCommand(store)
	cmd.now = func() time.Time { return now }

	status := domain.StatusConfirmed
	err := cmd.Execute(context.Background(), UpdateItemInput{ID: id, Status: &status})
	if err == nil {
		t.Fatal("expected validation error")
	}

	code := "CODE-P"
	err = cmd.Execute(context.Background(), UpdateItemInput{ID: id, Status: &status, InternalCode: &code})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if store.items[id].Status != domain.StatusConfirmed {
		t.Fatalf("not confirmed")
	}
	if store.items[id].InternalCode != "CODE-P" {
		t.Fatalf("internal_code=%s", store.items[id].InternalCode)
	}

	other := "CODE-OTHER"
	err = cmd.Execute(context.Background(), UpdateItemInput{ID: id, InternalCode: &other})
	if err == nil {
		t.Fatal("expected immutable internal code error")
	}
}

func TestUpdateItemCommandProvisionalStatusNoOpWhenAlreadyProvisional(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	id := "01ARZ3NDEKTSV4RRFFQ69G5FD0"
	store := &memWrite{items: map[string]domain.Item{
		id: {ID: id, Name: "Prov", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now},
	}}
	cmd := NewUpdateItemCommand(store)
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
