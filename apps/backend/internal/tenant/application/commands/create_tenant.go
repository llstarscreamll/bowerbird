package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/tenant/application/ports"
	"github.com/bowerbird/internal/tenant/domain"
)

var ErrSlugAlreadyExists = errors.New("tenant slug already exists")

type CreateTenantInput struct {
	Name           string
	Slug           string
	OwnerID        string
	OwnerEmail     string
	OwnerFirstName string
	OwnerLastName  string
	OwnerAvatarURL string
}

type CreateTenantCommand struct {
	repo        ports.TenantRepository
	provisioner ports.Provisioner
	defaults    ports.DefaultPackApplier
}

func NewCreateTenantCommand(repo ports.TenantRepository, provisioner ports.Provisioner, defaults ports.DefaultPackApplier) *CreateTenantCommand {
	return &CreateTenantCommand{repo: repo, provisioner: provisioner, defaults: defaults}
}

func (cmd *CreateTenantCommand) failProvisioning(ctx context.Context, orgID, slug string, cause error, step string) error {
	if markErr := cmd.repo.UpdateStatus(ctx, orgID, domain.StatusFailed); markErr != nil {
		return errors.Join(
			fmt.Errorf("failed to %s for %s: %w", step, slug, cause),
			fmt.Errorf("failed to mark tenant as failed: %w", markErr),
		)
	}

	return fmt.Errorf("failed to %s for %s: %w", step, slug, cause)
}

func (cmd *CreateTenantCommand) Execute(ctx context.Context, input CreateTenantInput) (*domain.Tenant, error) {
	org, err := domain.NewTenant(input.Name, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant data: %w", err)
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
		return nil, fmt.Errorf("failed to register tenant in control plane: %w", err)
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
		return nil, fmt.Errorf("failed to mark tenant as active: %w", err)
	}
	org.Status = domain.StatusActive

	if cmd.defaults != nil {
		if err := cmd.defaults.ApplyDefaultPack(ctx, org.ID, input.OwnerID); err != nil {
			return nil, fmt.Errorf("failed to apply default entitlements: %w", err)
		}
	}

	return org, nil
}
