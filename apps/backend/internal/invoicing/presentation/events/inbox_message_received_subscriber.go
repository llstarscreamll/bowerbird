package events

import (
	"context"

	awsEvents "github.com/aws/aws-lambda-go/events"
	contractEvents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	invoicingApp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type InboxMessageReceivedSubscriber struct {
	useCase *invoicingApp.ProcessInboxEventCommand
}

func NewInboxMessageReceivedSubscriber(useCase *invoicingApp.ProcessInboxEventCommand) *InboxMessageReceivedSubscriber {
	return &InboxMessageReceivedSubscriber{useCase: useCase}
}

func (s *InboxMessageReceivedSubscriber) DetailType() string {
	return contractEvents.InboxMessageReceivedDetailType
}

func (s *InboxMessageReceivedSubscriber) HandleEventBridge(ctx context.Context, event awsEvents.CloudWatchEvent) error {
	decoded, err := contractEvents.DecodeInboxMessageReceivedFromCloudWatchEvent(event)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantSlug)
	return s.useCase.Execute(msgCtx, decoded)
}
