package queries

import (
	"context"

	"github.com/atta/internal/catalog/application/ports"
)

// ItemDisplay is name + canonical internal code for linked-line enrichment.
type ItemDisplay struct {
	Name         string
	InternalCode string
}

type GetItemDisplaysQuery struct {
	items ports.ItemRepository
}

func NewGetItemDisplaysQuery(items ports.ItemRepository) *GetItemDisplaysQuery {
	if items == nil {
		panic("item repository is required")
	}
	return &GetItemDisplaysQuery{items: items}
}

func (q *GetItemDisplaysQuery) Execute(ctx context.Context, ids []string) (map[string]ItemDisplay, error) {
	out := make(map[string]ItemDisplay, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	items, err := q.items.GetItemsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]ItemDisplay, len(items))
	for _, item := range items {
		code := ""
		if parsed, ok := item.ParsedInternalCode(); ok {
			code = parsed.String()
		}
		byID[item.ID] = ItemDisplay{Name: item.Name, InternalCode: code}
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		out[id] = byID[id]
	}
	return out, nil
}
