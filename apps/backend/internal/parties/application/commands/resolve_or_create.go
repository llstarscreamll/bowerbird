package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
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

func (cmd *ResolveOrCreateFromIssuerCommand) Execute(ctx context.Context, taxID, name string) (*domain.Party, error) {
	if cmd.repo == nil {
		return nil, fmt.Errorf("party repository is required")
	}

	normalizedTaxID := domain.NormalizeTaxID(taxID)
	if normalizedTaxID == "" {
		return nil, nil
	}

	existing, err := cmd.repo.GetByTaxID(ctx, normalizedTaxID)
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

	party, err := domain.NewProvisionalSupplier(cmd.newID(), normalizedTaxID, name, cmd.now())
	if err != nil {
		return nil, err
	}
	if err := cmd.repo.Create(ctx, party); err != nil {
		// Concurrent create: re-read by tax id
		again, getErr := cmd.repo.GetByTaxID(ctx, normalizedTaxID)
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
		return nil, fmt.Errorf("party not found")
	}
	if input.Name != nil {
		if err := party.Rename(*input.Name, cmd.now()); err != nil {
			return nil, err
		}
	}
	if input.Roles != nil {
		party.ReplaceRoles(*input.Roles, cmd.now())
	}
	if input.Name == nil && input.Roles == nil {
		party.UpdatedAt = cmd.now().UTC()
	}
	if err := cmd.repo.Update(ctx, *party); err != nil {
		return nil, err
	}
	return party, nil
}
