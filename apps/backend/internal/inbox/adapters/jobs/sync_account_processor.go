package jobs

import (
	"context"
	"errors"
	"log"

	awsEvents "github.com/aws/aws-lambda-go/events"
	entitlementsDomain "github.com/bowerbird/internal/entitlements/domain"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	inboxPorts "github.com/bowerbird/internal/inbox/application/ports"
	inboxJobs "github.com/bowerbird/internal/inbox/contracts/jobs"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/tenant"
)

type ProcessInboxSyncAccount struct {
	command  *inboxCommands.SyncAccountCommand
	features inboxPorts.FeatureChecker
}

func NewProcessInboxSyncAccount(command *inboxCommands.SyncAccountCommand, features inboxPorts.FeatureChecker) *ProcessInboxSyncAccount {
	return &ProcessInboxSyncAccount{command: command, features: features}
}

func (h *ProcessInboxSyncAccount) JobType() string {
	return inboxJobs.InboxSyncAccountType
}

func (h *ProcessInboxSyncAccount) HandleSQS(ctx context.Context, msg awsEvents.SQSMessage) error {
	if h.command == nil {
		return errors.New("sync account command is required")
	}
	if _, err := tenant.TenantIDFromContext(ctx); err != nil {
		return errors.New("tenant id is required")
	}
	if h.features != nil {
		if err := h.features.RequireAny(ctx, entitlementsDomain.FeatureMailInbox, entitlementsDomain.FeatureInvoicingCaptureFromEmail); err != nil {
			var appErr *appErrors.AppError
			if errors.As(err, &appErr) && appErr.Code == appErrors.CodeForbidden {
				log.Printf("skipping inbox sync job %s: feature not available", msg.MessageId)
				return nil
			}
			return err
		}
	}

	decoded, err := inboxJobs.UnmarshalInboxSyncAccount([]byte(msg.Body))
	if err != nil {
		return err
	}

	err = h.command.Execute(ctx, inboxCommands.SyncAccountCommandInput{AccountID: decoded.AccountID})
	if err != nil {
		log.Printf("Failed to process job %s with ID %s: %v", h.JobType(), msg.MessageId, err)
	}
	return err
}
