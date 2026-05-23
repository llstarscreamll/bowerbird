package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/awsconfig"
)

type Config struct {
	Port                 string
	AWSRegion            string
	AWSEndpointURL       string
	AWSAccessKeyID       string
	AWSSecretAccessKey   string
	SSMParameterName     string
	EnableLocalEventLoop bool
	AllowedOrigins       string

	// Loaded from SSM
	DatabaseURL         string `json:"database_url"`
	SQSQueueURL         string `json:"sqs_queue_url"`
	EventBridgeQueueURL string `json:"eventbridge_queue_url"`
	S3BucketName        string `json:"s3_bucket_name"`
	ThirdPartyAPIKey    string `json:"third_party_api_key"`
}

func Load(ctx context.Context) (Config, error) {
	// 1. Load base env vars
	cfg := Config{
		Port:                 getEnv("PORT", "8080"),
		AWSRegion:            getEnv("AWS_REGION", "us-east-1"),
		AWSEndpointURL:       os.Getenv("AWS_ENDPOINT_URL"),
		AWSAccessKeyID:       getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey:   getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		SSMParameterName:     getEnv("SSM_PARAMETER_NAME", "/bowerbird/local/secrets"),
		EnableLocalEventLoop: getEnv("ENABLE_LOCAL_EVENT_LOOP", "true") == "true",
		AllowedOrigins:       getEnv("ALLOWED_ORIGINS", "*"),
	}

	// 2. Load AWS Config to fetch SSM
	awsCfg, err := awsconfig.Load(ctx, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
	if err != nil {
		return cfg, fmt.Errorf("load aws config for ssm: %w", err)
	}

	// 3. Fetch and merge secrets from SSM
	if cfg.SSMParameterName != "" {
		if err := loadSSMSecrets(ctx, awsCfg, cfg.AWSEndpointURL, &cfg); err != nil {
			return cfg, fmt.Errorf("load ssm secrets: %w", err)
		}
	}

	// Fallback to env vars if not provided by SSM
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	if cfg.SQSQueueURL == "" {
		cfg.SQSQueueURL = os.Getenv("SQS_QUEUE_URL")
	}
	if cfg.EventBridgeQueueURL == "" {
		cfg.EventBridgeQueueURL = os.Getenv("EVENTBRIDGE_QUEUE_URL")
	}
	if cfg.S3BucketName == "" {
		cfg.S3BucketName = os.Getenv("S3_BUCKET_NAME")
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required (from SSM or env)")
	}

	return cfg, nil
}

func loadSSMSecrets(ctx context.Context, awsCfg aws.Config, endpointURL string, cfg *Config) error {
	var client *ssm.Client
	if endpointURL != "" {
		client = ssm.NewFromConfig(awsCfg, func(o *ssm.Options) {
			o.BaseEndpoint = &endpointURL
		})
	} else {
		client = ssm.NewFromConfig(awsCfg)
	}

	param, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &cfg.SSMParameterName,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return err
	}

	if param.Parameter == nil || param.Parameter.Value == nil {
		return fmt.Errorf("parameter %s is empty", cfg.SSMParameterName)
	}

	return json.Unmarshal([]byte(*param.Parameter.Value), cfg)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
