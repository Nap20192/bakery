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
	if err := dbmigrate.ApplyMigrations(ctx, db, log, cfg.Migration.Dir); err != nil {
		log.Error("apply db migrations failed", "error", err)
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
		deps.WithAuthService(infra),
		deps.WithRbacService(),
		deps.WithOrderService(infra),
		deps.WithDepartmentService(infra),
		deps.WithMonitorService(infra),
		deps.WithTechCardService(infra),
		deps.WithSyncService(infra),
		deps.WithOrderBot(infra),
		deps.WithAPIServerConfig(infra),
	)
	if err != nil {
		log.Error("build app deps failed", "error", err)
		os.Exit(1)
	}

	admin, created, err := appDeps.AuthService.EnsureAdminUser(
		ctx,
		cfg.Admin.Username,
		cfg.Admin.Password,
	)
	if err != nil {
		log.Error("ensure admin failed", "error", err)
		os.Exit(1)
	}
	log.Info("admin user ready", "username", admin.Username, "role", admin.Role, "created", created)

	templateSeed, err := appDeps.OrderService.EnsureDefaultOrderTemplates(ctx, "templates/dishes.txt")
	if err != nil {
		log.Error("ensure default templates failed", "error", err)
		os.Exit(1)
	}
	log.Info("dish catalog ready", "catalog_items", templateSeed.CatalogItems)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Info("sync service started")
		return appDeps.SyncService.Run(groupCtx)
	})
	group.Go(func() error {
		log.Info("order cleanup started",
			"interval", cfg.OrderCleanup.Interval.String(),
			"retention", cfg.OrderCleanup.Retention.String(),
		)
		return appDeps.OrderService.RunCleanupTicker(groupCtx, cfg.OrderCleanup.Interval, cfg.OrderCleanup.Retention)
	})
	group.Go(func() error {
		log.Info("http api started", "addr", cfg.Server.Addr())
		return appDeps.APIServer.Start()
	})
	group.Go(func() error {
		log.Info("orderbot started",
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		return appDeps.APIServer.Shutdown(shutdownCtx)
	})

	if err := group.Wait(); err != nil {
		log.Error("worker stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("worker stopped")
}
