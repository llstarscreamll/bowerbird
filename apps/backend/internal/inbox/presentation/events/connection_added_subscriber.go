package events

import (
	"context"
	"errors"
	"log"

	contractevents "github.com/bowerbird/internal/contracts/events"
	entitlementsDomain "github.com/bowerbird/internal/entitlements/domain"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	inboxPorts "github.com/bowerbird/internal/inbox/application/ports"
	appErrors "github.com/bowerbird/internal/platform/errors"
	platformEvents "github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/tenant"
)

type ConnectionAddedSubscriber struct {
	dispatcher inboxCommands.SyncAccountJobDispatcher
	features   inboxPorts.FeatureChecker
}

func NewConnectionAddedSubscriber(dispatcher inboxCommands.SyncAccountJobDispatcher, features inboxPorts.FeatureChecker) *ConnectionAddedSubscriber {
	return &ConnectionAddedSubscriber{dispatcher: dispatcher, features: features}
}

func (s *ConnectionAddedSubscriber) DetailType() string {
	return contractevents.ConnectionAddedDetailType
}

func (s *ConnectionAddedSubscriber) Handle(ctx context.Context, event platformEvents.IntegrationEvent) error {
	if s.dispatcher == nil {
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

	return s.dispatcher.DispatchSyncAccount(msgCtx, inboxCommands.SyncAccountJob{
		TenantID:  decoded.TenantSlug,
		AccountID: decoded.ConnectionID,
		Provider:  decoded.Provider,
	})
}
