package ports

import "context"

type DefaultPackApplier interface {
	ApplyDefaultPack(ctx context.Context, tenantID, actorUserID string) error
}
