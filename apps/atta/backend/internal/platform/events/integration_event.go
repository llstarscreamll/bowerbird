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

type IntegrationEvent struct {
	ID                string
	Source            string
	DetailType        string
	Detail            []byte
	TenantSlug        string
	CorrelationID     string
	TenantAttestation string
}

type IntegrationEventHandler interface {
	DetailType() string
	Handle(ctx context.Context, event IntegrationEvent) error
}
