package commands

import (
	"context"
	"errors"
	"time"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
)

type CreatePartyCommand struct {
	repo  ports.PartyRepository
	now   func() time.Time
	newID func() string
}

func NewCreatePartyCommand(repo ports.PartyRepository) *CreatePartyCommand {
	return &CreatePartyCommand{repo: repo, now: time.Now, newID: id.NewULID}
}

type CreatePartyInput struct {
	Name  string
	TaxID string
	Roles []string
}

func (cmd *CreatePartyCommand) Execute(ctx context.Context, input CreatePartyInput) (*domain.Party, error) {
	taxID, err := domain.ParseTaxID(input.TaxID)
	if err != nil {
		return mapDomainValidation(err)
	}
	roles, err := domain.ParsePartyRoles(input.Roles)
	if err != nil {
		return mapDomainValidation(err)
	}

	existing, err := cmd.repo.GetByTaxID(ctx, taxID.String())
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, appErrors.New(appErrors.CodeConflict, "a party with this tax id already exists")
	}

	party, err := domain.NewConfirmedParty(cmd.newID(), taxID, input.Name, roles, cmd.now())
	if err != nil {
		return mapDomainValidation(err)
	}
	if err := cmd.repo.Create(ctx, party); err != nil {
		return nil, err
	}
	return &party, nil
}

func mapDomainValidation(err error) (*domain.Party, error) {
	if errors.Is(err, domain.ErrMissingTaxID) ||
		errors.Is(err, domain.ErrMissingPartyName) ||
		errors.Is(err, domain.ErrPartyIDRequired) ||
		errors.Is(err, domain.ErrMissingRoles) ||
		errors.Is(err, domain.ErrInvalidRole) {
		return nil, appErrors.New(appErrors.CodeValidation, err.Error())
	}
	return nil, err
}
