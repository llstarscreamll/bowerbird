package events

import (
	"context"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/bowerbird/internal/contracts/events"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	"github.com/bowerbird/internal/platform/tenant"
)

type ConnectionAddedSubscriber struct {
	command *inboxCommands.SyncAccountCommand
}

func NewConnectionAddedSubscriber(command *inboxCommands.SyncAccountCommand) *ConnectionAddedSubscriber {
	return &ConnectionAddedSubscriber{command: command}
}

func (s *ConnectionAddedSubscriber) DetailType() string {
	return contractevents.ConnectionAddedDetailType
}

func (s *ConnectionAddedSubscriber) HandleEventBridge(ctx context.Context, event awsevents.CloudWatchEvent) error {
	if s.command == nil {
		return nil
	}

	decoded, err := contractevents.UnmarshalConnectionAdded(event.Detail)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantSlug)
	return s.command.Execute(msgCtx, inboxCommands.SyncAccountCommandInput{AccountID: decoded.ConnectionID})
}
