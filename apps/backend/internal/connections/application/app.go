package application

import (
	"github.com/bowerbird/internal/connections/application/commands"
	"github.com/bowerbird/internal/connections/application/queries"
	"github.com/bowerbird/internal/connections/domain"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	MarkRequiresReconnect *commands.MarkRequiresReconnectCommand
}

type Queries struct {
	GetActiveConnections *queries.GetActiveConnectionsQuery
	DecryptCredentials   *queries.DecryptCredentialsQuery
	GetSharingPolicy     *queries.GetSharingPolicyQuery
}

func NewApplication(repo domain.Repository, credentialsService *commands.CredentialsService) *Application {
	if repo == nil {
		panic("connection repository is required")
	}
	if credentialsService == nil {
		panic("credentials service is required")
	}
	return &Application{
		Commands: Commands{
			MarkRequiresReconnect: commands.NewMarkRequiresReconnectCommand(repo),
		},
		Queries: Queries{
			GetActiveConnections: queries.NewGetActiveConnectionsQuery(repo),
			DecryptCredentials:   queries.NewDecryptCredentialsQuery(repo, credentialsService),
			GetSharingPolicy:     queries.NewGetSharingPolicyQuery(repo),
		},
	}
}
