package application

import (
	"context"

	invoicesapi "github.com/atta/internal/invoices/api"
	"github.com/atta/internal/invoices/application/ports"
)

type itemLinkSupport struct {
	repo ports.CatalogItemLinkRepository
}

func NewItemLinkSupport(repo ports.CatalogItemLinkRepository) invoicesapi.ItemLinkSupport {
	if repo == nil {
		panic("catalog item link repository is required")
	}
	return &itemLinkSupport{repo: repo}
}

func (s *itemLinkSupport) RelinkItems(ctx context.Context, fromIDs []string, toID string) error {
	return s.repo.RelinkCatalogItems(ctx, fromIDs, toID)
}

func (s *itemLinkSupport) HardConflictPairs(ctx context.Context) ([]invoicesapi.ItemIDPair, error) {
	pairs, err := s.repo.HardConflictItemPairs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]invoicesapi.ItemIDPair, 0, len(pairs))
	for _, pair := range pairs {
		out = append(out, invoicesapi.ItemIDPair{Left: pair.Left, Right: pair.Right})
	}
	return out, nil
}

func (s *itemLinkSupport) CountLinks(ctx context.Context, itemIDs []string) (map[string]int, error) {
	return s.repo.CountLinesByItemIDs(ctx, itemIDs)
}
