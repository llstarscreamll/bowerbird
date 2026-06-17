package handlers

import (
	"context"
	"errors"
	"log"

	awsEvents "github.com/aws/aws-lambda-go/events"
	commands "github.com/bowerbird/internal/invoices/application/commands"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/platform/tenant"
)

type ProcessInvoiceExtractionFromFiles struct {
	command *commands.CreateInvoicesFromFilesCommand
}

func NewProcessInvoiceExtractionFromFiles(command *commands.CreateInvoicesFromFilesCommand) *ProcessInvoiceExtractionFromFiles {
	if command == nil {
		panic("command is required")
	}

	return &ProcessInvoiceExtractionFromFiles{command: command}
}

func (h *ProcessInvoiceExtractionFromFiles) JobType() string {
	return contractJobs.InvoiceExtractionRequestedType
}

func (h *ProcessInvoiceExtractionFromFiles) HandleSQS(ctx context.Context, msg awsEvents.SQSMessage) error {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return errors.New("tenant id is required")
	}

	decoded, err := contractJobs.UnmarshalInvoiceExtractionRequested([]byte(msg.Body))
	if err != nil {
		return err
	}

	err = h.command.Execute(ctx, decoded)

	if err != nil {
		log.Printf("Failed to process job %s with ID %s: %v", h.JobType(), msg.MessageId, err)
	}

	return err
}
