package handlers

import (
	"context"
	"errors"

	commands "github.com/atta/internal/invoices/application/commands"
	contractJobs "github.com/atta/internal/invoices/contracts/jobs"
	"github.com/atta/internal/platform/jobs"
	"github.com/atta/internal/platform/tenant"
)

type ProcessInvoiceInboxBackfill struct {
	command *commands.BackfillInboxInvoicesCommand
}

func NewProcessInvoiceInboxBackfill(command *commands.BackfillInboxInvoicesCommand) *ProcessInvoiceInboxBackfill {
	if command == nil {
		panic("backfill command is required")
	}
	return &ProcessInvoiceInboxBackfill{command: command}
}

func (h *ProcessInvoiceInboxBackfill) JobType() string {
	return contractJobs.InvoiceInboxBackfillRequestedType
}

func (h *ProcessInvoiceInboxBackfill) Scope() jobs.Scope {
	return jobs.ScopeTenant
}

func (h *ProcessInvoiceInboxBackfill) Handle(ctx context.Context, msg jobs.JobMessage) error {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return errors.New("tenant id is required")
	}
	body, err := extractJobPayload(msg.Body)
	if err != nil {
		return err
	}
	decoded, err := contractJobs.UnmarshalInboxBackfillRequested(body)
	if err != nil {
		return err
	}
	return h.command.Execute(ctx, decoded)
}
