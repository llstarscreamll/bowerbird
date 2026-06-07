package handlers

import (
	"context"
	"errors"

	awsEvents "github.com/aws/aws-lambda-go/events"
	commands "github.com/bowerbird/internal/invoices/application/commands"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/platform/tenant"
)

type ProcessInvoiceExtractionRequested struct {
	command *commands.ProcessInvoiceExtractionJobCommand
}

func NewProcessInvoiceExtractionRequested(command *commands.ProcessInvoiceExtractionJobCommand) *ProcessInvoiceExtractionRequested {
	if command == nil {
		panic("command is required")
	}

	return &ProcessInvoiceExtractionRequested{command: command}
}

func (h *ProcessInvoiceExtractionRequested) JobType() string {
	return contractJobs.InvoiceExtractionRequestedType
}

func (h *ProcessInvoiceExtractionRequested) HandleSQS(ctx context.Context, message awsEvents.SQSMessage) error {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return errors.New("tenant id is required")
	}

	decoded, err := contractJobs.UnmarshalInvoiceExtractionRequested([]byte(message.Body))
	if err != nil {
		return err
	}

	_, err = h.command.Execute(ctx, decoded)
	return err
}
