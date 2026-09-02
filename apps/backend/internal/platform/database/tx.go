package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey struct{}

func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func QuerierFromContext(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return pool
}

func RunInTx(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txCtx := WithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

type RegistryUnitOfWork struct {
	registry *Registry
}

func NewRegistryUnitOfWork(registry *Registry) *RegistryUnitOfWork {
	return &RegistryUnitOfWork{registry: registry}
}

func (u *RegistryUnitOfWork) Run(ctx context.Context, fn func(context.Context) error) error {
	pool, err := u.registry.GetPool(ctx)
	if err != nil {
		return err
	}
	return RunInTx(ctx, pool, fn)
}
