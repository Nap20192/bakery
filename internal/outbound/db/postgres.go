package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Retry ladder: 1+2+4+8+16 seconds across six attempts. Variables rather than
// constants so tests can shrink the ladder instead of sleeping through it.
var (
	connectAttempts    = 6
	connectBackoffBase = time.Second
	connectBackoffMax  = 16 * time.Second
)

// OpenPostgres builds the pool and waits for the database to answer.
//
// The retry covers startup only. Once the pool exists, pgxpool reconnects on
// its own: it discards broken connections, dials replacements on demand, and
// prunes dead ones every HealthCheckPeriod. Nothing here should try to repeat
// that. What it cannot do is come into being before the database accepts
// connections — and a worker that boots beside a cold or failing-over database
// used to exit(1) on the first refused ping instead of waiting a few seconds.
func OpenPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute
	logPostgresConnectStart(config)

	backoff := connectBackoffBase
	for attempt := 1; ; attempt++ {
		pool, err := openPostgresOnce(ctx, config)
		if err == nil {
			slog.InfoContext(ctx, "postgres connected", append(postgresLogAttrs(config), "attempt", attempt)...)
			return pool, nil
		}
		if attempt >= connectAttempts || ctx.Err() != nil {
			slog.ErrorContext(ctx, "postgres connect failed", "error", err, "attempts", attempt)
			return nil, err
		}
		slog.WarnContext(ctx, "postgres connect failed, retrying", "error", err, "attempt", attempt, "in", backoff)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < connectBackoffMax {
			backoff *= 2
		}
	}
}

func openPostgresOnce(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func logPostgresConnectStart(config *pgxpool.Config) {
	slog.Info("postgres connecting", postgresLogAttrs(config)...)
}

func postgresLogAttrs(config *pgxpool.Config) []any {
	if config == nil || config.ConnConfig == nil {
		return nil
	}
	attrs := []any{
		"host", config.ConnConfig.Host,
		"port", config.ConnConfig.Port,
		"database", config.ConnConfig.Database,
		"user", config.ConnConfig.User,
		"sslmode", config.ConnConfig.RuntimeParams["sslmode"],
		"min_conns", config.MinConns,
		"max_conns", config.MaxConns,
	}
	return attrs
}
