package scheduler

import "context"

// Scheduler enqueues periodic background jobs via outbox (implementation deferred).
type Scheduler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Noop struct{}

func (Noop) Start(context.Context) error { return nil }
func (Noop) Stop(context.Context) error  { return nil }
