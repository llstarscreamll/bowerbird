package commands

import (
	"context"
	"time"

	"github.com/atta/internal/legalentities/application/ports"
	"github.com/atta/internal/legalentities/domain"
	appErrors "github.com/atta/internal/platform/errors"
)

type UpdateLegalEntityInput struct {
	ID        string
	TaxID     *string
	SchemeID  *string
	LegalName *string
}

type UpdateLegalEntityCommand struct {
	repo      ports.LegalEntityRepository
	publisher ports.RegistrationPublisher
	now       func() time.Time
}

func NewUpdateLegalEntityCommand(repo ports.LegalEntityRepository, publisher ports.RegistrationPublisher) *UpdateLegalEntityCommand {
	if repo == nil {
		panic("legal entity repository is required")
	}
	if publisher == nil {
		panic("registration publisher is required")
	}
	return &UpdateLegalEntityCommand{repo: repo, publisher: publisher, now: time.Now}
}

func (cmd *UpdateLegalEntityCommand) Execute(ctx context.Context, input UpdateLegalEntityInput) (*domain.LegalEntity, error) {
	entity, err := cmd.repo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, appErrors.New(appErrors.CodeNotFound, "legal entity not found")
	}

	changed := false
	if input.LegalName != nil {
		before := entity.LegalName
		if err := entity.Rename(*input.LegalName, cmd.now()); err != nil {
			return mapDomainValidation(err)
		}
		if entity.LegalName != before {
			changed = true
		}
	}

	if input.TaxID != nil || input.SchemeID != nil {
		taxRaw := entity.TaxID
		schemeRaw := entity.SchemeID
		if input.TaxID != nil {
			taxRaw = *input.TaxID
		}
		if input.SchemeID != nil {
			schemeRaw = *input.SchemeID
		}
		taxID, err := domain.ParseTaxID(taxRaw)
		if err != nil {
			return mapDomainValidation(err)
		}
		scheme, err := domain.ParseScheme(schemeRaw)
		if err != nil {
			return mapDomainValidation(err)
		}
		if entity.ChangeIdentity(taxID, scheme, cmd.now()) {
			changed = true
		}
	}

	if !changed {
		return entity, nil
	}
	if err := cmd.repo.Update(ctx, *entity); err != nil {
		return nil, err
	}
	if domain.RequiresRegistrationNotice(entity.PullEvents()) {
		if err := cmd.publisher.PublishRegistered(ctx); err != nil {
			return nil, err
		}
	}
	return entity, nil
}
