package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/application"
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/application/queries"
	"github.com/bowerbird/internal/catalog/domain"
	"github.com/bowerbird/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCatalog struct {
	items map[string]domain.Item
}

func (s *stubCatalog) CreateItem(ctx context.Context, item domain.Item) error {
	s.items[item.ID] = item
	return nil
}
func (s *stubCatalog) UpdateItem(ctx context.Context, item domain.Item) error {
	s.items[item.ID] = item
	return nil
}
func (s *stubCatalog) GetItemByID(ctx context.Context, id string) (*domain.Item, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	cp := item
	return &cp, nil
}
func (s *stubCatalog) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if item, ok := s.items[id]; ok {
			out[id] = item.Name
		}
	}
	return out, nil
}
func (s *stubCatalog) ListItems(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	out := []domain.Item{}
	for _, i := range s.items {
		out = append(out, i)
	}
	return out, nil
}
func (s *stubCatalog) FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error) {
	return nil, nil
}
func (s *stubCatalog) CreateAlias(ctx context.Context, alias domain.Alias) error { return nil }
func (s *stubCatalog) FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error) {
	return nil, nil
}
func (s *stubCatalog) ListInternalSKUsByItemIDs(ctx context.Context, itemIDs []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (s *stubCatalog) CreateItemWithAlias(ctx context.Context, item domain.Item, alias domain.Alias) error {
	s.items[item.ID] = item
	return nil
}
func (s *stubCatalog) UpdateItemWithOptionalAlias(ctx context.Context, item domain.Item, alias *domain.Alias) error {
	s.items[item.ID] = item
	return nil
}
func (s *stubCatalog) UpsertMemory(ctx context.Context, memory domain.MatchMemory) error { return nil }
func (s *stubCatalog) FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error) {
	return nil, nil
}

func TestCatalogRoutes_NoReviewQueue(t *testing.T) {
	now := time.Now().UTC()
	stub := &stubCatalog{
		items: map[string]domain.Item{"ITEM-1": {ID: "ITEM-1", Name: "Widget", Kind: domain.KindUnknown, Status: domain.StatusConfirmed, CreatedAt: now, UpdatedAt: now}},
	}
	app := &application.Application{
		Commands: application.Commands{
			CreateItem: commands.NewCreateItemCommand(stub, stub),
		},
		Queries: application.Queries{
			GetItemByID:  queries.NewGetItemByIDQuery(stub, stub),
			GetItemNames: queries.NewGetItemNamesQuery(stub),
			ListItems:    queries.NewListItemsQuery(stub, stub),
		},
	}
	ctrl := NewController(app)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/items", nil)
	rr := httptest.NewRecorder()
	err := ctrl.ListItems(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rr.Code)

	reviewReq := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/review-queue", nil)
	reviewRR := httptest.NewRecorder()
	mux := http.NewServeMux()
	NewRouter(ctrl).Register(mux, config.Config{}, func(next http.Handler) http.Handler { return next })
	mux.ServeHTTP(reviewRR, reviewReq)
	assert.Equal(t, http.StatusNotFound, reviewRR.Code)
}
