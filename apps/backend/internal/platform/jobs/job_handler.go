package jobs

import (
	"context"
	"errors"
)

var ErrJobScopeMismatch = errors.New("job scope mismatch")

type Scope string

const (
	ScopeTenant   Scope = "tenant"
	ScopePlatform Scope = "platform"
)

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
	Scope() Scope
	Handle(ctx context.Context, msg JobMessage) error
}
