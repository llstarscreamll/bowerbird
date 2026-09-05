package api

import "context"

const SharingPolicyPrivate = "private"

type ConnectionInfo struct {
	ID                   string
	Provider             string
	ProviderAccountEmail string
	OwnerUserID          string
	SharingPolicy        string
}

func (c ConnectionInfo) IsPrivate() bool {
	return c.SharingPolicy == SharingPolicyPrivate
}

// InternalService is the connections Open Host Service for inbox sync.
type InternalService interface {
	GetActiveConnections(ctx context.Context) ([]ConnectionInfo, error)
	DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error)
	MarkRequiresReconnect(ctx context.Context, connectionID, reason string) error
	GetSharingPolicy(ctx context.Context, connectionID string) (string, error)
}
