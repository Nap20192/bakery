package bot

import (
	"context"
	"log/slog"
	"time"

	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

const requestContextKey = "request_context"

// register wires the only two updates the bot handles: /start (begin the
// password gate) and free text (the password itself). Everything else lives in
// the mini app now.
func (b *OrderBot) register() {
	bt := b.tele
	bt.Use(b.privateChatOnly, b.logMiddleware)

	bt.Handle("/start", b.handleStart)
	bt.Handle(tele.OnText, b.handleText)
}

func (b *baseBot) logMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		start := time.Now()
		ctx := context.Background()
		if sender := c.Sender(); sender != nil {
			ctx = applog.WithTelegramUser(ctx, sender.ID, sender.Username)
		}

		if cb := c.Callback(); cb != nil {
			ctx = applog.WithPayload(ctx, applog.CallbackPayload(cb.Unique, cb.Data))
		} else if c.Message() != nil {
			ctx = applog.WithPayload(ctx, applog.MessagePayload(c.Text()))
		} else {
			ctx = applog.WithPayload(ctx, applog.UpdatePayload())
		}
		c.Set(requestContextKey, ctx)

		err := next(c)

		if err != nil {
			slog.ErrorContext(ctx, "telegram update failed",
				"error", err,
				"duration", time.Since(start).Round(time.Millisecond).String(),
			)
			return err
		}

		slog.InfoContext(ctx, "telegram update",
			"status", "ok",
			"duration", time.Since(start).Round(time.Millisecond).String(),
		)
		return err
	}
}

func requestContext(c tele.Context) context.Context {
	if raw := c.Get(requestContextKey); raw != nil {
		if ctx, ok := raw.(context.Context); ok {
			return ctx
		}
	}
	return context.Background()
}
