package postgresql

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreatePgxConnectionPool(ctx context.Context, sourceURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, sourceURL)
	if err != nil {
		log.Fatal(err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return pool
}
