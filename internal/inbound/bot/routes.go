package bot

import (
	"context"
	"log/slog"
	"time"

	"bakery/internal/app"
	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

const requestContextKey = "request_context"

func (b *OrderBot) register() {
	bt := b.tele
	bt.Use(b.logMiddleware)

	bt.Handle("/start", b.handleStart)
	bt.Handle("/help", b.handleStart)
	bt.Handle("/login", b.handleLogin)
	bt.Handle("/logout", b.handleLogout)
	bt.Handle("/adduser", b.handleAddUser)
	bt.Handle("/order", b.handleOrder)
	bt.Handle("/orders", b.handleOrders)
	bt.Handle("/templates", b.handleTemplates)
	bt.Handle("/addtemplate", b.handleAddTemplate, b.requirePermissions(app.PermissionTemplateManage))
	bt.Handle("/monitor", b.handleMonitor)
	bt.Handle("/sync", b.handleSync, b.requirePermissions(app.PermissionSync))
	// Авторизация оставлена только на служебные команды iiko.
	bt.Handle("/techcard", b.handleTechCard, b.requirePermissions(app.PermissionTechCard))
	bt.Handle("/template", b.handleTemplate)
	bt.Handle("/cancel", b.handleCancel)

	bt.Handle("\fconfirm", b.handleConfirm)
	bt.Handle("\fcancel_cb", b.handleCancelCallback)
	bt.Handle("\fsubmit_order", b.handleConfirm)
	bt.Handle("\fedit_order", b.handleEditOrder)
	bt.Handle("\fupdate_order", b.handleUpdateOrder)
	bt.Handle("\ftemplate_theme", b.handleTemplateTheme)
	bt.Handle("\ftemplate_use", b.handleTemplateUse)
	bt.Handle("\fopen_templates", b.handleTemplates)
	bt.Handle("\fopen_orders", b.handleOrders)
	bt.Handle("\fdept_shop", b.handleDepartmentShop)
	bt.Handle("\fdept_workshop", b.handleDepartmentWorkshop)
	bt.Handle("\fdept_select", b.handleDepartmentSelect)

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
			slog.ErrorContext(applog.ErrorContext(ctx, err), "telegram update failed",
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
