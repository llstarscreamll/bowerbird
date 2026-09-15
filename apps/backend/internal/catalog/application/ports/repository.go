package ports

import (
	"context"

	"github.com/bowerbird/internal/catalog/domain"
)

type ItemRepository interface {
	CreateItem(ctx context.Context, item domain.Item) error
	UpdateItem(ctx context.Context, item domain.Item) error
	GetItemByID(ctx context.Context, id string) (*domain.Item, error)
	GetItemNames(ctx context.Context, ids []string) (map[string]string, error)
	GetItemsByIDs(ctx context.Context, ids []string) ([]domain.Item, error)
	ListItems(ctx context.Context, filter ItemListFilter) ([]domain.Item, error)
	FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error)
}

type ItemListFilter struct {
	Kind           string
	Status         string
	Search         string
	CreationSource string
}

type AliasRepository interface {
	CreateAlias(ctx context.Context, alias domain.Alias) error
	FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error)
}

// CatalogWriteRepository persists an Item together with a supplier alias in one TX
// (provisional mint on invoice ingest).
type CatalogWriteRepository interface {
	CreateItemWithAlias(ctx context.Context, item domain.Item, alias domain.Alias) error
}

type MatchMemoryRepository interface {
	UpsertMemory(ctx context.Context, memory domain.MatchMemory) error
	FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error)
}

type SoftMatcher interface {
	Match(ctx context.Context, description string) ([]domain.Suggestion, error)
}
