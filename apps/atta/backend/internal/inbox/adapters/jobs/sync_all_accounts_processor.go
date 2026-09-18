package jobs

import (
	"context"
	"log"

	inboxCommands "github.com/atta/internal/inbox/application/commands"
	inboxJobs "github.com/atta/internal/inbox/contracts/jobs"
	platformJobs "github.com/atta/internal/platform/jobs"
	"github.com/atta/internal/platform/outbox/publisher"
)

type ProcessInboxSyncAllAccounts struct {
	command *inboxCommands.PlatformSyncAllAccountsCommand
}

func NewProcessInboxSyncAllAccounts(command *inboxCommands.PlatformSyncAllAccountsCommand) *ProcessInboxSyncAllAccounts {
	if command == nil {
		panic("platform sync all accounts command is required")
	}
	return &ProcessInboxSyncAllAccounts{command: command}
}

func (h *ProcessInboxSyncAllAccounts) JobType() string {
	return inboxJobs.InboxSyncAllAccountsType
}

func (h *ProcessInboxSyncAllAccounts) Scope() platformJobs.Scope {
	return platformJobs.ScopePlatform
}

func (h *ProcessInboxSyncAllAccounts) Handle(ctx context.Context, msg platformJobs.JobMessage) error {
	err := h.command.Execute(publisher.WithCorrelationID(ctx, parentCorrelation(msg)))
	if err != nil {
		log.Printf("Failed to process job %s with ID %s: %v", h.JobType(), msg.MessageID, err)
	}
	return err
}

func parentCorrelation(msg platformJobs.JobMessage) string {
	if msg.CorrelationID != "" {
		return msg.CorrelationID
	}
	return msg.MessageID
}
