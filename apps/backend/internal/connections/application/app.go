package application

import (
	"github.com/bowerbird/internal/connections/application/commands"
	"github.com/bowerbird/internal/connections/application/ports"
	"github.com/bowerbird/internal/connections/application/queries"
	"github.com/bowerbird/internal/connections/domain"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	MarkRequiresReconnect   *commands.MarkRequiresReconnectCommand
	UpsertMailboxConnection *commands.UpsertMailboxConnectionCommand
}

type Queries struct {
	GetActiveConnections *queries.GetActiveConnectionsQuery
	DecryptCredentials   *queries.DecryptCredentialsQuery
	GetSharingPolicy     *queries.GetSharingPolicyQuery
}

func NewApplication(repo domain.Repository, credentials ports.Credentials) *Application {
	if repo == nil {
		panic("connection repository is required")
	}
	if credentials == nil {
		panic("credentials are required")
	}
	return &Application{
		Commands: Commands{
			MarkRequiresReconnect:   commands.NewMarkRequiresReconnectCommand(repo),
			UpsertMailboxConnection: commands.NewUpsertMailboxConnectionCommand(repo, credentials),
		},
		Queries: Queries{
			GetActiveConnections: queries.NewGetActiveConnectionsQuery(repo),
			DecryptCredentials:   queries.NewDecryptCredentialsQuery(repo, credentials),
			GetSharingPolicy:     queries.NewGetSharingPolicyQuery(repo),
		},
	}
}
