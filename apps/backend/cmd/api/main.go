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

	"github.com/aws/aws-sdk-go-v2/aws"
	connectionsApp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	connectionsInfra "github.com/money-path/bowerbird/apps/backend/internal/connections/infrastructure"
	connectionsHttp "github.com/money-path/bowerbird/apps/backend/internal/connections/presentation/http"
	filesApp "github.com/money-path/bowerbird/apps/backend/internal/files/application"
	filesHttp "github.com/money-path/bowerbird/apps/backend/internal/files/presentation/http"
	"github.com/money-path/bowerbird/apps/backend/internal/health/application"
	healthInfra "github.com/money-path/bowerbird/apps/backend/internal/health/infrastructure"
	healthHttp "github.com/money-path/bowerbird/apps/backend/internal/health/presentation/http"
	identityApp "github.com/money-path/bowerbird/apps/backend/internal/identity/application"
	identityInfra "github.com/money-path/bowerbird/apps/backend/internal/identity/infrastructure"
	identityHttp "github.com/money-path/bowerbird/apps/backend/internal/identity/presentation/http"
	inboxApp "github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	inboxInfra "github.com/money-path/bowerbird/apps/backend/internal/inbox/infrastructure"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/infrastructure/provider"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/infrastructure/provider/gmail"
	inboxEvents "github.com/money-path/bowerbird/apps/backend/internal/inbox/presentation/events"
	inboxHttp "github.com/money-path/bowerbird/apps/backend/internal/inbox/presentation/http"
	invoicingApp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	invoicingLLM "github.com/money-path/bowerbird/apps/backend/internal/invoicing/infrastructure/llm"
	invoicingRepo "github.com/money-path/bowerbird/apps/backend/internal/invoicing/infrastructure/repository/postgres"
	invoicingXML "github.com/money-path/bowerbird/apps/backend/internal/invoicing/infrastructure/xml"
	invoicingEvents "github.com/money-path/bowerbird/apps/backend/internal/invoicing/presentation/events"
	orgApplication "github.com/money-path/bowerbird/apps/backend/internal/organization/application"
	orgInfra "github.com/money-path/bowerbird/apps/backend/internal/organization/infrastructure"
	orgHttp "github.com/money-path/bowerbird/apps/backend/internal/organization/presentation/http"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	awsConfig "github.com/money-path/bowerbird/apps/backend/internal/platform/awsconfig"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/config"
	platformCrypto "github.com/money-path/bowerbird/apps/backend/internal/platform/crypto"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/events"
	platformStorage "github.com/money-path/bowerbird/apps/backend/internal/platform/storage"
	platformS3 "github.com/money-path/bowerbird/apps/backend/internal/platform/storage/s3"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

