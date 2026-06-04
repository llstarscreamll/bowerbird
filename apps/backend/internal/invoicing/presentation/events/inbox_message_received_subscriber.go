package events

import (
	"context"

	awsEvents "github.com/aws/aws-lambda-go/events"
	contractEvents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	invoicingApp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type InboxMessageReceivedSubscriber struct {
	command *invoicingApp.CheckInboxMessageForInvoiceCandidatesCommand
}

func NewInboxMessageReceivedSubscriber(command *invoicingApp.CheckInboxMessageForInvoiceCandidatesCommand) *InboxMessageReceivedSubscriber {
	return &InboxMessageReceivedSubscriber{command: command}
}

func (s *InboxMessageReceivedSubscriber) DetailType() string {
	return contractEvents.InboxMessageReceivedDetailType
}

func (s *InboxMessageReceivedSubscriber) HandleEventBridge(ctx context.Context, event awsEvents.CloudWatchEvent) error {
	if s.command == nil {
		return nil
	}

	decoded, err := contractEvents.DecodeInboxMessageReceivedFromCloudWatchEvent(event)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantSlug)
	return s.command.Execute(msgCtx, decoded)
}
