package application

import (
	"context"
)

type ConnectionInfo struct {
	ID                   string
	Provider             string
	ProviderAccountEmail string
	OwnerUserID          string
	SharingPolicy        string
}

type InternalService interface {
	GetActiveConnections(ctx context.Context) ([]ConnectionInfo, error)
	DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error)
	MarkRequiresReconnect(ctx context.Context, connectionID, reason string) error
	GetSharingPolicy(ctx context.Context, connectionID string) (string, error)
}