func main() {
	ctxApp, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	cfg, err := config.Load(ctxApp)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := database.Connect(ctxApp, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	// Parse base DB URL for tenant databases (e.g. replacing 'bowerbird' with '%s')
	// Simplified assumption: the last path segment before ? is the db name.
	baseDbURL := strings.Replace(cfg.DatabaseURL, "/bowerbird?", "/%s?", 1)
	if baseDbURL == cfg.DatabaseURL {
		baseDbURL = strings.Replace(cfg.DatabaseURL, "/bowerbird", "/%s", 1)
	}
	registry := database.NewRegistry(pool, baseDbURL)
	defer registry.CloseAll()

	// Setup Health Context
	healthRepo := healthInfra.NewPostgresRepository(pool)
	healthUseCase := application.NewCheckHealthUseCase(healthRepo)
	healthHandler := healthHttp.NewHandler(healthUseCase)

	isDev := cfg.AppEnv == "development" || cfg.AppEnv == "local"
	mux := http.NewServeMux()
	healthHandler.Register(mux, isDev)

	// Setup Auth & Identity
	tokenGen := auth.NewTokenGenerator(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	authMiddleware := auth.Middleware(tokenGen)

	identityRepo := identityInfra.NewPostgresRepository(pool, registry)
	authService := identityApp.NewAuthService(identityRepo, tokenGen, cfg.AppEnv)
	identityService := identityApp.NewIdentityService(identityRepo)

	var googleConfig *oauth2.Config
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		googleConfig = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/auth/google/callback",
			Scopes:       []string{"email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}

	var microsoftConfig *oauth2.Config
	if cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != "" {
		microsoftConfig = &oauth2.Config{
			ClientID:     cfg.MicrosoftClientID,
			ClientSecret: cfg.MicrosoftClientSecret,
			RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/auth/microsoft/callback",
			Scopes:       []string{"User.Read"},
			Endpoint:     microsoft.AzureADEndpoint("common"),
		}
	}

	authHandler := identityHttp.NewAuthHandler(authService, identityService, googleConfig, microsoftConfig, strings.TrimRight(cfg.FrontendURL, "/"))
	authHandler.Register(mux, authMiddleware, isDev)

	// Setup Organization Context
	// Provide the root directory for migrations relative to the running binary (or use an env var)
	orgRepo := orgInfra.NewPostgresRepository(pool)

	migrationsDir := os.Getenv("TENANT_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations/tenant"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "apps/backend/migrations/tenant"
		}
	}
	orgProvisioner := orgInfra.NewPostgresProvisioner(pool, cfg.DatabaseURL, migrationsDir)
	orgCreateUseCase := orgApplication.NewCreateOrganizationUseCase(orgRepo, orgProvisioner)
	orgGetUseCase := orgApplication.NewGetOrganizationUseCase(orgRepo)
	orgHandler := orgHttp.NewHandler(orgCreateUseCase, orgGetUseCase)

	// Register Routes
	orgHandler.Register(mux, authMiddleware, isDev)

	// Setup AWS Config
	var awsCfg aws.Config
	if cfg.AWSEndpointURL != "" {
		awsCfg, err = awsConfig.Load(ctxApp, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
		if err != nil {
			log.Fatalf("load aws config: %v", err)
		}
	} else if cfg.AWSRegion != "" {
		awsCfg, err = awsConfig.Load(ctxApp, cfg.AWSRegion, "", "", "")
		if err != nil {
			log.Fatalf("load aws config: %v", err)
		}
	}

	if cfg.S3BucketName != "" {
		s3Client := awsConfig.NewS3Client(awsCfg, cfg.AWSEndpointURL)
		presignEndpointURL := cfg.S3PresignEndpointURL
		if presignEndpointURL == "" {
			presignEndpointURL = cfg.AWSEndpointURL
		}
		presignClient := awsConfig.NewS3PresignClient(awsCfg, presignEndpointURL)

		fileStore := platformS3.NewObjectStoreWithClients(s3Client, presignClient, cfg.S3BucketName)
		requestUploadURLUseCase := filesApp.NewRequestUploadURLCommand(fileStore)
		requestDownloadURLUseCase := filesApp.NewRequestDownloadURLCommand(fileStore)
		filesHandler := filesHttp.NewHandler(requestUploadURLUseCase, requestDownloadURLUseCase)
		filesHandler.Register(mux, authMiddleware, isDev)
	} else {
		log.Printf("file upload routes disabled: s3_bucket_name is empty")
	}

	var eventPublisher connectionsHttp.EventPublisher
	var businessEventPublisher events.BusinessEventPublisher
	if cfg.EventBusName != "" {
		ebClient := awsConfig.NewEventBridgeClient(awsCfg, cfg.AWSEndpointURL)
		ebPublisher := events.NewEventBridgePublisher(ebClient, cfg.EventBusName)
		eventPublisher = ebPublisher
		businessEventPublisher = ebPublisher
	}

	// Setup Connections Context
	var connectionsService connectionsApp.InternalService
	var connectionsHandler *connectionsHttp.Handler
	if cfg.InboxCredentialsEncryptionKey != "" {
		cipher, err := platformCrypto.NewAESCipherFromBase64Key(cfg.InboxCredentialsEncryptionKey)
		if err != nil {
			log.Fatalf("new cipher failed: %v", err)
		}
		credentialsService := connectionsApp.NewCredentialsService(cipher)
		connectionsRepo := connectionsInfra.NewPostgresRepository(registry)
		connectionsService = connectionsApp.NewInternalService(connectionsRepo, credentialsService)

		var connectionsGoogleConfig *oauth2.Config
		if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
			connectionsGoogleConfig = &oauth2.Config{
				ClientID:     cfg.GoogleClientID,
				ClientSecret: cfg.GoogleClientSecret,
				RedirectURL:  strings.TrimRight(cfg.BackendURL, "/") + "/api/v1/connections/google/callback",
				Scopes:       []string{"email", "https://www.googleapis.com/auth/gmail.modify"},
				Endpoint:     google.Endpoint,
			}
		}
		connectionsHandler = connectionsHttp.NewHandler(connectionsRepo, credentialsService, connectionsGoogleConfig, tokenGen, cipher, eventPublisher, strings.TrimRight(cfg.FrontendURL, "/"))
	} else {
		// Just for fallback if not provided, though it's typically required
		connectionsRepo := connectionsInfra.NewPostgresRepository(registry)
		connectionsService = connectionsApp.NewInternalService(connectionsRepo, nil)
		connectionsHandler = connectionsHttp.NewHandler(connectionsRepo, nil, nil, tokenGen, nil, eventPublisher, strings.TrimRight(cfg.FrontendURL, "/")) // In a real scenario this nil might cause panic if hit, but this block is a fallback
	}
	connectionsHandler.Register(mux, authMiddleware, isDev)

	// Setup Inbox Context
	inboxRepo := inboxInfra.NewPostgresRepository(registry)

	// Create Provider Factory for sync accounts
	var providerFactory inboxApp.ProviderClientFactory
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		providerFactory = provider.NewDefaultFactory(gmail.OAuthConfig{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		})
	}

	// The sync process needs these
	var syncAccountCommand *inboxApp.SyncAccountCommand
	var syncConnectionsCommand *inboxApp.SyncAllAccountsCommand
	var syncAccountJobDispatcher inboxApp.SyncAccountJobDispatcher
	if providerFactory != nil {
		if businessEventPublisher == nil {
			log.Fatal("event bus publisher is required for inbox sync")
		}
		if cfg.S3BucketName == "" {
			log.Fatal("s3 bucket name is required for inbox sync")
		}

		attachmentObjectStore := platformS3.NewObjectStore(awsConfig.NewS3Client(awsCfg, cfg.AWSEndpointURL), cfg.S3BucketName)

		syncAccountCommand = inboxApp.NewSyncAccountCommand(
			inboxRepo,
			inboxRepo,
			connectionsService,
			providerFactory,
			businessEventPublisher,
			attachmentObjectStore,
		)
		syncAccountJobDispatcher = inboxApp.NewInlineSyncAccountJobDispatcher(syncAccountCommand)
		syncConnectionsCommand = inboxApp.NewSyncAllAccountsCommand(connectionsService, syncAccountJobDispatcher)
	}

	listAccountHealthUseCase := inboxApp.NewListAccountHealthUseCase(inboxRepo, connectionsService)
	listMessagesUseCase := inboxApp.NewListMessagesUseCase(inboxRepo)
	getMessageUseCase := inboxApp.NewGetMessageUseCase(inboxRepo)
	inboxHandler := inboxHttp.NewHandler(listAccountHealthUseCase, listMessagesUseCase, getMessageUseCase, syncConnectionsCommand)
	inboxHandler.Register(mux, authMiddleware, isDev)

	// Setup Event Poller
	invoicingRepoAdapter := invoicingRepo.NewRepository(registry)
	invoicingXMLExtractor := invoicingXML.NewDIANUBL21Parser()

	var invoicingLLMExtractor *invoicingLLM.GeminiExtractor
	if cfg.GeminiAPIKey != "" {
		invoicingLLMExtractor, err = invoicingLLM.NewGeminiExtractor(invoicingLLM.GeminiExtractorConfig{
			APIKey:   cfg.GeminiAPIKey,
			Model:    cfg.GeminiModel,
			Endpoint: cfg.GeminiEndpoint,
		})
		if err != nil {
			log.Fatalf("new gemini extractor failed: %v", err)
		}
	}

	var invoicingStore platformStorage.FileStore
	if cfg.S3BucketName != "" {
		invoicingStore = platformS3.NewObjectStore(awsConfig.NewS3Client(awsCfg, cfg.AWSEndpointURL), cfg.S3BucketName)
	}

	checkInboxForInvoicesCommand := invoicingApp.NewCheckInboxMessageForInvoiceCandidatesCommand(businessEventPublisher)
	var extractInvoiceCommand *invoicingApp.ExtractInvoiceCommand
	if invoicingStore != nil {
		extractInvoiceCommand = invoicingApp.NewExtractInvoiceCommand(invoicingStore, invoicingXMLExtractor, invoicingLLMExtractor, invoicingRepoAdapter)
	}
	inboxMessageSubscriber := invoicingEvents.NewInboxMessageReceivedSubscriber(checkInboxForInvoicesCommand)
	invoiceExtractionSubscriber := invoicingEvents.NewInvoiceExtractionRequestedSubscriber(extractInvoiceCommand)

	inboxEventsSubscriber := inboxEvents.NewConnectionAddedSubscriber(syncAccountCommand)
	eventHandler := events.NewEventHandler(inboxMessageSubscriber, invoiceExtractionSubscriber, inboxEventsSubscriber)

	if cfg.EnableLocalEventLoop && cfg.AWSEndpointURL != "" {
		sqsClient := awsConfig.NewSQSClient(awsCfg, cfg.AWSEndpointURL)
		poller := events.NewPoller(sqsClient, eventHandler, cfg.SQSQueueURL, cfg.EventBridgeQueueURL)
		poller.Run(ctxApp)
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
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler, allowedOrigins string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Tenant-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
