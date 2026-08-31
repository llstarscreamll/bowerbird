package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	catalogModule "github.com/bowerbird/internal/catalog"
	connectionsModule "github.com/bowerbird/internal/connections"
	entitlementsModule "github.com/bowerbird/internal/entitlements"
	inboxModule "github.com/bowerbird/internal/inbox"
	invoicesModule "github.com/bowerbird/internal/invoices"
	invoicesJobs "github.com/bowerbird/internal/invoices/adapters/jobs"
	invoiceLinking "github.com/bowerbird/internal/invoices/adapters/linking"
	partiesModule "github.com/bowerbird/internal/parties"
	"github.com/bowerbird/internal/platform"
	platformCrypto "github.com/bowerbird/internal/platform/crypto"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	secretsModule "github.com/bowerbird/internal/secrets"
)

var jobHandler platformJobs.Handler

func init() {
	platformModule, err := platform.NewModule(context.Background())
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}

	cfg := platformModule.Config
	entitlementsApp := entitlementsModule.NewApplication(platformModule.ControlDB)

	secretsCipher, err := platformCrypto.NewAESCipherFromBase64Key(cfg.TenantSecretsEncryptionKey)
	if err != nil {
		log.Fatalf("failed to create tenant secrets cipher at boot: %v", err)
	}
	secretsApp := secretsModule.NewApplication(platformModule.TenantRegistry, secretsCipher)
	documentPasswordResolver := invoicesModule.NewSecretsPasswordAdapter(secretsModule.NewDocumentPasswordResolver(secretsApp))

	partiesApp := partiesModule.NewApplication(platformModule.TenantRegistry)
	catalogApp := catalogModule.NewApplication(platformModule.TenantRegistry)

	invoicesApp := invoicesModule.NewApplication(
		cfg,
		platformModule.EventBus,
		platformModule.JobQueue,
		platformModule.FileStore,
		platformModule.TenantRegistry,
		documentPasswordResolver,
		invoiceLinking.NewPartyResolverAdapter(partiesModule.NewIssuerPartyLookup(partiesApp)),
		invoiceLinking.NewCatalogResolverAdapter(catalogApp),
		invoiceLinking.NewCatalogNamesAdapter(catalogApp),
		invoiceLinking.NewCatalogMatchingAdapter(catalogApp),
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
