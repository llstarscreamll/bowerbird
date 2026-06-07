package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	connectionsModule "github.com/bowerbird/internal/connections"
	inboxModule "github.com/bowerbird/internal/inbox"
	invoicesModule "github.com/bowerbird/internal/invoices"
	invoicesEvents "github.com/bowerbird/internal/invoices/adapters/events"
	"github.com/bowerbird/internal/platform"
	platformCrypto "github.com/bowerbird/internal/platform/crypto"
	platformEvents "github.com/bowerbird/internal/platform/events"
)

var eventHandler platformEvents.EventHandler

func init() {
	platformModule, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}

	cfg := platformModule.Config
	invoicingApp := invoicesModule.NewApplication(
		cfg,
		platformModule.EventBus,
		platformModule.JobQueue,
		platformModule.FileStore,
		platformModule.TenantRegistry,
	)
	inboxMessageSubscriber := invoicesEvents.NewInboxMessageReceivedSubscriber(invoicingApp.Commands.CreateInvoicesFromInboxMessage)

	cipher, err := platformCrypto.NewAESCipherFromBase64Key(cfg.InboxCredentialsEncryptionKey)
	if err != nil {
		log.Fatalf("failed to create inbox credentials cipher at boot: %v", err)
	}

	connectionsApp := connectionsModule.NewApplication(platformModule.TenantRegistry, cipher)
	connectionsService := connectionsModule.NewInternalService(connectionsApp)

	inboxApp := inboxModule.NewApplication(
		cfg,
		connectionsService,
		platformModule.EventBus,
		platformModule.FileStore,
		platformModule.TenantRegistry,
	)
	connectionAddedSubscriber := inboxModule.NewConnectionAddedSubscriber(inboxApp)
	eventHandler = platformEvents.NewEventHandler(inboxMessageSubscriber, connectionAddedSubscriber)
}

func handle(ctx context.Context, event events.CloudWatchEvent) error {
	return eventHandler.HandleEventBridgeEvent(ctx, event)
}

func main() {
	lambda.Start(handle)
}
