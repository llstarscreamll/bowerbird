package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	awsConfig "github.com/bowerbird/internal/platform/awsconfig"
)

type Config struct {
	AppEnv                        string    `json:"app_env"`
	DeploymentTarget              string    `json:"deployment_target"`
	DefaultTenantSlug             string    `json:"default_tenant_slug"`
	Port                          string    `json:"port"`
	DatabaseURL                   string    `json:"database_url"`
	SQSQueueURL                   string    `json:"sqs_queue_url"`
	EventBridgeQueueURL           string    `json:"eventbridge_queue_url"`
	EventBusName                  string    `json:"event_bus_name"`
	S3BucketName                  string    `json:"s3_bucket_name"`
	S3PresignEndpointURL          string    `json:"s3_presign_endpoint_url"`
	MinIOEndpointURL              string    `json:"minio_endpoint_url"`
	RabbitMQURL                   string    `json:"rabbitmq_url"`
	AWSRegion                     string    `json:"aws_region"`
	AWSEndpointURL                string    `json:"aws_endpoint_url"`
	AWSAccessKeyID                string    `json:"aws_access_key_id"`
	AWSSecretAccessKey            string    `json:"aws_secret_access_key"`
	SSMParameterName              string    `json:"ssm_parameter_name"`
	AllowedOrigins                string    `json:"allowed_origins"`
	Debug                         bool      `json:"debug"`
	GoogleClientID                string    `json:"google_client_id"`
	GoogleClientSecret            string    `json:"google_client_secret"`
	MicrosoftClientID             string    `json:"microsoft_client_id"`
	MicrosoftClientSecret         string    `json:"microsoft_client_secret"`
	GeminiAPIKey                  string    `json:"gemini_api_key"`
	GeminiModel                   string    `json:"gemini_model"`
	GeminiEndpoint                string    `json:"gemini_endpoint"`
	InboxCredentialsEncryptionKey string    `json:"inbox_credentials_encryption_key"`
	TenantSecretsEncryptionKey    string    `json:"tenant_secrets_encryption_key"`
	MessagingAttestationSecret    string    `json:"messaging_attestation_secret"`
	FrontendURL                   string    `json:"frontend_url"`
	BackendURL                    string    `json:"backend_url"`
	PlatformOperatorEmails        []string  `json:"-"`
	JWT                           JWTConfig `json:"-"`
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

const (
	DeploymentTargetAWS    = "aws"
	DeploymentTargetOnPrem = "onprem"
)

func Load(ctx context.Context) (Config, error) {
	deploymentTarget := getEnv("DEPLOYMENT_TARGET", DeploymentTargetOnPrem)
	if deploymentTarget != DeploymentTargetAWS && deploymentTarget != DeploymentTargetOnPrem {
		return Config{}, fmt.Errorf("invalid DEPLOYMENT_TARGET: %q", deploymentTarget)
	}

	cfg := Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		DeploymentTarget:       deploymentTarget,
		DefaultTenantSlug:      getEnv("DEFAULT_TENANT_SLUG", "acme"),
		Port:                   getEnv("PORT", "8080"),
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		AWSEndpointURL:         os.Getenv("AWS_ENDPOINT_URL"),
		MinIOEndpointURL:       os.Getenv("MINIO_ENDPOINT_URL"),
		RabbitMQURL:            os.Getenv("RABBITMQ_URL"),
		S3PresignEndpointURL:   os.Getenv("S3_PRESIGN_ENDPOINT_URL"),
		AWSAccessKeyID:         getEnv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey:     getEnv("AWS_SECRET_ACCESS_KEY", "test"),
		AllowedOrigins:         getEnv("ALLOWED_ORIGINS", "https://app.bowerbird.dev,http://app.bowerbird.dev,http://localhost:4200"),
		FrontendURL:            getEnv("FRONTEND_URL", "https://app.bowerbird.dev"),
		BackendURL:             getEnv("BACKEND_URL", "https://api.bowerbird.dev"),
		PlatformOperatorEmails: parseCSVList(os.Getenv("PLATFORM_OPERATOR_EMAILS")),
	}

	defaultDebug := cfg.AppEnv == "development" || cfg.AppEnv == "local"
	cfg.Debug = getEnvAsBool("DEBUG", defaultDebug)

	if cfg.DeploymentTarget == DeploymentTargetAWS {
		cfg.SSMParameterName = getEnv("SSM_PARAMETER_NAME", "/bowerbird/local/secrets")
		awsCfg, err := awsConfig.Load(ctx, cfg.AWSRegion, cfg.AWSEndpointURL, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
		if err != nil {
			return cfg, fmt.Errorf("load aws config for ssm: %w", err)
		}
		if cfg.SSMParameterName != "" {
			if err := loadSSMSecrets(ctx, awsCfg, cfg.AWSEndpointURL, &cfg); err != nil {
				return cfg, fmt.Errorf("load ssm secrets: %w", err)
			}
		}
	}

	// Fallback to env vars
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	if cfg.RabbitMQURL == "" {
		cfg.RabbitMQURL = os.Getenv("RABBITMQ_URL")
	}
	if cfg.MinIOEndpointURL == "" {
		cfg.MinIOEndpointURL = os.Getenv("MINIO_ENDPOINT_URL")
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

	if cfg.DeploymentTarget == DeploymentTargetOnPrem {
		loadOnPremEnv(&cfg)
	}

	if cfg.DatabaseURL == "" {
		panic("DATABASE_URL is required (from SSM or env)")
	}
	if cfg.InboxCredentialsEncryptionKey == "" {
		panic("inbox_credentials_encryption_key is required from SSM or env")
	}
	if cfg.TenantSecretsEncryptionKey == "" {
		panic("tenant_secrets_encryption_key is required from SSM or env")
	}
	if cfg.EventBusName == "" && cfg.DeploymentTarget == DeploymentTargetAWS {
		panic("EVENT_BUS_NAME is required for aws deployment")
	}
	if cfg.S3BucketName == "" {
		panic("S3_BUCKET_NAME is required (from SSM or env)")
	}
	if cfg.DeploymentTarget == DeploymentTargetOnPrem && cfg.RabbitMQURL == "" {
		panic("RABBITMQ_URL is required for onprem deployment")
	}
	if cfg.GeminiAPIKey == "" {
		panic("GEMINI_API_KEY is required (from SSM or env)")
	}

	if cfg.MessagingAttestationSecret == "" {
		if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
			cfg.MessagingAttestationSecret = localMessagingAttestation
		}
	}

	if err := validateSecurityConfig(cfg); err != nil {
		panic(err.Error())
	}

	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
			accessSecret = "local-dev-access-secret-do-not-use-in-prod"
		} else {
			panic("JWT_ACCESS_SECRET is required")
		}
	}

	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
			refreshSecret = "local-dev-refresh-secret-do-not-use-in-prod"
		} else {
			panic("JWT_REFRESH_SECRET is required")
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

func loadOnPremEnv(cfg *Config) {
	if cfg.GoogleClientID == "" {
		cfg.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	if cfg.GoogleClientSecret == "" {
		cfg.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}
	if cfg.MicrosoftClientID == "" {
		cfg.MicrosoftClientID = os.Getenv("MICROSOFT_CLIENT_ID")
	}
	if cfg.MicrosoftClientSecret == "" {
		cfg.MicrosoftClientSecret = os.Getenv("MICROSOFT_CLIENT_SECRET")
	}
	if cfg.GeminiAPIKey == "" {
		cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = getEnv("GEMINI_MODEL", "gemini-2.0-flash")
	}
	if cfg.GeminiEndpoint == "" {
		cfg.GeminiEndpoint = getEnv("GEMINI_ENDPOINT", "https://generativelanguage.googleapis.com")
	}
	if cfg.InboxCredentialsEncryptionKey == "" {
		cfg.InboxCredentialsEncryptionKey = os.Getenv("INBOX_CREDENTIALS_ENCRYPTION_KEY")
	}
	if cfg.TenantSecretsEncryptionKey == "" {
		cfg.TenantSecretsEncryptionKey = os.Getenv("TENANT_SECRETS_ENCRYPTION_KEY")
	}
	if cfg.MessagingAttestationSecret == "" {
		cfg.MessagingAttestationSecret = os.Getenv("MESSAGING_ATTESTATION_SECRET")
	}
}

var (
	exampleInboxCredentialsKey = "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="
	exampleTenantSecretsKey    = "Ym93ZXJiaXJkLWxvY2FsLXNlY3JldHMta2V5LTMyYiE="
	localMessagingAttestation  = "local-dev-messaging-attestation-secret"
)

func validateSecurityConfig(cfg Config) error {
	if cfg.MessagingAttestationSecret == "" {
		return fmt.Errorf("MESSAGING_ATTESTATION_SECRET is required")
	}
	if cfg.AppEnv == "local" {
		return nil
	}
	if cfg.InboxCredentialsEncryptionKey == exampleInboxCredentialsKey {
		return fmt.Errorf("INBOX_CREDENTIALS_ENCRYPTION_KEY must not use the example value outside APP_ENV=local")
	}
	if cfg.TenantSecretsEncryptionKey == exampleTenantSecretsKey {
		return fmt.Errorf("TENANT_SECRETS_ENCRYPTION_KEY must not use the example value outside APP_ENV=local")
	}
	if cfg.MessagingAttestationSecret == localMessagingAttestation {
		return fmt.Errorf("MESSAGING_ATTESTATION_SECRET must not use the local default outside APP_ENV=local")
	}
	return nil
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

func parseCSVList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := strings.ToLower(strings.TrimSpace(part))
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
