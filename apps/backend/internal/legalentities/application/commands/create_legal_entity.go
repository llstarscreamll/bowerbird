package commands

import (
	"context"
	"errors"
	"time"

	"github.com/bowerbird/internal/legalentities/application/ports"
	"github.com/bowerbird/internal/legalentities/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
)

type CreateLegalEntityInput struct {
	TaxID     string
	SchemeID  string
	LegalName string
}

type CreateLegalEntityCommand struct {
	repo      ports.LegalEntityRepository
	publisher ports.RegistrationPublisher
	now       func() time.Time
	newID     func() string
}

func NewCreateLegalEntityCommand(repo ports.LegalEntityRepository, publisher ports.RegistrationPublisher) *CreateLegalEntityCommand {
	if repo == nil {
		panic("legal entity repository is required")
	}
	if publisher == nil {
		panic("registration publisher is required")
	}
	return &CreateLegalEntityCommand{repo: repo, publisher: publisher, now: time.Now, newID: id.NewULID}
}

func (cmd *CreateLegalEntityCommand) Execute(ctx context.Context, input CreateLegalEntityInput) (*domain.LegalEntity, error) {
	count, err := cmd.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.AssertRegistrable(count); err != nil {
		return nil, appErrors.New(appErrors.CodeConflict, err.Error())
	}

	taxID, err := domain.ParseTaxID(input.TaxID)
	if err != nil {
		return mapDomainValidation(err)
	}
	scheme, err := domain.ParseScheme(input.SchemeID)
	if err != nil {
		return mapDomainValidation(err)
	}
	entity, err := domain.NewLegalEntity(cmd.newID(), taxID, scheme, input.LegalName, cmd.now())
	if err != nil {
		return mapDomainValidation(err)
	}
	if err := cmd.repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	if domain.RequiresRegistrationNotice(entity.PullEvents()) {
		if err := cmd.publisher.PublishRegistered(ctx); err != nil {
			return nil, err
		}
	}
	return &entity, nil
}

func mapDomainValidation(err error) (*domain.LegalEntity, error) {
	if errors.Is(err, domain.ErrMissingTaxID) ||
		errors.Is(err, domain.ErrMissingLegalName) ||
		errors.Is(err, domain.ErrInvalidScheme) ||
		errors.Is(err, domain.ErrMissingID) {
		return nil, appErrors.New(appErrors.CodeValidation, err.Error())
	}
	return nil, err
}
