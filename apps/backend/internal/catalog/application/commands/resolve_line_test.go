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

type memItems struct {
	items map[string]domain.Item
}

func (m *memItems) CreateItem(ctx context.Context, item domain.Item) error {
	if m.items == nil {
		m.items = map[string]domain.Item{}
	}
	m.items[item.ID] = item
	return nil
}
func (m *memItems) UpdateItem(ctx context.Context, item domain.Item) error {
	m.items[item.ID] = item
	return nil
}
func (m *memItems) GetItemByID(ctx context.Context, id string) (*domain.Item, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	cp := item
	return &cp, nil
}
func (m *memItems) ListItems(ctx context.Context, filter ports.ItemListFilter) ([]domain.Item, error) {
	out := make([]domain.Item, 0, len(m.items))
	for _, i := range m.items {
		out = append(out, i)
	}
	return out, nil
}
func (m *memItems) FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error) {
	out := []domain.Item{}
	for _, i := range m.items {
		if domain.NormalizeDescription(i.Name) == normalizedDesc {
			out = append(out, i)
		}
	}
	return out, nil
}

type memAliases struct {
	byKey map[string]domain.Alias
}

func aliasKey(scheme, partyID, value string) string {
	return scheme + "|" + partyID + "|" + value
}

func (m *memAliases) CreateAlias(ctx context.Context, alias domain.Alias) error {
	if m.byKey == nil {
		m.byKey = map[string]domain.Alias{}
	}
	party := ""
	if alias.PartyID != nil {
		party = *alias.PartyID
	}
	key := aliasKey(alias.Scheme, party, alias.Value)
	if _, exists := m.byKey[key]; exists {
		return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
	}
	m.byKey[key] = alias
	return nil
}
func (m *memAliases) FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error) {
	a, ok := m.byKey[aliasKey(scheme, partyID, value)]
	if !ok {
		return nil, nil
	}
	cp := a
	return &cp, nil
}

type memMemories struct {
	byKey map[string]domain.MatchMemory
}

func (m *memMemories) UpsertMemory(ctx context.Context, memory domain.MatchMemory) error {
	if m.byKey == nil {
		m.byKey = map[string]domain.MatchMemory{}
	}
	m.byKey[memory.EvidenceKey] = memory
	return nil
}
func (m *memMemories) FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error) {
	mem, ok := m.byKey[evidenceKey]
	if !ok {
		return nil, nil
	}
	cp := mem
	return &cp, nil
}

type stubMatcher struct {
	suggestions []domain.Suggestion
}

func (s *stubMatcher) Match(ctx context.Context, description string) ([]domain.Suggestion, error) {
	return s.suggestions, nil
}

func newResolveCmd(items *memItems, aliases ports.AliasRepository, memories *memMemories, matcher ports.SoftMatcher) *ResolveInvoiceLineCommand {
	cmd := NewResolveInvoiceLineCommand(items, aliases, memories, matcher)
	n := 0
	cmd.newID = func() string {
		n++
		return "id-" + string(rune('A'+n-1))
	}
	return cmd
}

func TestResolve_LockedPreserved(t *testing.T) {
	cmd := newResolveCmd(&memItems{}, &memAliases{}, &memMemories{}, nil)
	res, err := cmd.Execute(context.Background(), domain.LineResolutionInput{
		ExistingLocked: true,
		ExistingItemID: "ITEM-1",
		ExistingMethod: domain.LinkMethodManual,
		ItemCode:       "X",
		PartyID:        "P",
	})
	require.NoError(t, err)
	assert.Equal(t, "ITEM-1", res.ItemID)
	assert.Equal(t, domain.LinkMethodManual, res.Method)
}

func TestResolve_MemoryLink(t *testing.T) {
	memories := &memMemories{byKey: map[string]domain.MatchMemory{}}
	itemID := "ITEM-MEM"
	key := domain.EvidenceKey("P", "ABC", domain.DescriptionFingerprint("Widget"), domain.EvidenceKindCodeDescription)
	memories.byKey[key] = domain.MatchMemory{EvidenceKey: key, Action: domain.MemoryActionLink, ItemID: &itemID}
	cmd := newResolveCmd(&memItems{}, &memAliases{}, memories, nil)
	res, err := cmd.Execute(context.Background(), domain.LineResolutionInput{
		PartyID: "P", ItemCode: "ABC", Description: "Widget",
	})
	require.NoError(t, err)
	assert.Equal(t, itemID, res.ItemID)
	assert.Equal(t, domain.LinkMethodMemory, res.Method)
}

