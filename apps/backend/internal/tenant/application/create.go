package application

import (
	"context"

	"github.com/bowerbird/internal/tenant/application/commands"
	"github.com/bowerbird/internal/tenant/application/ports"
	"github.com/bowerbird/internal/tenant/domain"
)

var ErrSlugAlreadyExists = commands.ErrSlugAlreadyExists

type CreateTenantCommand = commands.CreateTenantInput

type CreateTenantUseCase struct {
	command *commands.CreateTenantCommand
}

func NewCreateTenantUseCase(repo ports.TenantRepository, provisioner ports.Provisioner) *CreateTenantUseCase {
	return &CreateTenantUseCase{command: commands.NewCreateTenantCommand(repo, provisioner, nil)}
}

func NewCreateTenantUseCaseFromCommand(command *commands.CreateTenantCommand) *CreateTenantUseCase {
	if command == nil {
		panic("create tenant command is required")
	}

	return &CreateTenantUseCase{command: command}
}

func (uc *CreateTenantUseCase) Execute(ctx context.Context, input CreateTenantCommand) (*domain.Tenant, error) {
	return uc.command.Execute(ctx, input)
}
