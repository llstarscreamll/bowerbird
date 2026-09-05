package application

import (
	"context"

	"github.com/bowerbird/internal/parties/api"
	"github.com/bowerbird/internal/parties/application/commands"
)

func NewIssuerPartyLookupFromApp(app *Application) api.IssuerPartyLookup {
	if app == nil {
		panic("parties application is required")
	}
	return NewIssuerPartyLookup(app.Commands.ResolveOrCreateFromIssuer)
}

type issuerPartyLookup struct {
	resolve *commands.ResolveOrCreateFromIssuerCommand
}

func NewIssuerPartyLookup(cmd *commands.ResolveOrCreateFromIssuerCommand) api.IssuerPartyLookup {
	if cmd == nil {
		panic("resolve or create from issuer command is required")
	}
	return &issuerPartyLookup{resolve: cmd}
}

func (l *issuerPartyLookup) ResolveIssuerPartyID(ctx context.Context, taxID, name string) (string, error) {
	party, err := l.resolve.Execute(ctx, taxID, name)
	if err != nil {
		return "", err
	}
	if party == nil {
		return "", nil
	}
	return party.ID, nil
}

var _ api.IssuerPartyLookup = (*issuerPartyLookup)(nil)
