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
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type httpMemItems struct{ items map[string]domain.Item }

func (m *httpMemItems) CreateItem(ctx context.Context, item domain.Item) error {
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	m.items[item.ID] = item
	return nil
}
func (m *httpMemItems) UpdateItem(ctx context.Context, item domain.Item) error {
	m.items[item.ID] = item
	return nil
}
func (m *httpMemItems) GetItemByID(ctx context.Context, id string) (*domain.Item, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	cp := item
	return &cp, nil
}
func (m *httpMemItems) GetItemNames(ctx context.Context, ids []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *httpMemItems) GetItemsByIDs(ctx context.Context, ids []string) ([]domain.Item, error) {
	return nil, nil
}
func (m *httpMemItems) GetItemsByInternalCodes(ctx context.Context, codes []string) ([]domain.Item, error) {
	return nil, nil
}
func (m *httpMemItems) CreateItems(ctx context.Context, items []domain.Item) error { return nil }
func (m *httpMemItems) UpdateItems(ctx context.Context, items []domain.Item) error { return nil }
func (m *httpMemItems) ListItems(ctx context.Context, filter ports.ItemListFilter) (ports.ItemListPage, error) {
	return ports.ItemListPage{}, nil
}
func (m *httpMemItems) FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error) {
	return nil, nil
}

type httpMemAliases struct{ byKey map[string]domain.Alias }

func (m *httpMemAliases) key(scheme, partyID, value string) string {
	return scheme + "|" + partyID + "|" + value
}
func (m *httpMemAliases) CreateAlias(ctx context.Context, alias domain.Alias) error {
	if m.byKey == nil {
		m.byKey = map[string]domain.Alias{}
	}
	party := ""
	if alias.PartyID != nil {
		party = *alias.PartyID
	}
	key := m.key(alias.Scheme, party, alias.Value)
	if existing, ok := m.byKey[key]; ok {
		return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item").
			WithMeta("item_id", existing.ItemID)
	}
	m.byKey[key] = alias
	return nil
}
func (m *httpMemAliases) FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error) {
	a, ok := m.byKey[m.key(scheme, partyID, value)]
	if !ok {
		return nil, nil
	}
	cp := a
	return &cp, nil
}
func (m *httpMemAliases) ListAliasesByItemID(ctx context.Context, itemID string) ([]domain.Alias, error) {
	out := []domain.Alias{}
	for _, a := range m.byKey {
		if a.ItemID == itemID {
			out = append(out, a)
		}
	}
	return out, nil
}
func (m *httpMemAliases) DeleteAlias(ctx context.Context, itemID, aliasID string) error {
	for key, a := range m.byKey {
		if a.ID == aliasID && a.ItemID == itemID {
			delete(m.byKey, key)
			return nil
		}
	}
	return appErrors.New(appErrors.CodeNotFound, "alias not found")
}

func TestCatalogAliasHTTP(t *testing.T) {
	now := time.Now().UTC()
	items := &httpMemItems{items: map[string]domain.Item{
		"ITEM-1": {ID: "ITEM-1", Name: "Widget", Kind: domain.KindGoods, Status: domain.StatusConfirmed, CreationSource: domain.CreationSourceManual, CreatedAt: now, UpdatedAt: now},
		"ITEM-J": {ID: "ITEM-J", Name: "Other", Kind: domain.KindGoods, Status: domain.StatusConfirmed, CreationSource: domain.CreationSourceManual, CreatedAt: now, UpdatedAt: now},
	}}
	party := "P1"
	aliases := &httpMemAliases{byKey: map[string]domain.Alias{}}
	app := &application.Application{
		Commands: application.Commands{
			AddItemAlias:    commands.NewAddItemAliasCommand(items, aliases),
			RemoveItemAlias: commands.NewRemoveItemAliasCommand(items, aliases),
		},
		Queries: application.Queries{
			GetItemByID: queries.NewGetItemByIDQuery(items, aliases),
		},
	}
	ctrl := NewController(app)

	aliasID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type": "catalog_item_aliases",
			"id":   aliasID,
			"attributes": map[string]any{
				"scheme":   "supplier_sku",
				"value":    "ABC-1",
				"party_id": "P1",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/items/ITEM-1/aliases", bytes.NewReader(body))
	req.SetPathValue("id", "ITEM-1")
	rr := httptest.NewRecorder()
	require.NoError(t, ctrl.AddItemAlias(rr, req))
	assert.Equal(t, http.StatusCreated, rr.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/items/ITEM-1", nil)
	getReq.SetPathValue("id", "ITEM-1")
	getRR := httptest.NewRecorder()
	require.NoError(t, ctrl.GetItem(getRR, getReq))
	assert.Equal(t, http.StatusOK, getRR.Code)
	assert.Contains(t, getRR.Body.String(), `"scheme":"supplier_sku"`)

	aliases.byKey["supplier_sku|P1|DUP"] = domain.Alias{ID: "OLD", ItemID: "ITEM-J", Scheme: domain.AliasSchemeSupplierSKU, Value: "DUP", PartyID: &party, Source: domain.AliasSourceManual}
	conflictBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "catalog_item_aliases",
			"id":         "01ARZ3NDEKTSV4RRFFQ69G5FAW",
			"attributes": map[string]any{"scheme": "supplier_sku", "value": "DUP", "party_id": "P1"},
		},
	})
	conflictReq := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/items/ITEM-1/aliases", bytes.NewReader(conflictBody))
	conflictReq.SetPathValue("id", "ITEM-1")
	conflictRR := httptest.NewRecorder()
	err := ctrl.AddItemAlias(conflictRR, conflictReq)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeConflict, appErr.Code)
	assert.Equal(t, "ITEM-J", appErr.Meta["item_id"])

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/catalog/items/ITEM-1/aliases/"+aliasID, nil)
	delReq.SetPathValue("id", "ITEM-1")
	delReq.SetPathValue("aliasId", aliasID)
	delRR := httptest.NewRecorder()
	require.NoError(t, ctrl.RemoveItemAlias(delRR, delReq))
	assert.Equal(t, http.StatusNoContent, delRR.Code)
}
