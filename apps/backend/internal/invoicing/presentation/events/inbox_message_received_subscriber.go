package events

import (
	"context"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	invoicingapp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type InboxMessageReceivedSubscriber struct {
	useCase *invoicingapp.ProcessInboxEventUseCase
}

func NewInboxMessageReceivedSubscriber(useCase *invoicingapp.ProcessInboxEventUseCase) *InboxMessageReceivedSubscriber {
	return &InboxMessageReceivedSubscriber{useCase: useCase}
}

func (s *InboxMessageReceivedSubscriber) DetailType() string {
	return contractevents.InboxMessageReceivedDetailType
}

func (s *InboxMessageReceivedSubscriber) HandleEventBridge(ctx context.Context, event awsevents.CloudWatchEvent) error {
	decoded, err := contractevents.DecodeInboxMessageReceivedFromCloudWatchEvent(event)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantSlug)
	return s.useCase.Process(msgCtx, decoded)
}
