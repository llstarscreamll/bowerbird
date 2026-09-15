package application

import (
	"github.com/bowerbird/internal/legalentities/application/commands"
	"github.com/bowerbird/internal/legalentities/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	CreateLegalEntity *commands.CreateLegalEntityCommand
	UpdateLegalEntity *commands.UpdateLegalEntityCommand
}

type Queries struct {
	ListLegalEntities *queries.ListLegalEntitiesQuery
}
