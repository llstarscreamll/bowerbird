package config

import (
	"context"
	"os"
	"testing"
)

func TestLoad_DefaultDeploymentTargetOnPrem(t *testing.T) {
	os.Unsetenv("DEPLOYMENT_TARGET")
	t.Setenv("APP_ENV", "local")
	t.Setenv("DATABASE_URL", "postgres://bowerbird:bowerbird@localhost:5432/bowerbird?sslmode=disable")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	t.Setenv("INBOX_CREDENTIALS_ENCRYPTION_KEY", "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	t.Setenv("TENANT_SECRETS_ENCRYPTION_KEY", "Ym93ZXJiaXJkLWxvY2FsLXNlY3JldHMta2V5LTMyYiE=")

	cfg, err := Load(context.Background())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DeploymentTarget != DeploymentTargetOnPrem {
		t.Fatalf("expected onprem default, got %q", cfg.DeploymentTarget)
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
	t.Setenv("DATABASE_URL", "postgres://bowerbird:bowerbird@localhost:5432/bowerbird?sslmode=disable")
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

func TestLoad_AWSUsesSSMParameterName(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("DEPLOYMENT_TARGET", DeploymentTargetAWS)
	t.Setenv("SSM_PARAMETER_NAME", "/bowerbird/test/secrets")
	t.Setenv("DATABASE_URL", "postgres://bowerbird:bowerbird@localhost:5432/bowerbird?sslmode=disable")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("EVENT_BUS_NAME", "test-bus")
	t.Setenv("INBOX_CREDENTIALS_ENCRYPTION_KEY", "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	t.Setenv("TENANT_SECRETS_ENCRYPTION_KEY", "Ym93ZXJiaXJkLWxvY2FsLXNlY3JldHMta2V5LTMyYiE=")
	t.Setenv("MINIO_ENDPOINT_URL", "http://localhost:9000")

	_, err := Load(context.Background())
	if err == nil {
		t.Fatal("expected SSM load error without parameter present")
	}
}

func TestLoad_AWSRequiresEventBusName(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("DEPLOYMENT_TARGET", DeploymentTargetAWS)
	t.Setenv("DATABASE_URL", "postgres://bowerbird:bowerbird@localhost:5432/bowerbird?sslmode=disable")
	t.Setenv("S3_BUCKET_NAME", "test-bucket")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("INBOX_CREDENTIALS_ENCRYPTION_KEY", "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=")
	t.Setenv("TENANT_SECRETS_ENCRYPTION_KEY", "Ym93ZXJiaXJkLWxvY2FsLXNlY3JldHMta2V5LTMyYiE=")
	t.Setenv("SSM_PARAMETER_NAME", "")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing EVENT_BUS_NAME on aws")
		}
	}()
	_, _ = Load(context.Background())
}

func init() {
	_ = os.Setenv("JWT_ACCESS_SECRET", "test-access")
	_ = os.Setenv("JWT_REFRESH_SECRET", "test-refresh")
}
