package application

import (
	"github.com/bowerbird/internal/tenant/application/commands"
	"github.com/bowerbird/internal/tenant/application/ports"
	"github.com/bowerbird/internal/tenant/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	CreateTenant *commands.CreateTenantCommand
}

type Queries struct {
	GetTenant *queries.GetTenantQuery
}

func NewApplication(repo ports.TenantRepository, provisioner ports.Provisioner, defaults ports.DefaultPackApplier) *Application {
	return &Application{
		Commands: Commands{
			CreateTenant: commands.NewCreateTenantCommand(repo, provisioner, defaults),
		},
		Queries: Queries{
			GetTenant: queries.NewGetTenantQuery(repo),
		},
	}
}
