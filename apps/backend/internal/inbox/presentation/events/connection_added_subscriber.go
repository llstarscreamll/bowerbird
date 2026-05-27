package events

import (
	"context"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type ConnectionAddedSubscriber struct {
	useCase *application.InitialSyncUseCase
}

func NewConnectionAddedSubscriber(useCase *application.InitialSyncUseCase) *ConnectionAddedSubscriber {
	return &ConnectionAddedSubscriber{useCase: useCase}
}

func (s *ConnectionAddedSubscriber) DetailType() string {
	return contractevents.ConnectionAddedDetailType
}

func (s *ConnectionAddedSubscriber) HandleEventBridge(ctx context.Context, event awsevents.CloudWatchEvent) error {
	decoded, err := contractevents.UnmarshalConnectionAdded(event.Detail)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantSlug(ctx, decoded.TenantSlug)
	return s.useCase.Process(msgCtx, decoded.TenantSlug, decoded.ConnectionID, decoded.Provider)
}
