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
	ListInternalSKUsByItemIDs(ctx context.Context, itemIDs []string) (map[string]string, error)
}

// CatalogWriteRepository is the persistence ACL for Item + canonical InternalSKU.
// Internal SKU is an Alias (scheme=internal_sku), not a column on catalog_items;
// create/update must keep Item and that alias in one transaction so domain
// invariants (required SKU on manual create / confirm, SKU immutability) hold.
type CatalogWriteRepository interface {
	CreateItemWithAlias(ctx context.Context, item domain.Item, alias domain.Alias) error
	UpdateItemWithOptionalAlias(ctx context.Context, item domain.Item, alias *domain.Alias) error
}

type MatchMemoryRepository interface {
	UpsertMemory(ctx context.Context, memory domain.MatchMemory) error
	FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error)
}

type SoftMatcher interface {
	Match(ctx context.Context, description string) ([]domain.Suggestion, error)
}
