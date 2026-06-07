package application

import (
	"context"

	"github.com/bowerbird/internal/connections/application/commands"
	"github.com/bowerbird/internal/connections/application/queries"
	"github.com/bowerbird/internal/connections/domain"
)

type internalService struct {
	getActiveConnections *queries.GetActiveConnectionsQuery
	decryptCredentials   *queries.DecryptCredentialsQuery
	markReconnect        *commands.MarkRequiresReconnectCommand
	getSharingPolicy     *queries.GetSharingPolicyQuery
}

func NewInternalService(
	repo domain.Repository,
	credentialsService *CredentialsService,
) InternalService {
	app := NewApplication(repo, credentialsService)

	return &internalService{
		getActiveConnections: app.Queries.GetActiveConnections,
		decryptCredentials:   app.Queries.DecryptCredentials,
		markReconnect:        app.Commands.MarkRequiresReconnect,
		getSharingPolicy:     app.Queries.GetSharingPolicy,
	}
}

func (s *internalService) GetActiveConnections(ctx context.Context) ([]ConnectionInfo, error) {
	return s.getActiveConnections.Execute(ctx)
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
