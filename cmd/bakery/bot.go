package main

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"bakery/internal/config"
	"bakery/internal/deps"
	"bakery/internal/inbound/bot"
)

const botConnectRetryDelay = 30 * time.Second

var errBotNotReady = errors.New("telegram bot is not connected yet")

type botRunner struct {
	cfg     *config.Config
	appDeps *deps.AppDeps
	bot     atomic.Pointer[bot.OrderBot]
}

func newBotRunner(cfg *config.Config, appDeps *deps.AppDeps) (*botRunner, error) {
	if cfg.Telegram.BotToken == "" {
		if cfg.Telegram.BotEnv == "prod" || cfg.Telegram.BotEnv == "production" {
			return nil, errors.New("PROD_BOT_TOKEN is not set")
		}
		slog.Warn("telegram bot disabled: no bot token", "bot_env", cfg.Telegram.BotEnv)
		return nil, nil
	}
	return &botRunner{cfg: cfg, appDeps: appDeps}, nil
}

func (r *botRunner) HandleOrderEvent(ctx context.Context, eventType string, payload []byte) error {
	orderBot := r.bot.Load()
	if orderBot == nil {
		return errBotNotReady
	}
	return orderBot.HandleOrderEvent(ctx, eventType, payload)
}

func (r *botRunner) Run(ctx context.Context) {
	for {
		orderBot, err := bot.NewOrderBot(
			r.cfg.Telegram.BotToken,
			r.appDeps.AuthService,
			r.appDeps.DepartmentService,
			r.cfg.Telegram.MiniAppURL,
			r.cfg.Telegram.WorkshopChatID,
		)
		if err == nil {
			r.serve(ctx, orderBot)
			return
		}
		slog.ErrorContext(ctx, "telegram bot connect failed", "error", err, "retry_in", botConnectRetryDelay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(botConnectRetryDelay):
		}
	}
}

func (r *botRunner) serve(ctx context.Context, orderBot *bot.OrderBot) {
	r.bot.Store(orderBot)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		orderBot.Stop()
	}()
	slog.InfoContext(ctx, "telegram bot started", "bot_env", r.cfg.Telegram.BotEnv, "bot_username", orderBot.Username())
	orderBot.Start()
	<-stopped
}
