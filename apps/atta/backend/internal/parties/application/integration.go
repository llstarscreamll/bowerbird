package application

import (
	"context"

	"github.com/atta/internal/parties/api"
	"github.com/atta/internal/parties/application/commands"
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

func (l *issuerPartyLookup) ResolveIssuer(ctx context.Context, profile api.IssuerProfile) (string, error) {
	addrs := make([]commands.IssuerAddress, 0, len(profile.Addresses))
	for _, a := range profile.Addresses {
		addrs = append(addrs, commands.IssuerAddress{
			Line: a.Line, City: a.City, Department: a.Department,
			PostalZone: a.PostalZone, CountryCode: a.CountryCode, Kind: a.Kind,
		})
	}
	party, err := l.resolve.Execute(ctx, commands.IssuerProfile{
		TaxID: profile.TaxID, Name: profile.Name, SchemeID: profile.SchemeID,
		TaxpayerKind: profile.TaxpayerKind, TaxLevelCode: profile.TaxLevelCode,
		Emails: profile.Emails, Phones: profile.Phones, Addresses: addrs,
	})
	if err != nil {
		return "", err
	}
	if party == nil {
		return "", nil
	}
	return party.ID, nil
}

var _ api.IssuerPartyLookup = (*issuerPartyLookup)(nil)
