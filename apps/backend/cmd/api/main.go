package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/money-path/bowerbird/apps/backend/internal/health/application"
	healthinfra "github.com/money-path/bowerbird/apps/backend/internal/health/infrastructure"
	healthhttp "github.com/money-path/bowerbird/apps/backend/internal/health/presentation/http"
	orgapplication "github.com/money-path/bowerbird/apps/backend/internal/organization/application"
	orginfra "github.com/money-path/bowerbird/apps/backend/internal/organization/infrastructure"
	orghttp "github.com/money-path/bowerbird/apps/backend/internal/organization/presentation/http"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/awsconfig"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/config"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/database"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/events"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/tenant"
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

	// Setup Health Context
	healthRepo := healthinfra.NewPostgresRepository(pool)
	healthUseCase := application.NewCheckHealthUseCase(healthRepo)
	healthHandler := healthhttp.NewHandler(healthUseCase)

	mux := http.NewServeMux()
	healthHandler.Register(mux)

	// Setup Organization Context
	// Provide the root directory for migrations relative to the running binary (or use an env var)
	orgRepo := orginfra.NewPostgresRepository(pool)
	orgProvisioner := orginfra.NewPostgresProvisioner(pool, cfg.DatabaseURL, "apps/backend/migrations/tenant")
	orgUseCase := orgapplication.NewCreateOrganizationUseCase(orgRepo, orgProvisioner)
	orgHandler := orghttp.NewHandler(orgUseCase)

	// Register Routes
	orgHandler.Register(mux)

	// Setup Event Poller
	eventHandler := events.NewEventHandler()

	if cfg.EnableLocalEventLoop && cfg.AWSEndpointURL != "" {
		awsCfg, err := awsconfig.Load(ctxApp, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
		if err != nil {
			log.Fatalf("load aws config: %v", err)
		}

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
