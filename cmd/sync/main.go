package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"bakery/internal/config"
	"bakery/internal/deps"
	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/pkg/dbmigrate"
	"bakery/internal/pkg/helpers"
	"bakery/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	log, err := logger.InitLogger(helpers.Env("LOG_LEVEL", "INFO"), helpers.EnvBool("LOG_PRETTY", true), helpers.Env("LOG_DIR", ""))
	if err != nil {
		panic(err)
	}
	slog.SetDefault(log)

	cfg := config.New()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := outbounddb.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("open db failed", "error", err)
		os.Exit(1)
	}
	defer helpers.ClosePool(db)
	if err := dbmigrate.ApplyInitialSchema(ctx, db, log); err != nil {
		log.Error("apply db schema failed", "error", err)
		os.Exit(1)
	}

	infra, err := deps.NewInfraDeps(
		deps.WithConfig(cfg),
		deps.WithPostgres(db),
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
