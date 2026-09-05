package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	entitlementsapi "github.com/bowerbird/internal/entitlements/api"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	inboxJobs "github.com/bowerbird/internal/inbox/contracts/jobs"
	appErrors "github.com/bowerbird/internal/platform/errors"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
)

type ProcessInboxSyncAccount struct {
	command  *inboxCommands.SyncAccountCommand
	features entitlementsapi.Features
}

func NewProcessInboxSyncAccount(command *inboxCommands.SyncAccountCommand, features entitlementsapi.Features) *ProcessInboxSyncAccount {
	if command == nil {
		panic("sync account command is required")
	}
	if features == nil {
		panic("feature checker is required")
	}
	return &ProcessInboxSyncAccount{command: command, features: features}
}

func (h *ProcessInboxSyncAccount) JobType() string {
	return inboxJobs.InboxSyncAccountType
}

func (h *ProcessInboxSyncAccount) Handle(ctx context.Context, msg platformJobs.JobMessage) error {
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return errors.New("tenant id is required")
	}
	if err := h.features.RequireAny(ctx, entitlementsapi.FeatureMailInbox, entitlementsapi.FeatureInvoicingCaptureFromEmail); err != nil {
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) && appErr.Code == appErrors.CodeForbidden {
			log.Printf("skipping inbox sync job %s: feature not available", msg.MessageID)
			return nil
		}
		return err
	}

	body, err := extractJobPayload(msg.Body)
	if err != nil {
		return err
	}

	decoded, err := inboxJobs.UnmarshalInboxSyncAccount(body)
	if err != nil {
		return err
	}

	err = h.command.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: decoded.AccountID})
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
