package messaging

import (
	catalogModule "github.com/bowerbird/internal/catalog"
	connectionsModule "github.com/bowerbird/internal/connections"
	entitlementsModule "github.com/bowerbird/internal/entitlements"
	inboxModule "github.com/bowerbird/internal/inbox"
	invoicesModule "github.com/bowerbird/internal/invoices"
	partiesModule "github.com/bowerbird/internal/parties"
	"github.com/bowerbird/internal/platform"
	awsConfig "github.com/bowerbird/internal/platform/awsconfig"
	"github.com/bowerbird/internal/platform/config"
	platformCrypto "github.com/bowerbird/internal/platform/crypto"
	platformEvents "github.com/bowerbird/internal/platform/events"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/messaging/attestation"
	"github.com/bowerbird/internal/platform/outbox/relay"
	"github.com/bowerbird/internal/platform/outbox/relay/broker"
	awsbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/aws"
	rabbitmqbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/rabbitmq"
	outboxSweeper "github.com/bowerbird/internal/platform/outbox/sweeper"
	"github.com/bowerbird/internal/platform/scheduler"
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
	tenantLister := relay.NewControlPlaneTenantLister(platformModule.ControlDB)
	inboxJobs := inboxModule.RegisterJobs(inboxApp, entitlementsApp, tenantLister)
	sweeper := outboxSweeper.NewHandler(platformModule.TenantRegistry, tenantLister, 0)

	eventHandlers := append(append([]platformEvents.IntegrationEventHandler{}, invoiceEvents...), inboxEvents...)
	jobHandlers := append(append(append([]platformJobs.JobHandler{}, invoiceJobs...), inboxJobs...), sweeper)
	verifier := attestation.NewVerifier(cfg.MessagingAttestationSecret)

	return Handlers{
		Events: platformEvents.NewRouter(verifier, eventHandlers...),
		Jobs:   platformJobs.NewRouter(verifier, jobHandlers...),
	}
}

func WireScheduler(deps *platform.Dependencies) (*scheduler.Engine, func(), error) {
	transport, closeTransport, err := NewBrokerTransport(deps)
	if err != nil {
		return nil, func() {}, err
	}
	rules := append(scheduler.PlatformRules(), inboxModule.RegisterSchedules(deps.Config)...)
	engine, err := scheduler.NewEngine(transport, rules)
	if err != nil {
		closeTransport()
		return nil, func() {}, err
	}
	return engine, closeTransport, nil
}

func NewBrokerTransport(deps *platform.Dependencies) (broker.Transport, func(), error) {
	cfg := deps.Config
	jobKeys := WireMessagingHandlers(deps).Jobs.JobTypes()

	switch cfg.DeploymentTarget {
	case config.DeploymentTargetAWS:
		return awsbroker.NewTransport(
			awsConfig.NewEventBridgeClient(deps.AWSConfig, cfg.AWSEndpointURL),
			awsConfig.NewSQSClient(deps.AWSConfig, cfg.AWSEndpointURL),
			cfg.EventBusName,
			cfg.SQSQueueURL,
			cfg.MessagingAttestationSecret,
		), func() {}, nil
	default:
		conn := rabbitmqbroker.NewConnection(cfg.RabbitMQURL)
		transport, err := rabbitmqbroker.NewTransport(conn, cfg.MessagingAttestationSecret, jobKeys...)
		if err != nil {
			_ = conn.Close()
			return nil, func() {}, err
		}
		return transport, func() { _ = conn.Close() }, nil
	}
}
