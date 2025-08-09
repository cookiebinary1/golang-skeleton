// Package postgres holds shared pgx pool setup and readiness checks.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pgx connection pool, retrying until the database accepts
// connections (needed when services race the DB container at startup).
func Connect(ctx context.Context, log *slog.Logger, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.ConnConfig.ConnectTimeout = 3 * time.Second
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour

	var pool *pgxpool.Pool
	const attempts = 30
	for i := 1; i <= attempts; i++ {
		p, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			perr := p.Ping(ctx)
			if perr == nil {
				pool = p
				break
			}
			p.Close()
			err = perr
		}
		if i == attempts {
			return nil, fmt.Errorf("postgres not reachable after %d attempts: %w", attempts, err)
		}
		log.Warn("postgres not ready, retrying", "attempt", i, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	log.Info("connected to postgres")
	return pool, nil
}

// Ping reports whether the pool can serve queries; used by /readyz.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	return pool.Ping(ctx)
}
