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

type PartyChannelsCommand struct {
	repo  ports.PartyRepository
	now   func() time.Time
	newID func() string
}

func NewPartyChannelsCommand(repo ports.PartyRepository) *PartyChannelsCommand {
	if repo == nil {
		panic("party repository is required")
	}
	return &PartyChannelsCommand{repo: repo, now: time.Now, newID: id.NewULID}
}

func (cmd *PartyChannelsCommand) load(ctx context.Context, partyID string) (*domain.Party, error) {
	party, err := cmd.repo.GetByID(ctx, partyID)
	if err != nil {
		return nil, err
	}
	if party == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "party not found")
	}
	return party, nil
}

func (cmd *PartyChannelsCommand) AddEmail(ctx context.Context, partyID, value string) (*domain.Party, error) {
	party, err := cmd.load(ctx, partyID)
	if err != nil {
		return nil, err
	}
	email, err := party.AddEmail(cmd.newID(), value, domain.SourceManual, cmd.now())
	if err != nil {
		return mapChannelErr(err)
	}
	if err := cmd.repo.InsertEmail(ctx, party.ID, *email); err != nil {
		return nil, err
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return cmd.repo.GetByID(ctx, party.ID)
}

func (cmd *PartyChannelsCommand) AddPhone(ctx context.Context, partyID, value string) (*domain.Party, error) {
	party, err := cmd.load(ctx, partyID)
	if err != nil {
		return nil, err
	}
	phone, err := party.AddPhone(cmd.newID(), value, domain.SourceManual, cmd.now())
	if err != nil {
		return mapChannelErr(err)
	}
	if err := cmd.repo.InsertPhone(ctx, party.ID, *phone); err != nil {
		return nil, err
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return cmd.repo.GetByID(ctx, party.ID)
}

func (cmd *PartyChannelsCommand) AddAddress(ctx context.Context, partyID, line, city, department, postalZone, country, kind string) (*domain.Party, error) {
	party, err := cmd.load(ctx, partyID)
	if err != nil {
		return nil, err
	}
	addr, err := party.AddAddress(cmd.newID(), line, city, department, postalZone, country, kind, domain.SourceManual, cmd.now())
	if err != nil {
		return mapChannelErr(err)
	}
	if err := cmd.repo.InsertAddress(ctx, party.ID, *addr); err != nil {
		return nil, err
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return cmd.repo.GetByID(ctx, party.ID)
}

func (cmd *PartyChannelsCommand) RemoveEmail(ctx context.Context, partyID, emailID string) (*domain.Party, error) {
	party, err := cmd.load(ctx, partyID)
	if err != nil {
		return nil, err
	}
	if err := party.RemoveEmail(emailID, cmd.now()); err != nil {
		return mapChannelErr(err)
	}
	if err := cmd.repo.DeleteEmail(ctx, partyID, emailID); err != nil {
		return nil, err
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return cmd.repo.GetByID(ctx, party.ID)
}

func (cmd *PartyChannelsCommand) RemovePhone(ctx context.Context, partyID, phoneID string) (*domain.Party, error) {
	party, err := cmd.load(ctx, partyID)
	if err != nil {
		return nil, err
	}
	if err := party.RemovePhone(phoneID, cmd.now()); err != nil {
		return mapChannelErr(err)
	}
	if err := cmd.repo.DeletePhone(ctx, partyID, phoneID); err != nil {
		return nil, err
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return cmd.repo.GetByID(ctx, party.ID)
}

func (cmd *PartyChannelsCommand) RemoveAddress(ctx context.Context, partyID, addressID string) (*domain.Party, error) {
	party, err := cmd.load(ctx, partyID)
	if err != nil {
		return nil, err
	}
	if err := party.RemoveAddress(addressID, cmd.now()); err != nil {
		return mapChannelErr(err)
	}
	if err := cmd.repo.DeleteAddress(ctx, partyID, addressID); err != nil {
		return nil, err
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return cmd.repo.GetByID(ctx, party.ID)
}

func mapChannelErr(err error) (*domain.Party, error) {
	if errors.Is(err, domain.ErrDuplicateChannel) {
		return nil, appErrors.New(appErrors.CodeConflict, err.Error())
	}
	if errors.Is(err, domain.ErrChannelNotFound) {
		return nil, appErrors.New(appErrors.CodeNotFound, err.Error())
	}
	return mapDomainValidation(err)
}
