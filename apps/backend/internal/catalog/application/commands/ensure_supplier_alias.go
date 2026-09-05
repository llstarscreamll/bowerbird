package commands

import (
	"context"
	"time"

	"github.com/bowerbird/internal/catalog/application/ports"
	"github.com/bowerbird/internal/catalog/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
)

type EnsureSupplierAliasCommand struct {
	aliases ports.AliasRepository
	now     func() time.Time
	newID   func() string
}

func NewEnsureSupplierAliasCommand(aliases ports.AliasRepository) *EnsureSupplierAliasCommand {
	if aliases == nil {
		panic("alias repository is required")
	}
	return &EnsureSupplierAliasCommand{
		aliases: aliases,
		now:     time.Now,
		newID:   id.NewULID,
	}
}

func (cmd *EnsureSupplierAliasCommand) Execute(ctx context.Context, partyID, code, itemID string) error {
	existing, err := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, partyID, code)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.PointsTo(itemID) {
			return nil
		}
		return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item")
	}

	alias, err := domain.NewSupplierSKUAlias(cmd.newID(), itemID, partyID, code, cmd.now())
	if err != nil {
		return err
	}
	err = cmd.aliases.CreateAlias(ctx, alias)
	if err == nil {
		return nil
	}
	if !isConflict(err) {
		return err
	}
	again, findErr := cmd.aliases.FindBySchemePartyValue(ctx, domain.AliasSchemeSupplierSKU, partyID, code)
	if findErr != nil {
		return findErr
	}
	if again != nil && again.PointsTo(itemID) {
		return nil
	}
	return appErrors.New(appErrors.CodeConflict, "an alias with this scheme, party, and value already exists for another item")
}
