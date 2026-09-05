package queries

import (
	"context"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
)

type ListReviewQueueQuery struct {
	links   ports.InvoiceLineLinkRepository
	catalog ports.CatalogService
}

func NewListReviewQueueQuery(links ports.InvoiceLineLinkRepository, catalog ports.CatalogService) *ListReviewQueueQuery {
	if links == nil {
		panic("invoice line link repository is required")
	}
	if catalog == nil {
		panic("catalog service is required")
	}
	return &ListReviewQueueQuery{links: links, catalog: catalog}
}

func (q *ListReviewQueueQuery) Execute(ctx context.Context) ([]ports.ReviewLine, error) {
	lines, err := q.links.ListReviewLines(ctx, []string{domain.LinkStatusUnmatched, domain.LinkStatusSuggested})
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return lines, nil
	}

	itemIDs := make([]string, 0)
	for _, line := range lines {
		for _, s := range line.Suggestions {
			if s.ItemID != "" {
				itemIDs = append(itemIDs, s.ItemID)
			}
		}
	}
	if len(itemIDs) == 0 {
		return lines, nil
	}

	names, err := q.catalog.GetItemNames(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	for i, line := range lines {
		for j, s := range line.Suggestions {
			lines[i].Suggestions[j].Name = names[s.ItemID]
		}
	}
	return lines, nil
}
