package platform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/bowerbird/internal/platform/awsconfig"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	outboxPublisher "github.com/bowerbird/internal/platform/outbox/publisher"
	"github.com/bowerbird/internal/platform/outbox/relay"
	outboxStore "github.com/bowerbird/internal/platform/outbox/store"
	"github.com/bowerbird/internal/platform/scheduler"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	platformS3 "github.com/bowerbird/internal/platform/storage/s3"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Config         config.Config
	ControlDB      *pgxpool.Pool
	AWSConfig      aws.Config
	TenantRegistry *database.Registry
	FileStore      platformStorage.FileStore
	EventBus       events.EventBus
	TaskQueue      jobs.TaskQueue
	OutboxAppender outboxStore.Appender
	Scheduler      scheduler.Scheduler
}

func NewModule(ctx context.Context) (*Dependencies, error) {
	cfg, err := config.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	controlDB, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	tenantRegistry := database.NewRegistry(controlDB, buildBaseTenantDBUrl(cfg.DatabaseURL))
	outbox := outboxStore.NewRegistryStore(tenantRegistry)

	deps := &Dependencies{
		Config:         cfg,
		ControlDB:      controlDB,
		TenantRegistry: tenantRegistry,
		OutboxAppender: outbox,
		EventBus:       outboxPublisher.NewOutboxEventPublisher(outbox),
		TaskQueue:      outboxPublisher.NewOutboxTaskQueue(outbox),
	}
	deps.Scheduler = scheduler.NewOutboxScheduler(deps.TaskQueue, relay.NewControlPlaneTenantLister(controlDB), time.Hour)

	switch cfg.DeploymentTarget {
	case config.DeploymentTargetAWS:
		if err := wireAWS(ctx, cfg, deps); err != nil {
			return nil, err
		}
	case config.DeploymentTargetOnPrem:
		if err := wireOnPrem(ctx, cfg, deps); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported deployment target: %q", cfg.DeploymentTarget)
	}

	return deps, nil
}

func wireOnPrem(ctx context.Context, cfg config.Config, deps *Dependencies) error {
	endpoint := cfg.MinIOEndpointURL
	if endpoint == "" {
		endpoint = cfg.AWSEndpointURL
	}
	awsCfg, err := awsConfig.Load(ctx, cfg.AWSRegion, endpoint, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
	if err != nil {
		return fmt.Errorf("load object storage config: %w", err)
	}
	deps.AWSConfig = awsCfg

	s3Client := awsConfig.NewS3Client(awsCfg, endpoint)
	presignEndpoint := cfg.S3PresignEndpointURL
	if presignEndpoint == "" {
		presignEndpoint = endpoint
	}
	presignClient := awsConfig.NewS3PresignClient(awsCfg, presignEndpoint)
	deps.FileStore = platformS3.NewObjectStoreWithClients(s3Client, presignClient, cfg.S3BucketName)
	return nil
}

func wireAWS(ctx context.Context, cfg config.Config, deps *Dependencies) error {
	awsCfg, err := awsConfig.Load(ctx, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}
	deps.AWSConfig = awsCfg
	deps.FileStore = platformS3.NewObjectStore(awsConfig.NewS3Client(awsCfg, cfg.AWSEndpointURL), cfg.S3BucketName)
	return nil
}

func buildBaseTenantDBUrl(databaseURL string) string {
	baseDbURL := strings.Replace(databaseURL, "/bowerbird?", "/%s?", 1)
	if baseDbURL == databaseURL {
		baseDbURL = strings.Replace(databaseURL, "/bowerbird", "/%s", 1)
	}
	return baseDbURL
}

// RelayRepository returns the relay-side outbox port for a tenant database pool.
func RelayRepository(ctx context.Context, registry *database.Registry, tenantSlug string) (outboxStore.RelayRepository, error) {
	tenantCtx := tenant.WithTenantID(ctx, tenantSlug)
	pool, err := registry.GetPool(tenantCtx)
	if err != nil {
		return nil, err
	}
	return outboxStore.NewPostgresStore(pool), nil
}

// RelayStore is deprecated; use RelayRepository.
func RelayStore(ctx context.Context, registry *database.Registry, tenantSlug string) (outboxStore.RelayRepository, error) {
	return RelayRepository(ctx, registry, tenantSlug)
}
