package events

import (
	"context"
	"errors"
	"log"

	awsevents "github.com/aws/aws-lambda-go/events"
	contractevents "github.com/bowerbird/internal/contracts/events"
	entitlementsDomain "github.com/bowerbird/internal/entitlements/domain"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	inboxPorts "github.com/bowerbird/internal/inbox/application/ports"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/tenant"
)

type ConnectionAddedSubscriber struct {
	command  *inboxCommands.SyncAccountCommand
	features inboxPorts.FeatureChecker
}

func NewConnectionAddedSubscriber(command *inboxCommands.SyncAccountCommand, features inboxPorts.FeatureChecker) *ConnectionAddedSubscriber {
	return &ConnectionAddedSubscriber{command: command, features: features}
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
	if s.features != nil {
		if err := s.features.RequireAny(msgCtx, entitlementsDomain.FeatureMailInbox, entitlementsDomain.FeatureInvoicingCaptureFromEmail); err != nil {
			var appErr *appErrors.AppError
			if errors.As(err, &appErr) && appErr.Code == appErrors.CodeForbidden {
				log.Printf("skipping inbox sync after connection added: feature not available")
				return nil
			}
			return err
		}
	}
	return s.command.Execute(msgCtx, inboxCommands.SyncAccountCommandInput{AccountID: decoded.ConnectionID})
}
