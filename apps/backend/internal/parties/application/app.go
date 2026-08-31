package application

import (
	"github.com/bowerbird/internal/parties/application/commands"
	"github.com/bowerbird/internal/parties/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	ResolveOrCreateFromIssuer *commands.ResolveOrCreateFromIssuerCommand
	CreateParty               *commands.CreatePartyCommand
	UpdateParty               *commands.UpdatePartyCommand
}

type Queries struct {
	GetPartyByID *queries.GetPartyByIDQuery
	ListParties  *queries.ListPartiesQuery
}
