package application

import (
	"context"

	"github.com/bowerbird/internal/parties/application/commands"
)

// IssuerPartyLookup is the cross-context integration surface for invoice issuer resolution.
type IssuerPartyLookup interface {
	ResolveIssuerPartyID(ctx context.Context, taxID, name string) (partyID string, err error)
}

// NewIssuerPartyLookupFromApp builds the lookup from a wired Application.
func NewIssuerPartyLookupFromApp(app *Application) IssuerPartyLookup {
	if app == nil {
		return NewIssuerPartyLookup(nil)
	}
	return NewIssuerPartyLookup(app.Commands.ResolveOrCreateFromIssuer)
}

type issuerPartyLookup struct {
	resolve *commands.ResolveOrCreateFromIssuerCommand
}

func NewIssuerPartyLookup(cmd *commands.ResolveOrCreateFromIssuerCommand) IssuerPartyLookup {
	return &issuerPartyLookup{resolve: cmd}
}

func (l *issuerPartyLookup) ResolveIssuerPartyID(ctx context.Context, taxID, name string) (string, error) {
	if l == nil || l.resolve == nil {
		return "", nil
	}
	party, err := l.resolve.Execute(ctx, taxID, name)
	if err != nil {
		return "", err
	}
	if party == nil {
		return "", nil
	}
	return party.ID, nil
}
