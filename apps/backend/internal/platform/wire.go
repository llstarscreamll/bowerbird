package platform

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/bowerbird/internal/platform/awsconfig"
	"github.com/bowerbird/internal/platform/config"
	"github.com/bowerbird/internal/platform/database"
	"github.com/bowerbird/internal/platform/events"
	"github.com/bowerbird/internal/platform/jobs"
	platformStorage "github.com/bowerbird/internal/platform/storage"
	platformS3 "github.com/bowerbird/internal/platform/storage/s3"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Config         config.Config
	ControlDB      *pgxpool.Pool
	AWSConfig      aws.Config
	TenantRegistry *database.Registry
	FileStore      platformStorage.FileStore
	EventBus       events.EventBus
	JobQueue       jobs.Queue
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

	awsCfg, err := awsConfig.Load(ctx, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	eventBus := events.NewEventBridgePublisher(
		awsConfig.NewEventBridgeClient(awsCfg, cfg.AWSEndpointURL),
		cfg.EventBusName,
	)
	jobQueue := jobs.NewSQSQueue(
		awsConfig.NewSQSClient(awsCfg, cfg.AWSEndpointURL),
		cfg.SQSQueueURL,
	)
	fileStore := platformS3.NewObjectStore(awsConfig.NewS3Client(awsCfg, cfg.AWSEndpointURL), cfg.S3BucketName)

	return &Dependencies{
		Config:         cfg,
		ControlDB:      controlDB,
		AWSConfig:      awsCfg,
		TenantRegistry: tenantRegistry,
		FileStore:      fileStore,
		EventBus:       eventBus,
		JobQueue:       jobQueue,
	}, nil
}

func buildBaseTenantDBUrl(databaseURL string) string {
	baseDbURL := strings.Replace(databaseURL, "/bowerbird?", "/%s?", 1)
	if baseDbURL == databaseURL {
		baseDbURL = strings.Replace(databaseURL, "/bowerbird", "/%s", 1)
	}

	return baseDbURL
}
