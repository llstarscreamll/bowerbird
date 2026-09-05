package commands

import (
	"context"
	"time"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
)

type UpdatePartyCommand struct {
	repo ports.PartyRepository
	now  func() time.Time
}

func NewUpdatePartyCommand(repo ports.PartyRepository) *UpdatePartyCommand {
	if repo == nil {
		panic("party repository is required")
	}
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
