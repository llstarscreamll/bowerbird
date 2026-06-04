package events

import (
	"context"

	awsEvents "github.com/aws/aws-lambda-go/events"
	contractEvents "github.com/money-path/bowerbird/apps/backend/internal/contracts/events"
	invoicingApp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
)

type InvoiceExtractionRequestedSubscriber struct {
	command *invoicingApp.ExtractInvoiceCommand
}

func NewInvoiceExtractionRequestedSubscriber(command *invoicingApp.ExtractInvoiceCommand) *InvoiceExtractionRequestedSubscriber {
	return &InvoiceExtractionRequestedSubscriber{command: command}
}

func (s *InvoiceExtractionRequestedSubscriber) DetailType() string {
	return contractEvents.InvoiceExtractionRequestedDetailType
}

func (s *InvoiceExtractionRequestedSubscriber) HandleEventBridge(ctx context.Context, event awsEvents.CloudWatchEvent) error {
	if s.command == nil {
		return nil
	}

	decoded, err := contractEvents.DecodeInvoiceExtractionRequestedFromCloudWatchEvent(event)
	if err != nil {
		return err
	}

	msgCtx := tenant.WithTenantID(ctx, decoded.TenantSlug)
	_, err = s.command.Execute(msgCtx, decoded)
	return err
}
