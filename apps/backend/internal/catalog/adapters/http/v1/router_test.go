package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bowerbird/internal/catalog/application"
	"github.com/bowerbird/internal/catalog/application/commands"
	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/application/queries"
	"github.com/bowerbird/internal/catalog/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCatalog struct {
	items map[string]domain.Item
	links map[string]ports.LineLinkState
	mem   map[string]domain.MatchMemory
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
func (s *stubCatalog) UpsertMemory(ctx context.Context, memory domain.MatchMemory) error {
	s.mem[memory.EvidenceKey] = memory
	return nil
}
func (s *stubCatalog) FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error) {
	m, ok := s.mem[evidenceKey]
	if !ok {
		return nil, nil
	}
	cp := m
	return &cp, nil
}
func (s *stubCatalog) UpdateLineLink(ctx context.Context, lineID string, itemID *string, status, method string, locked bool, suggestions []domain.Suggestion) error {
	st := s.links[lineID]
	if itemID != nil {
		st.ItemID = *itemID
	} else {
		st.ItemID = ""
	}
	st.LinkStatus = status
	st.LinkMethod = method
	st.LinkLocked = locked
	s.links[lineID] = st
	return nil
}
func (s *stubCatalog) ListReviewLines(ctx context.Context, statuses []string) ([]ports.ReviewLine, error) {
	return nil, nil
}
func (s *stubCatalog) GetLineLinkState(ctx context.Context, lineID string) (*ports.LineLinkState, error) {
	st, ok := s.links[lineID]
	if !ok {
		return nil, nil
	}
	cp := st
	return &cp, nil
}
func (s *stubCatalog) SyncHeaderLinkingStatus(ctx context.Context, invoiceHeaderID string) error {
	return nil
}

func TestRememberLineDecision_LocksAndRemembers(t *testing.T) {
	now := time.Now().UTC()
	stub := &stubCatalog{
		items: map[string]domain.Item{"ITEM-1": {ID: "ITEM-1", Name: "Widget", Kind: domain.KindUnknown, Status: domain.StatusConfirmed, CreatedAt: now, UpdatedAt: now}},
		links: map[string]ports.LineLinkState{
			"LINE-1": {LineID: "LINE-1", PartyID: "P1", ItemCode: "SKU-1", Description: "Widget", LinkStatus: domain.LinkStatusUnmatched},
		},
		mem: map[string]domain.MatchMemory{},
	}
	remember := commands.NewRememberDecisionCommand(stub, stub, stub, stub)
	app := &application.Application{
		Commands: application.Commands{RememberDecision: remember},
		Queries: application.Queries{
			GetItemByID:     queries.NewGetItemByIDQuery(stub),
			GetItemNames:    queries.NewGetItemNamesQuery(stub),
			ListItems:       queries.NewListItemsQuery(stub),
			ListReviewQueue: queries.NewListReviewQueueQuery(stub),
		},
	}
	ctrl := NewController(app)
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"attributes": map[string]any{
				"item_id":  "ITEM-1",
				"action":   "link",
				"remember": true,
				"lock":     true,
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/lines/LINE-1/decisions", bytes.NewReader(body))
	req.SetPathValue("lineId", "LINE-1")
	rr := httptest.NewRecorder()
	err := ctrl.RememberLineDecision(rr, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.True(t, stub.links["LINE-1"].LinkLocked)
	assert.Equal(t, "ITEM-1", stub.links["LINE-1"].ItemID)
	assert.NotEmpty(t, stub.mem)
}
