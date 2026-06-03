package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/money-path/bowerbird/apps/backend/internal/platform/awsconfig"
)

type Config struct {
	AppEnv                        string    `json:"app_env"`
	Port                          string    `json:"port"`
	DatabaseURL                   string    `json:"database_url"`
	SQSQueueURL                   string    `json:"sqs_queue_url"`
	EventBridgeQueueURL           string    `json:"eventbridge_queue_url"`
	EventBusName                  string    `json:"event_bus_name"`
	S3BucketName                  string    `json:"s3_bucket_name"`
	S3PresignEndpointURL          string    `json:"s3_presign_endpoint_url"`
	AWSRegion                     string    `json:"aws_region"`
	AWSEndpointURL                string    `json:"aws_endpoint_url"`
	AWSAccessKeyID                string    `json:"aws_access_key_id"`
	AWSSecretAccessKey            string    `json:"aws_secret_access_key"`
	SSMParameterName              string    `json:"ssm_parameter_name"`
	EnableLocalEventLoop          bool      `json:"enable_local_event_loop"`
	AllowedOrigins                string    `json:"allowed_origins"`
	GoogleClientID                string    `json:"google_client_id"`
	GoogleClientSecret            string    `json:"google_client_secret"`
	MicrosoftClientID             string    `json:"microsoft_client_id"`
	MicrosoftClientSecret         string    `json:"microsoft_client_secret"`
	GeminiAPIKey                  string    `json:"gemini_api_key"`
	GeminiModel                   string    `json:"gemini_model"`
	GeminiEndpoint                string    `json:"gemini_endpoint"`
	InboxCredentialsEncryptionKey string    `json:"inbox_credentials_encryption_key"`
	FrontendURL                   string    `json:"frontend_url"`
	BackendURL                    string    `json:"backend_url"`
	JWT                           JWTConfig `json:"-"`
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func Load(ctx context.Context) (Config, error) {
	// 1. Load base env vars
	cfg := Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		Port:                 getEnv("PORT", "8080"),
		AWSRegion:            getEnv("AWS_REGION", "us-east-1"),
		AWSEndpointURL:       os.Getenv("AWS_ENDPOINT_URL"),
		S3PresignEndpointURL: os.Getenv("S3_PRESIGN_ENDPOINT_URL"),
		AWSAccessKeyID:       getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey:   getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		SSMParameterName:     getEnv("SSM_PARAMETER_NAME", "/bowerbird/local/secrets"),
		EnableLocalEventLoop: getEnv("ENABLE_LOCAL_EVENT_LOOP", "true") == "true",
		AllowedOrigins:       getEnv("ALLOWED_ORIGINS", "*"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:4200"),
		BackendURL:           getEnv("BACKEND_URL", "http://localhost:8080"),
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

	if cfg.EventBusName == "" {
		cfg.EventBusName = os.Getenv("EVENT_BUS_NAME")
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required (from SSM or env)")
	}
	if cfg.InboxCredentialsEncryptionKey == "" {
		return Config{}, fmt.Errorf("inbox_credentials_encryption_key is required from SSM")
	}

	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
			accessSecret = "local-dev-access-secret-do-not-use-in-prod"
		} else {
			return Config{}, fmt.Errorf("JWT_ACCESS_SECRET is required")
		}
	}

	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
			refreshSecret = "local-dev-refresh-secret-do-not-use-in-prod"
		} else {
			return Config{}, fmt.Errorf("JWT_REFRESH_SECRET is required")
		}
	}

	cfg.JWT = JWTConfig{
		AccessSecret:  accessSecret,
		RefreshSecret: refreshSecret,
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
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
