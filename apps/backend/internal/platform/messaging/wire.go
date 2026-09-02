package messaging

import (
	catalogModule "github.com/bowerbird/internal/catalog"
	connectionsModule "github.com/bowerbird/internal/connections"
	entitlementsModule "github.com/bowerbird/internal/entitlements"
	inboxModule "github.com/bowerbird/internal/inbox"
	invoicesModule "github.com/bowerbird/internal/invoices"
	invoiceLinking "github.com/bowerbird/internal/invoices/adapters/linking"
	partiesModule "github.com/bowerbird/internal/parties"
	"github.com/bowerbird/internal/platform"
	platformCrypto "github.com/bowerbird/internal/platform/crypto"
	platformEvents "github.com/bowerbird/internal/platform/events"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	outboxSweeper "github.com/bowerbird/internal/platform/outbox/sweeper"
	secretsModule "github.com/bowerbird/internal/secrets"
)

type Handlers struct {
	Events platformEvents.Router
	Jobs   platformJobs.Router
}

func WireMessagingHandlers(platformModule *platform.Dependencies) Handlers {
	cfg := platformModule.Config
	entitlementsApp := entitlementsModule.NewApplication(platformModule.ControlDB)

	secretsCipher, _ := platformCrypto.NewAESCipherFromBase64Key(cfg.TenantSecretsEncryptionKey)
	secretsApp := secretsModule.NewApplication(platformModule.TenantRegistry, secretsCipher)
	documentPasswordResolver := invoicesModule.NewSecretsPasswordAdapter(secretsModule.NewDocumentPasswordResolver(secretsApp))

	partiesApp := partiesModule.NewApplication(platformModule.TenantRegistry)
	catalogApp := catalogModule.NewApplication(platformModule.TenantRegistry)

	invoicingApp := invoicesModule.NewApplication(
		cfg,
		platformModule.EventBus,
		platformModule.TaskQueue,
		platformModule.FileStore,
		platformModule.TenantRegistry,
		documentPasswordResolver,
		invoiceLinking.NewPartyResolverAdapter(partiesModule.NewIssuerPartyLookup(partiesApp)),
		invoiceLinking.NewCatalogResolverAdapter(catalogApp),
		invoiceLinking.NewCatalogNamesAdapter(catalogApp),
		invoiceLinking.NewCatalogMatchingAdapter(catalogApp),
	)

	cipher, _ := platformCrypto.NewAESCipherFromBase64Key(cfg.InboxCredentialsEncryptionKey)
	connectionsApp := connectionsModule.NewApplication(platformModule.TenantRegistry, cipher)
	connectionsService := connectionsModule.NewInternalService(connectionsApp)

	inboxApp := inboxModule.NewApplication(
		cfg,
		connectionsService,
		platformModule.EventBus,
		platformModule.FileStore,
		platformModule.TenantRegistry,
		platformModule.TaskQueue,
	)

	invoiceEvents, invoiceJobs := invoicesModule.RegisterMessaging(invoicingApp)
	inboxEvents, inboxJobs := inboxModule.RegisterMessaging(inboxApp, entitlementsApp, platformModule.TaskQueue)
	sweeper := outboxSweeper.NewHandler(platformModule.TenantRegistry, cfg.DefaultTenantSlug, 0)

	eventHandlers := append(append([]platformEvents.IntegrationEventHandler{}, invoiceEvents...), inboxEvents...)
	jobHandlers := append(append(append([]platformJobs.JobHandler{}, invoiceJobs...), inboxJobs...), sweeper)
	verifier := attestation.NewVerifier(cfg.MessagingAttestationSecret)

	return Handlers{
		Events: platformEvents.NewRouter(verifier, eventHandlers...),
		Jobs:   platformJobs.NewRouter(verifier, jobHandlers...),
	}
}
