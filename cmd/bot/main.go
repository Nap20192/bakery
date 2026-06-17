// Command bot runs the Telegram bot as a standalone process, separate from the
// HTTP API / sync worker. For now it still builds the data-backed services it
// needs directly; a later stage swaps that backend for an HTTP API client so
// the bot talks to the application only over the API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"bakery/internal/config"
	"bakery/internal/deps"
	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/pkg/dbmigrate"
	"bakery/internal/pkg/helpers"
	"bakery/pkg/logger"
)

func main() {
	_ = config.LoadDotenv()

	cfg := config.New()

	log, err := logger.InitLogger(cfg.Log.Level, cfg.Log.Pretty, cfg.Log.Dir)
	if err != nil {
		panic(err)
	}
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := outbounddb.OpenPostgres(ctx, cfg.Database.URL)
	if err != nil {
		log.Error("open db failed", "error", err)
		os.Exit(1)
	}
	defer helpers.ClosePool(db)
	if err = dbmigrate.ApplyMigrations(ctx, db, log, cfg.Migration.Dir); err != nil {
		log.Error("apply db migrations failed", "error", err, "dir", cfg.Migration.Dir)
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
		deps.WithOrderEventBus(),
		deps.WithAuthService(infra),
		deps.WithRbacService(),
		deps.WithOrderService(infra),
		deps.WithDepartmentService(infra),
		deps.WithMonitorService(infra),
		deps.WithTechCardService(infra),
		deps.WithSyncService(infra),
		deps.WithOrderBot(infra),
	)
	if err != nil {
		log.Error("build bot deps failed", "error", err)
		os.Exit(1)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Info(
			"orderbot started",
			"bot_env", cfg.Telegram.BotEnv,
			"bot_name", appDeps.OrderBot.Name(),
			"bot_username", appDeps.OrderBot.Username(),
		)
		appDeps.OrderBot.Start()
		return nil
	})
	group.Go(func() error {
		<-groupCtx.Done()
		appDeps.OrderBot.Stop()
		return nil
	})

	if err := group.Wait(); err != nil {
		log.Error("bot stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("bot stopped")
}
