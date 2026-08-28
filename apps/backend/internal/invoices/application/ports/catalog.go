package ports

import "context"

// CatalogService is an anti-corruption port for resolving catalog item display names
// without reach-through SQL into catalog tables from the invoices persistence layer.
type CatalogService interface {
	GetItemNames(ctx context.Context, ids []string) (map[string]string, error)
}
