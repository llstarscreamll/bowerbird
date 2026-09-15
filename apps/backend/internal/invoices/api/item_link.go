package api

import "context"

type ItemIDPair struct {
	Left  string
	Right string
}

type ItemLinkSupport interface {
	RelinkItems(ctx context.Context, fromIDs []string, toID string) error
	HardConflictPairs(ctx context.Context) ([]ItemIDPair, error)
	CountLinks(ctx context.Context, itemIDs []string) (map[string]int, error)
}
