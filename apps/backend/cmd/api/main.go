package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	catalogModule "github.com/bowerbird/internal/catalog"
	connectionsModule "github.com/bowerbird/internal/connections"
	connectionsApp "github.com/bowerbird/internal/connections/application"
	entitlementsModule "github.com/bowerbird/internal/entitlements"
	filesModule "github.com/bowerbird/internal/files"
	"github.com/bowerbird/internal/health"
	identityModule "github.com/bowerbird/internal/identity"
	inboxModule "github.com/bowerbird/internal/inbox"
	invoicesModule "github.com/bowerbird/internal/invoices"
	invoicesEvents "github.com/bowerbird/internal/invoices/adapters/events"
	invoicesJobs "github.com/bowerbird/internal/invoices/adapters/jobs"
	invoiceLinking "github.com/bowerbird/internal/invoices/adapters/linking"
	partiesModule "github.com/bowerbird/internal/parties"
	"github.com/bowerbird/internal/platform"
	"github.com/bowerbird/internal/platform/auth"
	awsConfig "github.com/bowerbird/internal/platform/awsconfig"
	platformCrypto "github.com/bowerbird/internal/platform/crypto"
	"github.com/bowerbird/internal/platform/events"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
	rbacModule "github.com/bowerbird/internal/rbac"
	secretsModule "github.com/bowerbird/internal/secrets"
	tenantModule "github.com/bowerbird/internal/tenant"
)

