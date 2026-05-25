package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/money-path/bowerbird/apps/backend/internal/organization/domain"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/id"
)

var (
	ErrSlugAlreadyExists = errors.New("organization slug already exists")
)

type CreateOrganizationCommand struct {
	Name           string
	Slug           string
	OwnerID        string
	OwnerEmail     string
	OwnerFirstName string
	OwnerLastName  string
	OwnerAvatarURL string
}

type CreateOrganizationUseCase struct {
	repo        domain.Repository
	provisioner domain.Provisioner
}

func (uc *CreateOrganizationUseCase) failProvisioning(ctx context.Context, orgID, slug string, cause error, step string) error {
	if markErr := uc.repo.UpdateStatus(ctx, orgID, domain.StatusFailed); markErr != nil {
		return errors.Join(
			fmt.Errorf("failed to %s for %s: %w", step, slug, cause),
			fmt.Errorf("failed to mark organization as failed: %w", markErr),
		)
	}

	return fmt.Errorf("failed to %s for %s: %w", step, slug, cause)
}

func NewCreateOrganizationUseCase(repo domain.Repository, provisioner domain.Provisioner) *CreateOrganizationUseCase {
	return &CreateOrganizationUseCase{
		repo:        repo,
		provisioner: provisioner,
	}
}

func (uc *CreateOrganizationUseCase) Execute(ctx context.Context, cmd CreateOrganizationCommand) (*domain.Organization, error) {
	// 1. Create domain entity (validates basic rules)
	org, err := domain.NewOrganization(cmd.Name, cmd.Slug)
	if err != nil {
		return nil, fmt.Errorf("invalid organization data: %w", err)
	}
	org.ID = id.NewULID()

	// 2. Check for uniqueness in Control Plane
	exists, err := uc.repo.ExistsBySlug(ctx, org.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if exists {
		return nil, ErrSlugAlreadyExists
	}

	// 3. Save to Control Plane first to reserve the slug in provisioning status.
	if err := uc.repo.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to register organization in control plane: %w", err)
	}

	// 4. Provision physical database
	if err := uc.provisioner.CreateDatabase(ctx, org.DBName); err != nil {
		return nil, uc.failProvisioning(ctx, org.ID, org.Slug, err, "provision database")
	}

	// 5. Apply schemas to the new database
	if err := uc.provisioner.MigrateDatabase(ctx, org.DBName); err != nil {
		return nil, uc.failProvisioning(ctx, org.ID, org.Slug, err, "migrate database")
	}
	owner := domain.OwnerData{
		ID:        cmd.OwnerID,
		Email:     cmd.OwnerEmail,
		FirstName: cmd.OwnerFirstName,
		LastName:  cmd.OwnerLastName,
		AvatarURL: cmd.OwnerAvatarURL,
	}
	if err := uc.provisioner.SeedOwner(ctx, org.DBName, owner); err != nil {
		return nil, uc.failProvisioning(ctx, org.ID, org.Slug, err, "seed owner")
	}

	// 8. Add membership to control plane
	if err := uc.repo.AddMembership(ctx, cmd.OwnerID, org.ID, "OWNER"); err != nil {
		return nil, uc.failProvisioning(ctx, org.ID, org.Slug, err, "add owner membership")
	}

	if err := uc.repo.UpdateStatus(ctx, org.ID, domain.StatusActive); err != nil {
		return nil, fmt.Errorf("failed to mark organization as active: %w", err)
	}
	org.Status = domain.StatusActive

	return org, nil
}
