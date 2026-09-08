package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns        = int32(4)
	defaultMinConns        = int32(0)
	defaultMaxConnLifetime = 30 * time.Minute
	defaultMaxConnIdleTime = 30 * time.Second
)

func applyPoolDefaults(cfg *pgxpool.Config) {
	// Tenant registries cache one pool per tenant, per process (API, relay, consumers,
	// scheduler). MinConns>0 reserves connections for every tenant forever; the relay
	// also ticks every tenant every 5s, so idle timeouts never fire there.
	cfg.MaxConns = defaultMaxConns
	cfg.MinConns = defaultMinConns
	cfg.MaxConnLifetime = defaultMaxConnLifetime
	cfg.MaxConnIdleTime = defaultMaxConnIdleTime
}

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	applyPoolDefaults(cfg)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