func main() {
	ctxApp, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	platformModule, err := platform.NewModule(ctxApp)
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}
	cfg := platformModule.Config
	pool := platformModule.ControlDB
	defer pool.Close()

	tenantsDbRegistry := platformModule.TenantRegistry
	defer tenantsDbRegistry.CloseAll()

	mux := http.NewServeMux()

	// Setup Health Context
	healthApp := health.NewApplication(pool)
	health.NewHTTPHandler(mux, healthApp, cfg)

	// Setup Auth & Identity
	tokenGen := auth.NewTokenGenerator(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	authMiddleware := auth.Middleware(tokenGen)

	identityApp := identityModule.NewApplication(cfg, pool, tenantsDbRegistry, tokenGen)
	identityModule.NewHTTPHandler(mux, identityApp, pool, tenantsDbRegistry, authMiddleware, cfg)

	entitlementsApp := entitlementsModule.NewApplication(pool)
	entitlementsModule.NewHTTPHandler(mux, entitlementsApp, identityApp, pool, authMiddleware, cfg)

	// Setup Organization Context
	// Provide the root directory for migrations relative to the running binary (or use an env var)
	migrationsDir := os.Getenv("TENANT_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations/tenant"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "apps/backend/migrations/tenant"
		}
	}
	organizationApp := tenantModule.NewApplication(pool, cfg.DatabaseURL, migrationsDir, entitlementsApp)
	tenantModule.NewHTTPHandler(mux, organizationApp, authMiddleware, cfg)

	// Setup AWS Config
	awsCfg := platformModule.AWSConfig

	if cfg.S3BucketName != "" {
		filesApp := filesModule.NewApplication(platformModule.FileStore)
		filesModule.NewHTTPHandler(mux, filesApp, authMiddleware, cfg)
	} else {
		log.Printf("file upload routes disabled: s3_bucket_name is empty")
	}

	var connectionsEventBus events.EventBus
	if cfg.EventBusName != "" {
		connectionsEventBus = platformModule.EventBus
	}

	// Setup Connections Context
	var connectionsService connectionsApp.InternalService
	if cfg.InboxCredentialsEncryptionKey != "" {
		cipher, err := platformCrypto.NewAESCipherFromBase64Key(cfg.InboxCredentialsEncryptionKey)
		if err != nil {
			log.Fatalf("new cipher failed: %v", err)
		}
		connectionsApp := connectionsModule.NewApplication(tenantsDbRegistry, cipher)
		connectionsService = connectionsModule.NewInternalService(connectionsApp)
		connectionsModule.NewHTTPHandler(mux, cfg, tenantsDbRegistry, cipher, tokenGen, cipher, connectionsEventBus, authMiddleware, entitlementsApp)
	} else {
		connectionsApp := connectionsModule.NewApplication(tenantsDbRegistry, nil)
		connectionsService = connectionsModule.NewInternalService(connectionsApp)
		connectionsModule.NewHTTPHandler(mux, cfg, tenantsDbRegistry, nil, tokenGen, nil, connectionsEventBus, authMiddleware, entitlementsApp)
	}

	// Setup Inbox Context
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" && cfg.S3BucketName == "" {
		log.Fatal("s3 bucket name is required for inbox sync")
	}

	inboxApp := inboxModule.NewApplication(
		cfg,
		connectionsService,
		platformModule.EventBus,
		platformModule.FileStore,
		tenantsDbRegistry,
		platformModule.JobQueue,
	)
	inboxModule.NewHTTPHandler(mux, inboxApp, authMiddleware, cfg, entitlementsApp)

	rbacService := rbacModule.NewService(tenantsDbRegistry)
	rbacModule.NewHTTPHandler(mux, rbacService, authMiddleware, cfg)

	secretsCipher, err := platformCrypto.NewAESCipherFromBase64Key(cfg.TenantSecretsEncryptionKey)
	if err != nil {
		log.Fatalf("new tenant secrets cipher failed: %v", err)
	}
	secretsApp := secretsModule.NewApplication(tenantsDbRegistry, secretsCipher)
	secretsModule.NewHTTPHandler(mux, secretsApp, rbacService, authMiddleware, cfg)
	documentPasswordResolver := invoicesModule.NewSecretsPasswordAdapter(secretsModule.NewDocumentPasswordResolver(secretsApp))

	partiesApp := partiesModule.NewApplication(tenantsDbRegistry)
	partiesModule.NewHTTPHandler(mux, partiesApp, authMiddleware, cfg)
	catalogApp := catalogModule.NewApplication(tenantsDbRegistry)
	catalogModule.NewHTTPHandler(mux, catalogApp, authMiddleware, cfg)

	invoicingApp := invoicesModule.NewApplication(
		cfg,
		platformModule.EventBus,
		platformModule.JobQueue,
		platformModule.FileStore,
		tenantsDbRegistry,
		documentPasswordResolver,
		invoiceLinking.NewPartyResolverAdapter(partiesModule.NewIssuerPartyLookup(partiesApp)),
		invoiceLinking.NewCatalogResolverAdapter(catalogApp),
		invoiceLinking.NewCatalogNamesAdapter(catalogApp),
		invoiceLinking.NewCatalogMatchingAdapter(catalogApp),
	)
	invoicesModule.NewHTTPHandler(mux, invoicingApp, authMiddleware, cfg)

	inboxMessageSubscriber := invoicesEvents.NewInboxMessageReceivedSubscriber(invoicingApp.Commands.CreateInvoicesFromInboxMessage)
	invoiceExtractionProcessor := invoicesJobs.NewInvoiceExtractionRequestedProcessor(invoicingApp.Commands.ProcessInvoiceExtractionJob)

	inboxEventsSubscriber := inboxModule.NewConnectionAddedSubscriber(inboxApp, entitlementsApp)
	inboxSyncProcessor := inboxModule.NewSyncAccountProcessor(inboxApp, entitlementsApp)
	eventHandler := events.NewEventHandler(inboxMessageSubscriber, inboxEventsSubscriber)
	jobHandler := platformJobs.NewHandler(invoiceExtractionProcessor, inboxSyncProcessor)

	if cfg.EnableLocalEventLoop && cfg.AWSEndpointURL != "" {
		sqsClient := awsConfig.NewSQSClient(awsCfg, cfg.AWSEndpointURL)
		jobsPoller := platformJobs.NewPoller(sqsClient, jobHandler, cfg.SQSQueueURL)
		eventsPoller := events.NewPoller(sqsClient, eventHandler, cfg.EventBridgeQueueURL)
		jobsPoller.Run(ctxApp)
		eventsPoller.Run(ctxApp)
		log.Printf("local event loop enabled: sqs=%t eventbridge=%t", cfg.SQSQueueURL != "", cfg.EventBridgeQueueURL != "")
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      withSecurityHeaders(withCORS(tenant.Middleware(mux), cfg.AllowedOrigins)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("http api listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	cancelApp()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler, allowedOriginsCSV string) http.Handler {
	allowed := parseOrigins(allowedOriginsCSV)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if match, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", match)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Tenant-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseOrigins(csv string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(csv, ",") {
		origin := strings.TrimSpace(part)
		if origin == "" || origin == "*" {
			continue
		}
		out[origin] = origin
	}
	return out
}
