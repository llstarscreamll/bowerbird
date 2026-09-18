package application

import (
	"context"

	"github.com/atta/internal/connections/api"
	"github.com/atta/internal/connections/application/commands"
	"github.com/atta/internal/connections/application/queries"
)

type internalService struct {
	getActiveConnections *queries.GetActiveConnectionsQuery
	getConnection        *queries.GetConnectionQuery
	decryptCredentials   *queries.DecryptCredentialsQuery
	markReconnect        *commands.MarkRequiresReconnectCommand
	getSharingPolicy     *queries.GetSharingPolicyQuery
}

func NewInternalService(app *Application) api.InternalService {
	if app == nil {
		panic("connections application is required")
	}
	if app.Queries.GetActiveConnections == nil {
		panic("get active connections query is required")
	}
	if app.Queries.GetConnection == nil {
		panic("get connection query is required")
	}
	if app.Queries.DecryptCredentials == nil {
		panic("decrypt credentials query is required")
	}
	if app.Commands.MarkRequiresReconnect == nil {
		panic("mark requires reconnect command is required")
	}
	if app.Queries.GetSharingPolicy == nil {
		panic("get sharing policy query is required")
	}

	return &internalService{
		getActiveConnections: app.Queries.GetActiveConnections,
		getConnection:        app.Queries.GetConnection,
		decryptCredentials:   app.Queries.DecryptCredentials,
		markReconnect:        app.Commands.MarkRequiresReconnect,
		getSharingPolicy:     app.Queries.GetSharingPolicy,
	}
}

func (s *internalService) GetActiveConnections(ctx context.Context) ([]api.ConnectionInfo, error) {
	items, err := s.getActiveConnections.Execute(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]api.ConnectionInfo, 0, len(items))
	for _, item := range items {
		out = append(out, api.ConnectionInfo{
			ID:                   item.ID,
			Provider:             item.Provider,
			ProviderAccountEmail: item.ProviderAccountEmail,
			OwnerUserID:          item.OwnerUserID,
			SharingPolicy:        item.SharingPolicy,
		})
	}
	return out, nil
}

func (s *internalService) GetConnection(ctx context.Context, connectionID string) (api.ConnectionInfo, error) {
	item, err := s.getConnection.Execute(ctx, connectionID)
	if err != nil {
		return api.ConnectionInfo{}, err
	}
	return api.ConnectionInfo{
		ID:                   item.ID,
		Provider:             item.Provider,
		ProviderAccountEmail: item.ProviderAccountEmail,
		OwnerUserID:          item.OwnerUserID,
		SharingPolicy:        item.SharingPolicy,
	}, nil
}

func (s *internalService) DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error) {
	return s.decryptCredentials.Execute(ctx, connectionID)
}

func (s *internalService) MarkRequiresReconnect(ctx context.Context, connectionID, reason string) error {
	return s.markReconnect.Execute(ctx, connectionID, reason)
}

func (s *internalService) GetSharingPolicy(ctx context.Context, connectionID string) (string, error) {
	return s.getSharingPolicy.Execute(ctx, connectionID)
}

var _ api.InternalService = (*internalService)(nil)
