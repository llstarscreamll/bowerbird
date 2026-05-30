package events

import (
	"context"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type ConnectionAddedSubscriber struct {
	command *application.SyncAccountCommand
}

func NewConnectionAddedSubscriber(command *application.SyncAccountCommand) *ConnectionAddedSubscriber {
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
	return s.command.Execute(msgCtx, application.SyncAccountCommandInput{AccountID: decoded.ConnectionID})
}
