package ports

import "context"

// ItemDisplay is the catalog projection needed to render a linked invoice line.
type ItemDisplay struct {
	Name        string
	InternalSKU string
}

// CatalogService is an anti-corruption port for resolving catalog item display data
// without reach-through SQL into catalog tables from the invoices persistence layer.
type CatalogService interface {
	GetItemNames(ctx context.Context, ids []string) (map[string]string, error)
	GetItemDisplays(ctx context.Context, ids []string) (map[string]ItemDisplay, error)
}
