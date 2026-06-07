package application

import (
	"context"

	"github.com/bowerbird/internal/organization/application/commands"
	"github.com/bowerbird/internal/organization/application/ports"
	"github.com/bowerbird/internal/organization/domain"
)

var ErrSlugAlreadyExists = commands.ErrSlugAlreadyExists

type CreateOrganizationCommand = commands.CreateOrganizationInput

type CreateOrganizationUseCase struct {
	command *commands.CreateOrganizationCommand
}

func NewCreateOrganizationUseCase(repo ports.OrganizationRepository, provisioner ports.Provisioner) *CreateOrganizationUseCase {
	return &CreateOrganizationUseCase{command: commands.NewCreateOrganizationCommand(repo, provisioner)}
}

func NewCreateOrganizationUseCaseFromCommand(command *commands.CreateOrganizationCommand) *CreateOrganizationUseCase {
	if command == nil {
		panic("create organization command is required")
	}

	return &CreateOrganizationUseCase{command: command}
}

func (uc *CreateOrganizationUseCase) Execute(ctx context.Context, input CreateOrganizationCommand) (*domain.Organization, error) {
	return uc.command.Execute(ctx, input)
}
