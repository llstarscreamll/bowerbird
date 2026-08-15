package application

import (
	"github.com/bowerbird/internal/secrets/application/commands"
	"github.com/bowerbird/internal/secrets/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	CreateSecret   *commands.CreateSecretCommand
	UpdateSecret   *commands.UpdateSecretCommand
	DeleteSecret   *commands.DeleteSecretCommand
	MarkSecretUsed *commands.MarkSecretUsedCommand
}

type Queries struct {
	ListSecrets      *queries.ListSecretsQuery
	GetSecretByID    *queries.GetSecretByIDQuery
	ResolveByPurpose *queries.ResolveByPurposeQuery
}