func TestResolve_HardAlias(t *testing.T) {
	aliases := &memAliases{byKey: map[string]domain.Alias{
		aliasKey(domain.AliasSchemeSupplierSKU, "P", "SKU-1"): {ItemID: "ITEM-HARD", Scheme: domain.AliasSchemeSupplierSKU, Value: "SKU-1"},
	}}
	cmd := newResolveCmd(&memItems{}, aliases, &memMemories{}, nil)
	res, err := cmd.Execute(context.Background(), domain.LineResolutionInput{PartyID: "P", ItemCode: "SKU-1", Description: "Thing"})
	require.NoError(t, err)
	assert.Equal(t, "ITEM-HARD", res.ItemID)
	assert.Equal(t, domain.LinkMethodHard, res.Method)
	assert.False(t, res.Minted)
}

func TestResolve_SoftSuggestOnlyNoAutoLink(t *testing.T) {
	items := &memItems{items: map[string]domain.Item{"I1": {ID: "I1", Name: "MacBook Air"}}}
	matcher := &stubMatcher{suggestions: []domain.Suggestion{{ItemID: "I1", Score: 1, Reason: "test"}}}
	cmd := newResolveCmd(items, &memAliases{}, &memMemories{}, matcher)
	res, err := cmd.Execute(context.Background(), domain.LineResolutionInput{
		Description: "MacBook Air",
		// no party/code → no mint
	})
	require.NoError(t, err)
	assert.Equal(t, domain.LinkStatusSuggested, res.Status)
	assert.Empty(t, res.ItemID)
	assert.Len(t, res.Suggestions, 1)
}

func TestResolve_ProvisionalMintOnHardMiss(t *testing.T) {
	items := &memItems{}
	aliases := &memAliases{}
	cmd := newResolveCmd(items, aliases, &memMemories{}, nil)
	ids := []string{"ITEM-NEW", "ALIAS-NEW"}
	i := 0
	cmd.newID = func() string {
		id := ids[i]
		i++
		return id
	}
	res, err := cmd.Execute(context.Background(), domain.LineResolutionInput{
		PartyID: "P", ItemCode: "NEW-1", Description: "New Product",
	})
	require.NoError(t, err)
	assert.True(t, res.Minted)
	assert.Equal(t, "ITEM-NEW", res.ItemID)
	assert.Equal(t, domain.LinkMethodHard, res.Method)
	assert.Len(t, items.items, 1)
}

func TestResolve_EmptyCodeNoMint(t *testing.T) {
	cmd := newResolveCmd(&memItems{}, &memAliases{}, &memMemories{}, nil)
	res, err := cmd.Execute(context.Background(), domain.LineResolutionInput{
		PartyID: "P", ItemCode: "", Description: "Service fee",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.LinkStatusUnmatched, res.Status)
	assert.Empty(t, res.ItemID)
	assert.False(t, res.Minted)
}

// raceAliases: Find misses on the initial hard lookup, then CreateAlias conflicts
// and Find returns the concurrent winner (simulates unique-constraint race).
type raceAliases struct {
	finds  int
	winner domain.Alias
}

func (r *raceAliases) CreateAlias(ctx context.Context, alias domain.Alias) error {
	return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists")
}
func (r *raceAliases) FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error) {
	r.finds++
	if r.finds == 1 {
		return nil, nil
	}
	cp := r.winner
	return &cp, nil
}

func TestResolve_ProvisionalMintAliasRaceReturnsWinner(t *testing.T) {
	now := time.Now().UTC()
	winner := domain.Item{ID: "ITEM-WIN", Name: "Winner", Kind: domain.KindUnknown, Status: domain.StatusProvisional, CreatedAt: now, UpdatedAt: now}
	items := &memItems{items: map[string]domain.Item{"ITEM-WIN": winner}}
	aliases := &raceAliases{winner: domain.Alias{
		ID: "ALIAS-WIN", ItemID: "ITEM-WIN", Scheme: domain.AliasSchemeSupplierSKU, Value: "SKU-RACE",
	}}
	cmd := newResolveCmd(items, aliases, &memMemories{}, nil)
	n := 0
	cmd.newID = func() string {
		n++
		if n == 1 {
			return "ITEM-ORPHAN"
		}
		return "ALIAS-ORPHAN"
	}

	res, err := cmd.Execute(context.Background(), domain.LineResolutionInput{
		PartyID: "P", ItemCode: "SKU-RACE", Description: "Race Product",
	})
	require.NoError(t, err)
	assert.False(t, res.Minted)
	assert.Equal(t, "ITEM-WIN", res.ItemID)
	assert.Equal(t, domain.LinkMethodHard, res.Method)
	assert.Contains(t, items.items, "ITEM-WIN")
	assert.Contains(t, items.items, "ITEM-ORPHAN") // losing mint's orphan item
}
