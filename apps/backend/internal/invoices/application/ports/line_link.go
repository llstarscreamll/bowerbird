package ports

import (
	"context"

	"github.com/bowerbird/internal/invoices/domain"
)

type EnrichedSuggestion struct {
	ItemID string  `json:"item_id"`
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
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
	Suggestions     []EnrichedSuggestion
}

type InvoiceLineLinkRepository interface {
	SaveLineLink(ctx context.Context, lineID string, link domain.LineLink) error
	ListReviewLines(ctx context.Context, statuses []string) ([]ReviewLine, error)
	GetLineForDecision(ctx context.Context, lineID string) (*domain.LineForDecision, error)
	SyncHeaderLinkingStatus(ctx context.Context, invoiceHeaderID string) error
}
