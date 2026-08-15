package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	connectionsModule "github.com/bowerbird/internal/connections"
	entitlementsModule "github.com/bowerbird/internal/entitlements"
	inboxModule "github.com/bowerbird/internal/inbox"
	invoicesModule "github.com/bowerbird/internal/invoices"
	invoicesJobs "github.com/bowerbird/internal/invoices/adapters/jobs"
	"github.com/bowerbird/internal/platform"
	platformCrypto "github.com/bowerbird/internal/platform/crypto"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
)

var jobHandler platformJobs.Handler

func init() {
	platformModule, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}

	cfg := platformModule.Config
	entitlementsApp := entitlementsModule.NewApplication(platformModule.ControlDB)
	invoicesApp := invoicesModule.NewApplication(
		cfg,
		platformModule.EventBus,
		platformModule.JobQueue,
		platformModule.FileStore,
		platformModule.TenantRegistry,
	)

	processorCommand := invoicesJobs.NewInvoiceExtractionRequestedProcessor(
		invoicesApp.Commands.ProcessInvoiceExtractionJob,
	)

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
		platformModule.JobQueue,
	)

	jobHandler = platformJobs.NewHandler(processorCommand, inboxModule.NewSyncAccountProcessor(inboxApp, entitlementsApp))
}

func handle(ctx context.Context, event events.SQSEvent) error {
	return jobHandler.HandleSQSEvent(ctx, event)
}

func main() {
	lambda.Start(handle)
}
