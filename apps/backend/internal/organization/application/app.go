package application

import (
	"github.com/bowerbird/internal/organization/application/commands"
	"github.com/bowerbird/internal/organization/application/ports"
	"github.com/bowerbird/internal/organization/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	CreateOrganization *commands.CreateOrganizationCommand
}

type Queries struct {
	GetOrganization *queries.GetOrganizationQuery
}

func NewApplication(repo ports.OrganizationRepository, provisioner ports.Provisioner) *Application {
	return &Application{
		Commands: Commands{
			CreateOrganization: commands.NewCreateOrganizationCommand(repo, provisioner),
		},
		Queries: Queries{
			GetOrganization: queries.NewGetOrganizationQuery(repo),
		},
	}
}
