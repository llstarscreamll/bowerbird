package jobs

import "context"

type JobMessage struct {
	MessageID         string
	JobType           string
	TenantSlug        string
	CorrelationID     string
	TenantAttestation string
	Body              []byte
}

type JobHandler interface {
	JobType() string
	Handle(ctx context.Context, msg JobMessage) error
}
