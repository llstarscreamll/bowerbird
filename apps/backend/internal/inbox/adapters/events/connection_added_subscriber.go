package events

import (
	"context"
	"errors"
	"log"

	contractevents "github.com/bowerbird/internal/contracts/events"
	entitlementsapi "github.com/bowerbird/internal/entitlements/api"
	inboxCommands "github.com/bowerbird/internal/inbox/application/commands"
	appErrors "github.com/bowerbird/internal/platform/errors"
	platformEvents "github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/tenant"
)

type ConnectionAddedSubscriber struct {
	dispatcher inboxCommands.SyncAccountJobDispatcher
	features   entitlementsapi.Features
}

func NewConnectionAddedSubscriber(dispatcher inboxCommands.SyncAccountJobDispatcher, features entitlementsapi.Features) *ConnectionAddedSubscriber {
	if dispatcher == nil {
		panic("sync account job dispatcher is required")
	}
	if features == nil {
		panic("feature checker is required")
	}
	return &ConnectionAddedSubscriber{dispatcher: dispatcher, features: features}
}

func (s *ConnectionAddedSubscriber) DetailType() string {
	return contractevents.ConnectionAddedDetailType
}

func (s *ConnectionAddedSubscriber) Handle(ctx context.Context, event platformEvents.IntegrationEvent) error {
	decoded, err := contractevents.UnmarshalConnectionAdded(event.Detail)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantSlug)
	if err := s.features.RequireAny(msgCtx, entitlementsapi.FeatureMailInbox, entitlementsapi.FeatureInvoicingCaptureFromEmail); err != nil {
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) && appErr.Code == appErrors.CodeForbidden {
			log.Printf("skipping inbox sync after connection added: feature not available")
			return nil
		}
		return err
	}

	return s.dispatcher.DispatchSyncAccount(msgCtx, inboxCommands.SyncAccountJob{
		TenantID:  decoded.TenantSlug,
		AccountID: decoded.ConnectionID,
		Provider:  decoded.Provider,
	})
}
