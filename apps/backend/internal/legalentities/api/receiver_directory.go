package api

import "context"

type ReceiverDirectory interface {
	HasAny(ctx context.Context) (bool, error)
	ReceiverMatches(ctx context.Context, taxID string) (bool, error)
}
