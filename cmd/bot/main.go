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
	"bakery/internal/inbound/bot"
	outbounddb "bakery/internal/outbound/db"
	"bakery/internal/pkg/dbmigrate"
	"bakery/pkg/logger"
	"bakery/pkg/rabbitmq"
)

func main() {
	config.LoadDotenv()

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
	defer db.Close()
	if err = dbmigrate.ApplyMigrations(ctx, db, log, cfg.Migration.Dir); err != nil {
		log.Error("apply db migrations failed", "error", err, "dir", cfg.Migration.Dir)
		os.Exit(1)
	}

	rabbitConn, err := rabbitmq.NewConn(rabbitmq.ConnString(cfg.RabbitMQ.URL))
	if err != nil {
		log.Error("connect rabbitmq failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = rabbitConn.Close() }()
	slog.Info("connected to rabbitmq", "url", cfg.RabbitMQ.URL)
	slog.Info("starting bot", "bot_env", cfg.Telegram.BotEnv)
	slog.Info("chat_id", "chat_id", cfg.Telegram.WorkshopChatID)

	if cfg.Telegram.BotToken == "" {
		switch cfg.Telegram.BotEnv {
		case "prod", "production":
			log.Error("PROD_BOT_TOKEN не задан")
		default:
			log.Error("TEST_BOT_TOKEN не задан")
		}
		os.Exit(1)
	}

	infra, err := deps.NewInfraDeps(
		deps.WithConfig(cfg),
		deps.WithPostgres(db),
		deps.WithRepositories(),
		deps.WithRabbitMQ(rabbitConn),
	)
	if err != nil {
		log.Error("build infra deps failed", "error", err)
		os.Exit(1)
	}

	appDeps, err := deps.NewAppDeps(
		deps.WithAuthService(infra),
		deps.WithDepartmentService(infra),
	)
	if err != nil {
		log.Error("build bot deps failed", "error", err)
		os.Exit(1)
	}

	orderBot, err := bot.NewOrderBot(
		cfg.Telegram.BotToken,
		appDeps.AuthService,
		appDeps.DepartmentService,
		infra.EventConsumer(),
		cfg.Telegram.MiniAppURL,
		cfg.Telegram.WorkshopChatID,
	)
	if err != nil {
		log.Error("build order bot failed", "error", err)
		os.Exit(1)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Info(
			"orderbot started",
			"bot_env", cfg.Telegram.BotEnv,
			"bot_name", orderBot.Name(),
			"bot_username", orderBot.Username(),
		)
		orderBot.Start()
		return nil
	})
	group.Go(func() error {
		log.Info("order events consumer started")
		return orderBot.ConsumeOrderEvents(groupCtx)
	})
	group.Go(func() error {
		<-groupCtx.Done()
		orderBot.Stop()
		return nil
	})

	if err := group.Wait(); err != nil {
		log.Error("bot stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("bot stopped")
}
