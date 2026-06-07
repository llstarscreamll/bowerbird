package events

import "context"

type BusinessEvent struct {
	Source     string
	DetailType string
	Detail     []byte
}

type EventBus interface {
	Publish(ctx context.Context, event BusinessEvent) error
}
