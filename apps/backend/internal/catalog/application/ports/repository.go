package ports

import (
	"context"

	"github.com/bowerbird/internal/catalog/domain"
)

type ItemRepository interface {
	CreateItem(ctx context.Context, item domain.Item) error
	UpdateItem(ctx context.Context, item domain.Item) error
	GetItemByID(ctx context.Context, id string) (*domain.Item, error)
	ListItems(ctx context.Context, filter ItemListFilter) ([]domain.Item, error)
	FindByNormalizedDescription(ctx context.Context, normalizedDesc string) ([]domain.Item, error)
}

type ItemListFilter struct {
	Kind   string
	Status string
	Search string
}

type AliasRepository interface {
	CreateAlias(ctx context.Context, alias domain.Alias) error
	FindBySchemePartyValue(ctx context.Context, scheme, partyID, value string) (*domain.Alias, error)
}

type MatchMemoryRepository interface {
	UpsertMemory(ctx context.Context, memory domain.MatchMemory) error
	FindMemoryByEvidenceKey(ctx context.Context, evidenceKey string) (*domain.MatchMemory, error)
}

type InvoiceLineLinkRepository interface {
	UpdateLineLink(ctx context.Context, lineID string, itemID *string, status, method string, locked bool, suggestions []domain.Suggestion) error
	ListReviewLines(ctx context.Context, statuses []string) ([]ReviewLine, error)
	GetLineLinkState(ctx context.Context, lineID string) (*LineLinkState, error)
	SyncHeaderLinkingStatus(ctx context.Context, invoiceHeaderID string) error
}

type ReviewLine struct {
	LineID          string
	InvoiceHeaderID string
	LineNumber      int
	ItemCode        string
	Description     string
	ItemID          string
	LinkStatus      string
	LinkMethod      string
	LinkLocked      bool
	Suggestions     []domain.Suggestion
}

type LineLinkState struct {
	LineID          string
	InvoiceHeaderID string
	ItemID          string
	LinkStatus      string
	LinkMethod      string
	LinkLocked      bool
	ItemCode        string
	Description     string
	PartyID         string
}

type SoftMatcher interface {
	Match(ctx context.Context, description string) ([]domain.Suggestion, error)
}
