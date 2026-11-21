package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"bakery/frontend"
	"bakery/internal/config"
	"bakery/internal/deps"
	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/outbound/db/pgimport"
	"bakery/internal/pkg/dbmigrate"
	"bakery/pkg/logger"
)

const apiCallTimeout = 20 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("bakery stopped with error", "error", err)
		os.Exit(1)
	}
	slog.Info("bakery stopped")
}

func run() error {
	config.LoadDotenv()
	cfg := config.New()

	log, err := logger.InitLogger(cfg.Log.Level, cfg.Log.Pretty, cfg.Log.Dir)
	if err != nil {
		return err
	}
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := importPostgresOnce(ctx, cfg); err != nil {
		return err
	}
	db, err := outbounddb.OpenSQLite(ctx, cfg.Database.Path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if err := dbmigrate.ApplyMigrations(ctx, db, log, cfg.Migration.Dir); err != nil {
		return err
	}

	infra, err := deps.NewInfraDeps(
		deps.WithConfig(cfg),
		deps.WithDatabase(db),
		deps.WithRepositories(),
		deps.WithIikoClient(),
	)
	if err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)
	appDeps, err := deps.NewAppDeps(
		deps.WithAuthService(infra),
		deps.WithOrderService(infra),
		deps.WithDepartmentService(infra),
		deps.WithMonitorService(infra),
		deps.WithTechCardService(infra),
		deps.WithSyncService(infra),
		deps.WithAdminService(),
		deps.WithAPIServerConfig(groupCtx, infra),
	)
	if err != nil {
		return err
	}

	admin, created, err := appDeps.AuthService.EnsureAdminUser(ctx, cfg.Admin.Username, cfg.Admin.Password)
	if err != nil {
		return err
	}
	log.Info("admin user ready", "username", admin.Username, "role", admin.Role, "created", created)

	templateSeed, err := appDeps.OrderService.EnsureDefaultOrderTemplates(ctx)
	if err != nil {
		return err
	}
	log.Info("dish catalog ready", "catalog_items", templateSeed.CatalogItems)

	runner, err := newBotRunner(cfg, appDeps)
	if err != nil {
		return err
	}
	relay, err := deps.NewOrderOutboxRelay(infra, runner)
	if err != nil {
		return err
	}

	uiHandler, err := frontend.NewHandler(appDeps.APIServer.Handler(), apiCallTimeout, log)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           uiHandler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	group.Go(func() error {
		return appDeps.SyncService.Run(groupCtx)
	})
	group.Go(func() error {
		return appDeps.OrderService.RunCleanupTicker(groupCtx, cfg.OrderCleanup.Interval, cfg.OrderCleanup.Retention)
	})
	group.Go(func() error {
		return relay.Run(groupCtx)
	})
	group.Go(func() error {
		log.Info("http server started", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	group.Go(func() error {
		return runner.Run(groupCtx)
	})
	group.Go(func() error {
		<-groupCtx.Done()
		defer appDeps.APIServer.Shutdown()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	})
	return group.Wait()
}

func importPostgresOnce(ctx context.Context, cfg *config.Config) error {
	if cfg.Database.ImportPostgresURL == "" {
		return nil
	}
	_, err := os.Stat(cfg.Database.Path)
	switch {
	case err == nil:
		slog.WarnContext(ctx, "IMPORT_POSTGRES_URL is set but the database already exists; import skipped", "path", cfg.Database.Path)
		return nil
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("stat sqlite database: %w", err)
	}
	return pgimport.Import(ctx, cfg.Database.ImportPostgresURL, cfg.Database.Path, cfg.Migration.Dir)
}
