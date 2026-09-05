package messaging

import (
	catalogModule "github.com/bowerbird/internal/catalog"
	connectionsModule "github.com/bowerbird/internal/connections"
	entitlementsModule "github.com/bowerbird/internal/entitlements"
	inboxModule "github.com/bowerbird/internal/inbox"
	invoicesModule "github.com/bowerbird/internal/invoices"
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

	secretsCipher, err := platformCrypto.NewAESCipherFromBase64Key(cfg.TenantSecretsEncryptionKey)
	if err != nil {
		panic("tenant secrets cipher is required")
	}
	secretsApp := secretsModule.NewApplication(platformModule.TenantRegistry, secretsCipher)

	partiesApp := partiesModule.NewApplication(platformModule.TenantRegistry)
	catalogApp := catalogModule.NewApplication(platformModule.TenantRegistry)

	invoicingApp := invoicesModule.NewApplication(
		cfg,
		platformModule.EventBus,
		platformModule.TaskQueue,
		platformModule.FileStore,
		platformModule.TenantRegistry,
		secretsModule.NewDocumentPasswordResolver(secretsApp),
		catalogModule.NewInvoiceSupport(catalogApp),
		partiesModule.NewIssuerPartyLookup(partiesApp),
	)

	cipher, err := platformCrypto.NewAESCipherFromBase64Key(cfg.InboxCredentialsEncryptionKey)
	if err != nil {
		panic("inbox credentials cipher is required")
	}
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

	invoiceEvents := invoicesModule.RegisterEvents(invoicingApp)
	invoiceJobs := invoicesModule.RegisterJobs(invoicingApp)
	inboxEvents := inboxModule.RegisterEvents(entitlementsApp, platformModule.TaskQueue)
	inboxJobs := inboxModule.RegisterJobs(inboxApp, entitlementsApp)
	sweeper := outboxSweeper.NewHandler(platformModule.TenantRegistry, 0)

	eventHandlers := append(append([]platformEvents.IntegrationEventHandler{}, invoiceEvents...), inboxEvents...)
	jobHandlers := append(append(append([]platformJobs.JobHandler{}, invoiceJobs...), inboxJobs...), sweeper)
	verifier := attestation.NewVerifier(cfg.MessagingAttestationSecret)

	return Handlers{
		Events: platformEvents.NewRouter(verifier, eventHandlers...),
		Jobs:   platformJobs.NewRouter(verifier, jobHandlers...),
	}
}
