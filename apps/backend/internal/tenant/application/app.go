package application

import (
	"github.com/bowerbird/internal/tenant/api"
	"github.com/bowerbird/internal/tenant/application/commands"
	"github.com/bowerbird/internal/tenant/application/ports"
	"github.com/bowerbird/internal/tenant/application/queries"
)

type Application struct {
	Commands  Commands
	Queries   Queries
	directory api.Directory
}

type Commands struct {
	CreateTenant *commands.CreateTenantCommand
}

type Queries struct {
	GetTenant *queries.GetTenantQuery
}

func NewApplication(repo ports.TenantRepository, provisioner ports.Provisioner, defaults ports.DefaultPackApplier) *Application {
	if repo == nil {
		panic("tenant repository is required")
	}
	if provisioner == nil {
		panic("tenant provisioner is required")
	}
	if defaults == nil {
		panic("default pack applier is required")
	}
	return &Application{
		Commands: Commands{
			CreateTenant: commands.NewCreateTenantCommand(repo, provisioner, defaults),
		},
		Queries: Queries{
			GetTenant: queries.NewGetTenantQuery(repo),
		},
		directory: newTenantDirectory(repo),
	}
}

func (a *Application) Directory() api.Directory {
	return a.directory
}
