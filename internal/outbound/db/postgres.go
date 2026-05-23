package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func OpenPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour
	logPostgresConnectStart(config)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.ErrorContext(ctx, "postgres pool create failed", "error", err)
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		slog.ErrorContext(ctx, "postgres ping failed", "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, "postgres connected", postgresLogAttrs(config)...)
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
