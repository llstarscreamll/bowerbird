package ports

import (
	"context"
	"time"

	"github.com/atta/internal/catalog/domain"
)

type ItemRepository interface {
	CreateItem(ctx context.Context, item domain.Item) error
	UpdateItem(ctx context.Context, item domain.Item) error
	GetItemByID(ctx context.Context, id string) (*domain.Item, error)
	GetItemNames(ctx context.Context, ids []string) (map[string]string, error)
	GetItemsByIDs(ctx context.Context, ids []string) ([]domain.Item, error)
	GetItemsByInternalCodes(ctx context.Context, codes []string) ([]domain.Item, error)
	CreateItems(ctx context.Context, items []domain.Item) error
	UpdateItems(ctx context.Context, items []domain.Item) error
	ListItems(ctx context.Context, filter ItemListFilter) (ItemListPage, error)
	FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error)
}

type ItemListFilter struct {
	Kind           string
	Status         string
	Search         string
	CreationSource string
	Limit          int
	AfterName      string
	AfterID        string
}

type ItemListPage struct {
	Items   []domain.Item
	HasMore bool
}

type AliasRepository interface {
	CreateAlias(ctx context.Context, alias domain.Alias) error
	FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error)
	ListAliasesByItemID(ctx context.Context, itemID string) ([]domain.Alias, error)
	DeleteAlias(ctx context.Context, itemID, aliasID string) error
}

// CatalogWriteRepository persists an Item together with aliases in one TX
// (provisional mint on invoice ingest) and teaching (aliases + memory).
type CatalogWriteRepository interface {
	CreateItemWithAlias(ctx context.Context, item domain.Item, alias domain.Alias) error
	CreateItemWithAliases(ctx context.Context, item domain.Item, aliases []domain.Alias) error
	RememberDecision(ctx context.Context, aliases []domain.Alias, memory domain.MatchMemory) error
}

type MatchMemoryRepository interface {
	UpsertMemory(ctx context.Context, memory domain.MatchMemory) error
	FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error)
}

type ItemIDPair struct {
	Left  string
	Right string
}

type ItemLinkSupport interface {
	RelinkItems(ctx context.Context, fromIDs []string, toID string) error
	HardConflictPairs(ctx context.Context) ([]ItemIDPair, error)
	CountLinks(ctx context.Context, itemIDs []string) (map[string]int, error)
}

type MergePersistence struct {
	Survivor        domain.Item
	Merged          []domain.Item
	ReassignAliases []domain.Alias
	DeleteAliasIDs  []string
}

type ItemMergeRepository interface {
	ApplyMerge(ctx context.Context, in MergePersistence) error
}

type DuplicateIndex interface {
	DescriptionDuplicateGroups(ctx context.Context) ([][]domain.Item, error)
	CrossPartySKUGroups(ctx context.Context) ([][]domain.Item, error)
}

type NotDuplicateRepository interface {
	UpsertNotDuplicatePairs(ctx context.Context, pairs []domain.NotDuplicatePair) error
	ListNotDuplicatePairs(ctx context.Context) ([]domain.NotDuplicatePair, error)
}

type SoftMatcher interface {
	Match(ctx context.Context, description string) ([]domain.Suggestion, error)
}

type ImportListFilter struct {
	Limit        int
	AfterCreated time.Time
	AfterID      string
}

type ImportListPage struct {
	Items   []domain.CatalogImport
	HasMore bool
}

type ImportErrorListFilter struct {
	ImportID     string
	Limit        int
	AfterFileRow int
	AfterID      string
}

type ImportErrorListPage struct {
	Items   []domain.ImportRowError
	Total   int64
	HasMore bool
}

type ImportRepository interface {
	CreateImport(ctx context.Context, imp domain.CatalogImport) error
	UpdateImport(ctx context.Context, imp domain.CatalogImport) error
	GetImportByID(ctx context.Context, id string) (*domain.CatalogImport, error)
	GetActiveImport(ctx context.Context) (*domain.CatalogImport, error)
	ListImports(ctx context.Context, filter ImportListFilter) (ImportListPage, error)
	InsertImportErrors(ctx context.Context, rows []domain.ImportRowError) error
	ListImportErrors(ctx context.Context, filter ImportErrorListFilter) (ImportErrorListPage, error)
	PurgeStaleImports(ctx context.Context, before time.Time, batchSize int) (int64, error)
	ApplyImportChunk(ctx context.Context, imp domain.CatalogImport, creates, updates []domain.Item, errs []domain.ImportRowError) error
}
