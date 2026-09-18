package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atta/internal/catalog/application"
	"github.com/atta/internal/catalog/application/commands"
	"github.com/atta/internal/catalog/application/ports"
	"github.com/atta/internal/catalog/application/queries"
	"github.com/atta/internal/catalog/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type httpMemMerge struct {
	items   *httpMemItems
	aliases *httpMemAliases
}

func (m *httpMemMerge) ApplyMerge(ctx context.Context, in ports.MergePersistence) error {
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
		}
	}
	for _, alias := range in.ReassignAliases {
		party := ""
		if alias.PartyID != nil {
			party = *alias.PartyID
		}
		m.aliases.byKey[m.aliases.key(alias.Scheme, party, alias.Value)] = alias
	}
	return nil
}

type httpMemPairs struct {
	pairs []domain.NotDuplicatePair
}

func (m *httpMemPairs) UpsertNotDuplicatePairs(ctx context.Context, pairs []domain.NotDuplicatePair) error {
	m.pairs = append(m.pairs, pairs...)
	return nil
}
func (m *httpMemPairs) ListNotDuplicatePairs(ctx context.Context) ([]domain.NotDuplicatePair, error) {
	return m.pairs, nil
}

type httpMemIndex struct {
	items *httpMemItems
}

func (m *httpMemIndex) DescriptionDuplicateGroups(ctx context.Context) ([][]domain.Item, error) {
	byNorm := map[string][]domain.Item{}
	for _, item := range m.items.items {
		if item.IsMerged() {
			continue
		}
		norm := domain.NormalizeDescription(item.Name)
		if norm == "" {
			continue
		}
		byNorm[norm] = append(byNorm[norm], item)
	}
	out := make([][]domain.Item, 0)
	for _, group := range byNorm {
		if len(group) > 1 {
			out = append(out, group)
		}
	}
	return out, nil
}
func (m *httpMemIndex) CrossPartySKUGroups(ctx context.Context) ([][]domain.Item, error) {
	return nil, nil
}

type httpMemLinks struct{}

func (httpMemLinks) RelinkItems(ctx context.Context, fromIDs []string, toID string) error { return nil }
func (httpMemLinks) HardConflictPairs(ctx context.Context) ([]ports.ItemIDPair, error) {
	return nil, nil
}
func (httpMemLinks) CountLinks(ctx context.Context, itemIDs []string) (map[string]int, error) {
	return map[string]int{}, nil
}

func mergeHTTPApp(now time.Time) (*Controller, *httpMemItems, *httpMemAliases) {
	items := &httpMemItems{items: map[string]domain.Item{
		"ITEM-A": {ID: "ITEM-A", Name: "Same Name", Kind: domain.KindGoods, Status: domain.StatusConfirmed, CreationSource: domain.CreationSourceManual, InternalCode: "INT-A", CreatedAt: now, UpdatedAt: now},
		"ITEM-B": {ID: "ITEM-B", Name: "Same Name", Kind: domain.KindGoods, Status: domain.StatusConfirmed, CreationSource: domain.CreationSourceManual, InternalCode: "INT-B", CreatedAt: now, UpdatedAt: now},
	}}
	aliases := &httpMemAliases{byKey: map[string]domain.Alias{}}
	pairs := &httpMemPairs{}
	index := &httpMemIndex{items: items}
	merge := &httpMemMerge{items: items, aliases: aliases}
	app := &application.Application{
		Commands: application.Commands{
			MergeItems:        commands.NewMergeItemsCommand(items, aliases, merge, httpMemLinks{}),
			MarkNotDuplicates: commands.NewMarkNotDuplicatesCommand(items, pairs),
		},
		Queries: application.Queries{
			GetItemByID:           queries.NewGetItemByIDQuery(items, aliases),
			ListDuplicateClusters: queries.NewListDuplicateClustersQuery(items, index, aliases, pairs, httpMemLinks{}),
		},
	}
	return NewController(app), items, aliases
}

func TestCatalogMergeHTTP(t *testing.T) {
	now := time.Now().UTC()
	ctrl, items, aliases := mergeHTTPApp(now)
	party := "P1"
	aliases.byKey["supplier_sku|P1|SKU-B"] = domain.Alias{
		ID: "AL-B", ItemID: "ITEM-B", Scheme: domain.AliasSchemeSupplierSKU, Value: "SKU-B", PartyID: &party, Source: domain.AliasSourceManual,
	}

	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type": "catalog_item_merges",
			"attributes": map[string]any{
				"survivor_id":   "ITEM-A",
				"source_ids":    []string{"ITEM-B"},
				"internal_code": "INT-A",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/items/merges", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	require.NoError(t, ctrl.MergeItems(rr, req))
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":"ITEM-A"`)
	assert.Contains(t, rr.Body.String(), `"SKU-B"`)
	assert.Equal(t, domain.StatusMerged, items.items["ITEM-B"].Status)

	goneReq := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/items/ITEM-B", nil)
	goneReq.SetPathValue("id", "ITEM-B")
	goneRR := httptest.NewRecorder()
	err := ctrl.GetItem(goneRR, goneReq)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeGone, appErr.Code)
	assert.Equal(t, "ITEM-A", appErr.Meta["merged_into_id"])
}

func TestCatalogDuplicateClustersHTTP(t *testing.T) {
	now := time.Now().UTC()
	ctrl, _, _ := mergeHTTPApp(now)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/items/duplicate-clusters", nil)
	rr := httptest.NewRecorder()
	require.NoError(t, ctrl.ListDuplicateClusters(rr, req))
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"type":"catalog_duplicate_clusters"`)
	assert.Contains(t, rr.Body.String(), `"normalized_description"`)
	assert.Contains(t, rr.Body.String(), `"ITEM-A"`)
	assert.Contains(t, rr.Body.String(), `"ITEM-B"`)
}

func TestCatalogNotDuplicatesHTTP(t *testing.T) {
	now := time.Now().UTC()
	ctrl, _, _ := mergeHTTPApp(now)
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "catalog_item_not_duplicates",
			"attributes": map[string]any{"item_ids": []string{"ITEM-A", "ITEM-B"}},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/items/not-duplicates", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	require.NoError(t, ctrl.MarkNotDuplicates(rr, req))
	assert.Equal(t, http.StatusNoContent, rr.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/items/duplicate-clusters", nil)
	listRR := httptest.NewRecorder()
	require.NoError(t, ctrl.ListDuplicateClusters(listRR, listReq))
	assert.Equal(t, http.StatusOK, listRR.Code)
	assert.NotContains(t, listRR.Body.String(), `"ITEM-A"`)
}

func TestCatalogMergeHTTPRejectsEmptySources(t *testing.T) {
	now := time.Now().UTC()
	ctrl, _, _ := mergeHTTPApp(now)
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "catalog_item_merges",
			"attributes": map[string]any{"survivor_id": "ITEM-A", "source_ids": []string{}},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/catalog/items/merges", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	err := ctrl.MergeItems(rr, req)
	require.Error(t, err)
	var appErr *appErrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, appErrors.CodeValidation, appErr.Code)
}
