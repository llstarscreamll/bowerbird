package queries

import (
	"context"
	"encoding/json"

	"github.com/bowerbird/internal/invoices/application/ports"
	"github.com/bowerbird/internal/invoices/domain"
)

type GetInvoiceByIDQuery struct {
	repo    ports.InvoiceQueryRepository
	catalog ports.CatalogService
}

func NewGetInvoiceByIDQuery(repo ports.InvoiceQueryRepository, catalog ports.CatalogService) *GetInvoiceByIDQuery {
	return &GetInvoiceByIDQuery{repo: repo, catalog: catalog}
}

type InvoiceDetails struct {
	Header domain.InvoiceHeaderRecord
	Lines  []domain.InvoiceLineRecord
}

type suggestionDTO struct {
	ItemID string  `json:"item_id"`
	Name   string  `json:"name,omitempty"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

func (q *GetInvoiceByIDQuery) Execute(ctx context.Context, id string) (*InvoiceDetails, error) {
	header, lines, err := q.repo.GetInvoiceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := q.enrichSuggestionNames(ctx, lines); err != nil {
		return nil, err
	}
	if err := q.enrichLinkedItemDisplays(ctx, lines); err != nil {
		return nil, err
	}

	return &InvoiceDetails{
		Header: *header,
		Lines:  lines,
	}, nil
}

func (q *GetInvoiceByIDQuery) enrichSuggestionNames(ctx context.Context, lines []domain.InvoiceLineRecord) error {
	if q.catalog == nil || len(lines) == 0 {
		return nil
	}

	parsed := make([][]suggestionDTO, len(lines))
	ids := make([]string, 0)
	for i, line := range lines {
		if len(line.Suggestions) == 0 {
			parsed[i] = []suggestionDTO{}
			continue
		}
		var suggestions []suggestionDTO
		if err := json.Unmarshal(line.Suggestions, &suggestions); err != nil {
			parsed[i] = []suggestionDTO{}
			continue
		}
		parsed[i] = suggestions
		for _, s := range suggestions {
			if s.ItemID != "" {
				ids = append(ids, s.ItemID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	names, err := q.catalog.GetItemNames(ctx, ids)
	if err != nil {
		return err
	}

	for i, suggestions := range parsed {
		if len(suggestions) == 0 {
			continue
		}
		for j := range suggestions {
			suggestions[j].Name = names[suggestions[j].ItemID]
		}
		raw, err := json.Marshal(suggestions)
		if err != nil {
			return err
		}
		lines[i].Suggestions = raw
	}
	return nil
}

func (q *GetInvoiceByIDQuery) enrichLinkedItemDisplays(ctx context.Context, lines []domain.InvoiceLineRecord) error {
	if q.catalog == nil || len(lines) == 0 {
		return nil
	}

	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		if line.ItemID != "" {
			ids = append(ids, line.ItemID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	displays, err := q.catalog.GetItemDisplays(ctx, ids)
	if err != nil {
		return err
	}

	for i := range lines {
		if lines[i].ItemID == "" {
			continue
		}
		if d, ok := displays[lines[i].ItemID]; ok {
			lines[i].ItemName = d.Name
			lines[i].ItemSKU = d.InternalSKU
		}
	}
	return nil
}
