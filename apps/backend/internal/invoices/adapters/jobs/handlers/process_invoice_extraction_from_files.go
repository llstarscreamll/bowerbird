package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	commands "github.com/bowerbird/internal/invoices/application/commands"
	contractJobs "github.com/bowerbird/internal/invoices/contracts/jobs"
	"github.com/bowerbird/internal/platform/jobs"
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

func (h *ProcessInvoiceExtractionFromFiles) Handle(ctx context.Context, msg jobs.JobMessage) error {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return errors.New("tenant id is required")
	}

	body, err := extractJobPayload(msg.Body)
	if err != nil {
		return err
	}

	decoded, err := contractJobs.UnmarshalInvoiceExtractionRequested(body)
	if err != nil {
		return err
	}

	err = h.command.Execute(ctx, decoded)
	if err != nil {
		log.Printf("Failed to process job %s with ID %s: %v", h.JobType(), msg.MessageID, err)
	}
	return err
}

func extractJobPayload(body []byte) ([]byte, error) {
	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Payload) > 0 {
		return envelope.Payload, nil
	}
	return body, nil
}
