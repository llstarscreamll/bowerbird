package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/bowerbird/internal/organization/application/ports"
	"github.com/bowerbird/internal/organization/domain"
	"github.com/bowerbird/internal/platform/id"
)

var ErrSlugAlreadyExists = errors.New("organization slug already exists")

type CreateOrganizationInput struct {
	Name           string
	Slug           string
	OwnerID        string
	OwnerEmail     string
	OwnerFirstName string
	OwnerLastName  string
	OwnerAvatarURL string
}

type CreateOrganizationCommand struct {
	repo        ports.OrganizationRepository
	provisioner ports.Provisioner
}

func NewCreateOrganizationCommand(repo ports.OrganizationRepository, provisioner ports.Provisioner) *CreateOrganizationCommand {
	return &CreateOrganizationCommand{repo: repo, provisioner: provisioner}
}

func (cmd *CreateOrganizationCommand) failProvisioning(ctx context.Context, orgID, slug string, cause error, step string) error {
	if markErr := cmd.repo.UpdateStatus(ctx, orgID, domain.StatusFailed); markErr != nil {
		return errors.Join(
			fmt.Errorf("failed to %s for %s: %w", step, slug, cause),
			fmt.Errorf("failed to mark organization as failed: %w", markErr),
		)
	}

	return fmt.Errorf("failed to %s for %s: %w", step, slug, cause)
}

func (cmd *CreateOrganizationCommand) Execute(ctx context.Context, input CreateOrganizationInput) (*domain.Organization, error) {
	org, err := domain.NewOrganization(input.Name, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("invalid organization data: %w", err)
	}
	org.ID = id.NewULID()

	exists, err := cmd.repo.ExistsBySlug(ctx, org.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if exists {
		return nil, ErrSlugAlreadyExists
	}

	if err := cmd.repo.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to register organization in control plane: %w", err)
	}

	if err := cmd.provisioner.CreateDatabase(ctx, org.DBName); err != nil {
		return nil, cmd.failProvisioning(ctx, org.ID, org.Slug, err, "provision database")
	}

	if err := cmd.provisioner.MigrateDatabase(ctx, org.DBName); err != nil {
		return nil, cmd.failProvisioning(ctx, org.ID, org.Slug, err, "migrate database")
	}
	owner := domain.OwnerData{
		ID:        input.OwnerID,
		Email:     input.OwnerEmail,
		FirstName: input.OwnerFirstName,
		LastName:  input.OwnerLastName,
		AvatarURL: input.OwnerAvatarURL,
	}
	if err := cmd.provisioner.SeedOwner(ctx, org.DBName, owner); err != nil {
		return nil, cmd.failProvisioning(ctx, org.ID, org.Slug, err, "seed owner")
	}

	if err := cmd.repo.AddMembership(ctx, input.OwnerID, org.ID, "OWNER"); err != nil {
		return nil, cmd.failProvisioning(ctx, org.ID, org.Slug, err, "add owner membership")
	}

	if err := cmd.repo.UpdateStatus(ctx, org.ID, domain.StatusActive); err != nil {
		return nil, fmt.Errorf("failed to mark organization as active: %w", err)
	}
	org.Status = domain.StatusActive

	return org, nil
}
