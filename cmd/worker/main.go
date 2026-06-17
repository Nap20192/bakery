package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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
	_ = loadDotenv()

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
		log.Info(
			"order cleanup started",
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

func loadDotenv() error {
	paths := []string{".env"}
	if wd, err := os.Getwd(); err == nil {
		for current := wd; ; current = filepath.Dir(current) {
			paths = append(paths, filepath.Join(current, ".env"))
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		for current := filepath.Dir(exe); ; current = filepath.Dir(current) {
			paths = append(paths, filepath.Join(current, ".env"))
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}

	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}

	for _, path := range unique {
		if err := godotenv.Load(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}
