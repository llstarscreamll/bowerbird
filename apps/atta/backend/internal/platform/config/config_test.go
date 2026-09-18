package config

import (
	"context"
	"os"
	"testing"
)

func TestDirectDatabaseURLFallsBackToDatabaseURL(t *testing.T) {
	cfg := Config{DatabaseURL: "postgres://pooled/atta"}
	if got := cfg.DirectDatabaseURL(); got != cfg.DatabaseURL {
		t.Fatalf("expected fallback %q, got %q", cfg.DatabaseURL, got)
	}
	cfg.DatabaseDirectURL = "postgres://direct/atta"
	if got := cfg.DirectDatabaseURL(); got != cfg.DatabaseDirectURL {
		t.Fatalf("expected direct url, got %q", got)
	}
}

func TestLoad_DefaultDeploymentTargetOnPrem(t *testing.T) {
	os.Unsetenv("DEPLOYMENT_TARGET")
	os.Unsetenv("BACKEND_URL")
	os.Unsetenv("FRONTEND_URL")
	t.Setenv("APP_ENV", "local")
	t.Setenv("DATABASE_URL", "postgres://atta:atta@localhost:5432/atta?sslmode=disable")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	t.Setenv("INBOX_CREDENTIALS_ENCRYPTION_KEY", "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	t.Setenv("TENANT_SECRETS_ENCRYPTION_KEY", "YXR0YS1sb2NhbC1zZWNyZXRzLWtleS0zMmJ5dGVzISE=")

	cfg, err := Load(context.Background())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DeploymentTarget != DeploymentTargetOnPrem {
		t.Fatalf("expected onprem default, got %q", cfg.DeploymentTarget)
	}
	if cfg.BackendURL != "https://app.atta.dev" {
		t.Fatalf("expected same-origin backend URL, got %q", cfg.BackendURL)
	}
	if cfg.FrontendURL != "https://app.atta.dev" {
		t.Fatalf("expected frontend URL, got %q", cfg.FrontendURL)
	}
}

func TestLoad_InvalidDeploymentTarget(t *testing.T) {
	t.Setenv("DEPLOYMENT_TARGET", "gcp")
	_, err := Load(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid deployment target")
	}
}

func TestLoad_RejectsExampleEncryptionKeysOutsideLocal(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://atta:atta@localhost:5432/atta?sslmode=disable")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	t.Setenv("INBOX_CREDENTIALS_ENCRYPTION_KEY", exampleInboxCredentialsKey)
	t.Setenv("TENANT_SECRETS_ENCRYPTION_KEY", exampleTenantSecretsKey)
	t.Setenv("MESSAGING_ATTESTATION_SECRET", "prod-attestation-secret")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for example encryption keys outside local")
		}
	}()
	_, _ = Load(context.Background())
}

func TestLoad_AWSRequiresSSMParameterName(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("DEPLOYMENT_TARGET", DeploymentTargetAWS)
	t.Setenv("SSM_PARAMETER_NAME", "")
	t.Setenv("DATABASE_URL", "postgres://atta:atta@localhost:5432/atta?sslmode=disable")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("INBOX_CREDENTIALS_ENCRYPTION_KEY", "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	t.Setenv("TENANT_SECRETS_ENCRYPTION_KEY", "YXR0YS1sb2NhbC1zZWNyZXRzLWtleS0zMmJ5dGVzISE=")

	_, err := Load(context.Background())
	if err == nil {
		t.Fatal("expected error when SSM_PARAMETER_NAME is empty on aws")
	}
}

func TestLoad_AWSUsesSSMParameterName(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("DEPLOYMENT_TARGET", DeploymentTargetAWS)
	t.Setenv("SSM_PARAMETER_NAME", "/atta/test/secrets")
	t.Setenv("DATABASE_URL", "postgres://atta:atta@localhost:5432/atta?sslmode=disable")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("EVENT_BUS_NAME", "test-bus")
	t.Setenv("INBOX_CREDENTIALS_ENCRYPTION_KEY", "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	t.Setenv("TENANT_SECRETS_ENCRYPTION_KEY", "YXR0YS1sb2NhbC1zZWNyZXRzLWtleS0zMmJ5dGVzISE=")
	t.Setenv("MINIO_ENDPOINT_URL", "http://localhost:9000")

	_, err := Load(context.Background())
	if err == nil {
		t.Fatal("expected SSM load error without parameter present")
	}
}

func init() {
	_ = os.Setenv("JWT_ACCESS_SECRET", "test-access")
	_ = os.Setenv("JWT_REFRESH_SECRET", "test-refresh")
	_ = os.Setenv("AWS_ACCESS_KEY_ID", "atta")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "attasecret")
}
