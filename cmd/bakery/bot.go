package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"bakery/internal/config"
	"bakery/internal/deps"
	"bakery/internal/inbound/bot"
	orderoutbox "bakery/internal/services/order/infra/outbox"
)

const botConnectRetryDelay = 30 * time.Second

type botRunner struct {
	cfg     *config.Config
	appDeps *deps.AppDeps
	bot     atomic.Pointer[bot.OrderBot]
}

func newBotRunner(cfg *config.Config, appDeps *deps.AppDeps) (*botRunner, error) {
	if cfg.Telegram.BotToken == "" {
		return nil, fmt.Errorf("telegram bot token is not set for BOT_ENV=%s: it signs web sessions and runs the bot", cfg.Telegram.BotEnv)
	}
	if err := bot.ValidateMiniAppURL(cfg.Telegram.MiniAppURL); err != nil {
		return nil, fmt.Errorf("MINI_APP_URL: %w", err)
	}
	return &botRunner{cfg: cfg, appDeps: appDeps}, nil
}

func (r *botRunner) HandleOrderEvent(ctx context.Context, eventType string, payload []byte) error {
	orderBot := r.bot.Load()
	if orderBot == nil {
		return orderoutbox.ErrHandlerNotReady
	}
	return orderBot.HandleOrderEvent(ctx, eventType, payload)
}

func (r *botRunner) Run(ctx context.Context) error {
	for {
		orderBot, err := r.connect(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err == nil {
			r.serve(ctx, orderBot)
			return nil
		}
		if bot.IsRejectedToken(err) {
			return fmt.Errorf("telegram rejected the bot token: %w", err)
		}
		slog.ErrorContext(ctx, "telegram bot connect failed", "error", err, "retry_in", botConnectRetryDelay)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(botConnectRetryDelay):
		}
	}
}

type connectResult struct {
	bot *bot.OrderBot
	err error
}

func (r *botRunner) connect(ctx context.Context) (*bot.OrderBot, error) {
	done := make(chan connectResult, 1)
	go func() {
		orderBot, err := bot.NewOrderBot(
			r.cfg.Telegram.BotToken,
			r.appDeps.AuthService,
			r.appDeps.DepartmentService,
			r.cfg.Telegram.MiniAppURL,
			r.cfg.Telegram.WorkshopChatID,
		)
		done <- connectResult{bot: orderBot, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-done:
		return result.bot, result.err
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
