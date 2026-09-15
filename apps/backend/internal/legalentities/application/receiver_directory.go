package application

import (
	"context"

	"github.com/bowerbird/internal/legalentities/api"
)

func NewReceiverDirectory(app *Application) api.ReceiverDirectory {
	if app == nil {
		panic("legalentities application is required")
	}
	return &receiverDirectory{app: app}
}

type receiverDirectory struct {
	app *Application
}

func (d *receiverDirectory) HasAny(ctx context.Context) (bool, error) {
	items, err := d.app.Queries.ListLegalEntities.Execute(ctx)
	if err != nil {
		return false, err
	}
	return len(items) > 0, nil
}

func (d *receiverDirectory) ReceiverMatches(ctx context.Context, taxID string) (bool, error) {
	items, err := d.app.Queries.ListLegalEntities.Execute(ctx)
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if item.ReceivesAs(taxID) {
			return true, nil
		}
	}
	return false, nil
}

var _ api.ReceiverDirectory = (*receiverDirectory)(nil)
