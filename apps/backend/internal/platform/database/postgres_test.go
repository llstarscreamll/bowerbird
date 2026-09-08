package database

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyPoolDefaultsDoesNotReserveConnections(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://bowerbird:bowerbird@localhost:5432/bowerbird")
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	applyPoolDefaults(cfg)

	if cfg.MinConns != 0 {
		t.Fatalf("MinConns=%d, want 0 so each cached tenant pool does not reserve connections", cfg.MinConns)
	}
	if cfg.MaxConns != 4 {
		t.Fatalf("MaxConns=%d, want 4", cfg.MaxConns)
	}
	if cfg.MaxConnIdleTime != 30*time.Second {
		t.Fatalf("MaxConnIdleTime=%s, want 30s", cfg.MaxConnIdleTime)
	}
}
