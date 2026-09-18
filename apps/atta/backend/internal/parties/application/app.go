package application

import (
	"github.com/atta/internal/parties/application/commands"
	"github.com/atta/internal/parties/application/queries"
)

type Application struct {
	Commands Commands
	Queries  Queries
}

type Commands struct {
	ResolveOrCreateFromIssuer *commands.ResolveOrCreateFromIssuerCommand
	CreateParty               *commands.CreatePartyCommand
	UpdateParty               *commands.UpdatePartyCommand
	PartyChannels             *commands.PartyChannelsCommand
}

type Queries struct {
	GetPartyByID *queries.GetPartyByIDQuery
	ListParties  *queries.ListPartiesQuery
}
