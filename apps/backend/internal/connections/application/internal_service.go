package application

import (
	"context"

	"github.com/bowerbird/internal/connections/application/queries"
)

type ConnectionInfo = queries.ConnectionInfo

type InternalService interface {
	GetActiveConnections(ctx context.Context) ([]ConnectionInfo, error)
	DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error)
	MarkRequiresReconnect(ctx context.Context, connectionID, reason string) error
	GetSharingPolicy(ctx context.Context, connectionID string) (string, error)
}
