package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"

	"bakery/internal/config"
	"bakery/internal/deps"
	"bakery/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	log, err := logger.InitLogger(getEnv("LOG_LEVEL", "INFO"), getEnvBool("LOG_PRETTY", true), getEnv("LOG_DIR", ""))
	if err != nil {
		panic(err)
	}
	slog.SetDefault(log)

	cfg := config.New()
	db, err := openSQLite(cfg.DBPath)
	if err != nil {
		log.Error("open db failed", "error", err)
		os.Exit(1)
	}
	defer closeDB(log, db)

	infra, err := deps.NewInfraDeps(
		deps.WithConfig(cfg),
		deps.WithSQLite(db),
		deps.WithRepositories(),
		deps.WithIikoClient(),
	)
	if err != nil {
		log.Error("build infra deps failed", "error", err)
		os.Exit(1)
	}

	appDeps, err := deps.NewAppDeps(
		deps.WithSyncService(infra),
	)
	if err != nil {
		log.Error("build app deps failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Info("sync service started")
		return appDeps.SyncService.Run(groupCtx)
	})

	if err := group.Wait(); err != nil {
		log.Error("sync service stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("sync service stopped")
}

func openSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared&mode=rwc", path))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func closeDB(log *slog.Logger, db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		log.Error("close db failed", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
