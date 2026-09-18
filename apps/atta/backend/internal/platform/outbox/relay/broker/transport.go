package broker

import (
	"context"

	"github.com/atta/internal/platform/outbox/store"
)

type Transport interface {
	DeliverEvent(ctx context.Context, row store.EventRow) error
	DeliverJob(ctx context.Context, row store.JobRow) error
}
