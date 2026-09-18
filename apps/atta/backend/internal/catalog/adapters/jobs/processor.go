package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/atta/internal/catalog/application/commands"
	contractJobs "github.com/atta/internal/catalog/contracts/jobs"
	"github.com/atta/internal/platform/jobs"
	"github.com/atta/internal/platform/outbox/publisher"
	"github.com/atta/internal/platform/tenant"
)

type ProcessCatalogImport struct {
	command *commands.ProcessCatalogImportCommand
}

func NewProcessCatalogImport(command *commands.ProcessCatalogImportCommand) *ProcessCatalogImport {
	if command == nil {
		panic("process catalog import command is required")
	}
	return &ProcessCatalogImport{command: command}
}

func (h *ProcessCatalogImport) JobType() string {
	return contractJobs.CatalogImportRequestedType
}

func (h *ProcessCatalogImport) Scope() jobs.Scope {
	return jobs.ScopeTenant
}

func (h *ProcessCatalogImport) Handle(ctx context.Context, msg jobs.JobMessage) error {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return errors.New("tenant id is required")
	}
	body, err := extractJobPayload(msg.Body)
	if err != nil {
		return err
	}
	decoded, err := contractJobs.UnmarshalCatalogImportRequested(body)
	if err != nil {
		return err
	}
	err = h.command.Execute(ctx, decoded)
	if err != nil {
		log.Printf("Failed to process job %s with ID %s import %s: %v", h.JobType(), msg.MessageID, decoded.ImportID, err)
	}
	return err
}

type ProcessCatalogImportPurge struct {
	command *commands.PlatformPurgeStaleCatalogImportsCommand
}

func NewProcessCatalogImportPurge(command *commands.PlatformPurgeStaleCatalogImportsCommand) *ProcessCatalogImportPurge {
	if command == nil {
		panic("platform purge command is required")
	}
	return &ProcessCatalogImportPurge{command: command}
}

func (h *ProcessCatalogImportPurge) JobType() string {
	return contractJobs.CatalogImportPurgeRequestedType
}

func (h *ProcessCatalogImportPurge) Scope() jobs.Scope {
	return jobs.ScopePlatform
}

func (h *ProcessCatalogImportPurge) Handle(ctx context.Context, msg jobs.JobMessage) error {
	err := h.command.Execute(publisher.WithCorrelationID(ctx, parentCorrelation(msg)))
	if err != nil {
		log.Printf("Failed to process job %s with ID %s: %v", h.JobType(), msg.MessageID, err)
	}
	return err
}

func parentCorrelation(msg jobs.JobMessage) string {
	if msg.CorrelationID != "" {
		return msg.CorrelationID
	}
	return msg.MessageID
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
