package infrastructure

import (
	"testing"

	inboxapp "github.com/money-path/bowerbird/apps/backend/internal/inbox/application"
	"github.com/money-path/bowerbird/apps/backend/internal/inbox/domain"
)

func TestPostgresRepositoryImplementsRepository(t *testing.T) {
	var _ domain.SyncCursorRepository = (*PostgresRepository)(nil)
	var _ domain.MessageRepository = (*PostgresRepository)(nil)
	var _ inboxapp.MessageQueryRepository = (*PostgresRepository)(nil)
}

func TestDefaultRawData(t *testing.T) {
	if got := string(defaultRawData(nil)); got != "{}" {
		t.Fatalf("expected default raw data to be {}, got %q", got)
	}

	expected := "{\"foo\":\"bar\"}"
	if got := string(defaultRawData([]byte(expected))); got != expected {
		t.Fatalf("expected raw data passthrough, got %q", got)
	}
}
