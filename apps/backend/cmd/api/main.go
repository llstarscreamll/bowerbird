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
	connectionsapp "github.com/money-path/bowerbird/apps/backend/internal/connections/application"
	connectionsinfra "github.com/money-path/bowerbird/apps/backend/internal/connections/infrastructure"
	connectionshttp "github.com/money-path/bowerbird/apps/backend/internal/connections/presentation/http"
	"github.com/money-path/bowerbird/apps/backend/internal/health/application"
	healthinfra "github.com/money-path/bowerbird/apps/backend/internal/health/infrastructure"
	healthhttp "github.com/money-path/bowerbird/apps/backend/internal/health/presentation/http"
	identityapp "github.com/money-path/bowerbird/apps/backend/internal/identity/application"
	identityinfra "github.com/money-path/bowerbird/apps/backend/internal/identity/infrastructure"
	identityhttp "github.com/money-path/bowerbird/apps/backend/internal/identity/presentation/http"
	inboxapp "github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	inboxinfra "github.com/money-path/bowerbird/apps/backend/internal/inbox/infrastructure"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/infrastructure/provider"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/infrastructure/provider/gmail"
	inboxevents "github.com/money-path/bowerbird/apps/backend/internal/inbox/presentation/events"
	inboxhttp "github.com/money-path/bowerbird/apps/backend/internal/inbox/presentation/http"
	invoicingapp "github.com/money-path/bowerbird/apps/backend/internal/invoicing/application"
	invoicinginfra "github.com/money-path/bowerbird/apps/backend/internal/invoicing/infrastructure/router"
	invoicingevents "github.com/money-path/bowerbird/apps/backend/internal/invoicing/presentation/events"
	orgapplication "github.com/money-path/bowerbird/apps/backend/internal/organization/application"
	orginfra "github.com/money-path/bowerbird/apps/backend/internal/organization/infrastructure"
	orghttp "github.com/money-path/bowerbird/apps/backend/internal/organization/presentation/http"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/auth"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/awsconfig"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/config"
	platformcrypto "github.com/money-path/bowerbird/apps/backend/internal/platform/crypto"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/events"
	platforms3 "github.com/money-path/bowerbird/apps/backend/internal/platform/storage/s3"
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
	baseDBURL := strings.Replace(cfg.DatabaseURL, "/bowerbird?", "/%s?", 1)
	if baseDBURL == cfg.DatabaseURL {
		baseDBURL = strings.Replace(cfg.DatabaseURL, "/bowerbird", "/%s", 1)
	}
	registry := database.NewRegistry(pool, baseDBURL)
	defer registry.CloseAll()

	// Setup Health Context
	healthRepo := healthinfra.NewPostgresRepository(pool)
	healthUseCase := application.NewCheckHealthUseCase(healthRepo)
	healthHandler := healthhttp.NewHandler(healthUseCase)

	isDev := cfg.AppEnv == "development" || cfg.AppEnv == "local"
	mux := http.NewServeMux()
	healthHandler.Register(mux, isDev)

	// Setup Auth & Identity
	tokenGen := auth.NewTokenGenerator(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	authMiddleware := auth.Middleware(tokenGen)

	identityRepo := identityinfra.NewPostgresRepository(pool, registry)
	authService := identityapp.NewAuthService(identityRepo, tokenGen, cfg.AppEnv)
	identityService := identityapp.NewIdentityService(identityRepo)

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

	authHandler := identityhttp.NewAuthHandler(authService, identityService, googleConfig, microsoftConfig, strings.TrimRight(cfg.FrontendURL, "/"))
	authHandler.Register(mux, authMiddleware, isDev)

	// Setup Organization Context
	// Provide the root directory for migrations relative to the running binary (or use an env var)
	orgRepo := orginfra.NewPostgresRepository(pool)

	migrationsDir := os.Getenv("TENANT_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations/tenant"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "apps/backend/migrations/tenant"
		}
	}
	orgProvisioner := orginfra.NewPostgresProvisioner(pool, cfg.DatabaseURL, migrationsDir)
	orgCreateUseCase := orgapplication.NewCreateOrganizationUseCase(orgRepo, orgProvisioner)
	orgGetUseCase := orgapplication.NewGetOrganizationUseCase(orgRepo)
	orgHandler := orghttp.NewHandler(orgCreateUseCase, orgGetUseCase)

	// Register Routes
	orgHandler.Register(mux, authMiddleware, isDev)

	// Setup AWS Config
	var awsCfg aws.Config
	if cfg.AWSEndpointURL != "" {
		awsCfg, err = awsconfig.Load(ctxApp, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
		if err != nil {
			log.Fatalf("load aws config: %v", err)
		}
	} else if cfg.AWSRegion != "" {
		awsCfg, err = awsconfig.Load(ctxApp, cfg.AWSRegion, "", "", "")
		if err != nil {
			log.Fatalf("load aws config: %v", err)
		}
	}

	var eventPublisher connectionshttp.EventPublisher
	var businessEventPublisher events.BusinessEventPublisher
	if cfg.EventBusName != "" {
		ebClient := awsconfig.NewEventBridgeClient(awsCfg, cfg.AWSEndpointURL)
		ebPublisher := events.NewEventBridgePublisher(ebClient, cfg.EventBusName)
		eventPublisher = ebPublisher
		businessEventPublisher = ebPublisher
	}

	// Setup Connections Context
	var connectionsService connectionsapp.InternalService
	var connectionsHandler *connectionshttp.Handler
	if cfg.InboxCredentialsEncryptionKey != "" {
		cipher, err := platformcrypto.NewAESCipherFromBase64Key(cfg.InboxCredentialsEncryptionKey)
		if err != nil {
			log.Fatalf("new cipher failed: %v", err)
		}
		credentialsService := connectionsapp.NewCredentialsService(cipher)
		connectionsRepo := connectionsinfra.NewPostgresRepository(registry)
		connectionsService = connectionsapp.NewInternalService(connectionsRepo, credentialsService)

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
		connectionsHandler = connectionshttp.NewHandler(connectionsRepo, credentialsService, connectionsGoogleConfig, tokenGen, cipher, eventPublisher, strings.TrimRight(cfg.FrontendURL, "/"))
	} else {
		// Just for fallback if not provided, though it's typically required
		connectionsRepo := connectionsinfra.NewPostgresRepository(registry)
		connectionsService = connectionsapp.NewInternalService(connectionsRepo, nil)
		connectionsHandler = connectionshttp.NewHandler(connectionsRepo, nil, nil, tokenGen, nil, eventPublisher, strings.TrimRight(cfg.FrontendURL, "/")) // In a real scenario this nil might cause panic if hit, but this block is a fallback
	}
	connectionsHandler.Register(mux, authMiddleware, isDev)

	// Setup Inbox Context
	inboxRepo := inboxinfra.NewPostgresRepository(registry)

	// Create Provider Factory for sync accounts
	var providerFactory inboxapp.ProviderClientFactory
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		providerFactory = provider.NewDefaultFactory(gmail.OAuthConfig{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		})
	}

	// The sync process needs these
	var syncAccountCommand *inboxapp.SyncAccountCommand
	var syncConnectionsCommand *inboxapp.SyncAllAccountsCommand
	var syncAccountJobDispatcher inboxapp.SyncAccountJobDispatcher
	if providerFactory != nil {
		if businessEventPublisher == nil {
			log.Fatal("event bus publisher is required for inbox sync")
		}
		if cfg.S3BucketName == "" {
			log.Fatal("s3 bucket name is required for inbox sync")
		}

		attachmentObjectStore := platforms3.NewObjectStore(awsconfig.NewS3Client(awsCfg, cfg.AWSEndpointURL), cfg.S3BucketName)

		syncAccountCommand = inboxapp.NewSyncAccountCommand(
			inboxRepo,
			inboxRepo,
			connectionsService,
			providerFactory,
			businessEventPublisher,
			attachmentObjectStore,
		)
		syncAccountJobDispatcher = inboxapp.NewInlineSyncAccountJobDispatcher(syncAccountCommand)
		syncConnectionsCommand = inboxapp.NewSyncAllAccountsCommand(connectionsService, syncAccountJobDispatcher)
	}

	listAccountHealthUseCase := inboxapp.NewListAccountHealthUseCase(inboxRepo, connectionsService)
	listMessagesUseCase := inboxapp.NewListMessagesUseCase(inboxRepo)
	getMessageUseCase := inboxapp.NewGetMessageUseCase(inboxRepo)
	inboxHandler := inboxhttp.NewHandler(listAccountHealthUseCase, listMessagesUseCase, getMessageUseCase, syncConnectionsCommand)
	inboxHandler.Register(mux, authMiddleware, isDev)

	// Setup Event Poller
	invoicingRouter := invoicinginfra.NewLogRouter()
	invoicingUseCase := invoicingapp.NewProcessInboxEventUseCase(invoicingRouter)
	inboxMessageSubscriber := invoicingevents.NewInboxMessageReceivedSubscriber(invoicingUseCase)

	inboxEventsSubscriber := inboxevents.NewConnectionAddedSubscriber(syncAccountCommand)
	eventHandler := events.NewEventHandler(inboxMessageSubscriber, inboxEventsSubscriber)

	if cfg.EnableLocalEventLoop && cfg.AWSEndpointURL != "" {
		sqsClient := awsconfig.NewSQSClient(awsCfg, cfg.AWSEndpointURL)
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
