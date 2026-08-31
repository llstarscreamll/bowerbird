package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
)

type ResolveOrCreateFromIssuerCommand struct {
	repo  ports.PartyRepository
	now   func() time.Time
	newID func() string
}

func NewResolveOrCreateFromIssuerCommand(repo ports.PartyRepository) *ResolveOrCreateFromIssuerCommand {
	return &ResolveOrCreateFromIssuerCommand{repo: repo, now: time.Now, newID: id.NewULID}
}

func (cmd *ResolveOrCreateFromIssuerCommand) Execute(ctx context.Context, taxIDRaw, name string) (*domain.Party, error) {
	if cmd.repo == nil {
		return nil, fmt.Errorf("party repository is required")
	}

	taxID, err := domain.ParseTaxID(taxIDRaw)
	if err != nil {
		if errors.Is(err, domain.ErrMissingTaxID) {
			return nil, nil
		}
		return nil, err
	}

	existing, err := cmd.repo.GetByTaxID(ctx, taxID.String())
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.EnsureSupplierRole(cmd.now()) {
			if err := cmd.repo.Update(ctx, *existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	party := domain.NewProvisionalSupplier(cmd.newID(), taxID, name, cmd.now())
	if err := cmd.repo.Create(ctx, party); err != nil {
		again, getErr := cmd.repo.GetByTaxID(ctx, taxID.String())
		if getErr == nil && again != nil {
			return again, nil
		}
		return nil, err
	}
	return &party, nil
}

type UpdatePartyCommand struct {
	repo ports.PartyRepository
	now  func() time.Time
}

func NewUpdatePartyCommand(repo ports.PartyRepository) *UpdatePartyCommand {
	return &UpdatePartyCommand{repo: repo, now: time.Now}
}

type UpdatePartyInput struct {
	ID    string
	Name  *string
	Roles *[]string
}

func (cmd *UpdatePartyCommand) Execute(ctx context.Context, input UpdatePartyInput) (*domain.Party, error) {
	party, err := cmd.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if party == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "party not found")
	}

	var rolesVO *domain.PartyRoles
	if input.Roles != nil {
		parsed, err := domain.ParsePartyRoles(*input.Roles)
		if err != nil {
			return mapDomainValidation(err)
		}
		rolesVO = &parsed
	}

	changed, err := party.UpdateProfile(input.Name, rolesVO, cmd.now())
	if err != nil {
		return mapDomainValidation(err)
	}
	if !changed {
		return party, nil
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return party, nil
}
