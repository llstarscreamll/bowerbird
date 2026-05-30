package events

import "context"

type BusinessEvent struct {
	Source     string
	DetailType string
	Detail     []byte
}

type BusinessEventPublisher interface {
	PublishBusinessEvent(ctx context.Context, event BusinessEvent) error
}
