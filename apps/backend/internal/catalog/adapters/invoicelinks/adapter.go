package invoicelinks

import (
	"context"

	"github.com/bowerbird/internal/catalog/application/ports"
	invoicesapi "github.com/bowerbird/internal/invoices/api"
)

type Adapter struct {
	inner invoicesapi.ItemLinkSupport
}

func New(inner invoicesapi.ItemLinkSupport) ports.ItemLinkSupport {
	if inner == nil {
		panic("invoice item link support is required")
	}
	return &Adapter{inner: inner}
}

func (a *Adapter) RelinkItems(ctx context.Context, fromIDs []string, toID string) error {
	return a.inner.RelinkItems(ctx, fromIDs, toID)
}

func (a *Adapter) HardConflictPairs(ctx context.Context) ([]ports.ItemIDPair, error) {
	pairs, err := a.inner.HardConflictPairs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ports.ItemIDPair, 0, len(pairs))
	for _, pair := range pairs {
		out = append(out, ports.ItemIDPair{Left: pair.Left, Right: pair.Right})
	}
	return out, nil
}

func (a *Adapter) CountLinks(ctx context.Context, itemIDs []string) (map[string]int, error) {
	return a.inner.CountLinks(ctx, itemIDs)
}
