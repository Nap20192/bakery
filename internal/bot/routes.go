package bot

import (
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) register() {
	bt := b.tele
	bt.Use(b.logMiddleware)

	bt.Handle("/start", b.handleStart)
	bt.Handle("/login", b.handleLogin)
	bt.Handle("/logout", b.handleLogout)
	bt.Handle("/adduser", b.handleAddUser, b.requirePermissions(permManageUsers))
	bt.Handle("/addgroup", b.handleAddGroup, b.requirePermissions(permManageGroups))
	bt.Handle("/groups", b.handleGroups, b.requirePermissions(permMonitor))
	bt.Handle("/orders", b.handleOrders, b.requirePermissions(permViewOrders))
	bt.Handle("/monitor", b.handleMonitor, b.requirePermissions(permMonitor))
	bt.Handle("/template", b.handleTemplate, b.requirePermissions(permCreateOrder))
	bt.Handle("/cancel", b.handleCancel, b.requirePermissions(permCreateOrder))

	bt.Handle("\fconfirm", b.handleConfirm, b.requirePermissions(permCreateOrder))
	bt.Handle("\fcancel_cb", b.handleCancelCallback, b.requirePermissions(permCreateOrder))

	bt.Handle(tele.OnText, b.handleText)
}

func (b *baseBot) logMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		start := time.Now()
		sender := c.Sender()

		var kind, payload string
		if cb := c.Callback(); cb != nil {
			kind = "callback"
			payload = cb.Unique + " data=" + cb.Data
		} else if c.Message() != nil {
			kind = "message"
			payload = c.Text()
			if len(payload) > 80 {
				payload = payload[:80] + "…"
			}
		} else {
			kind = "update"
		}

		err := next(c)

		status := "ok"
		if err != nil {
			status = "err: " + err.Error()
		}
		slog.Info("telegram update",
			"kind", kind,
			"user_id", sender.ID,
			"username", sender.Username,
			"payload", payload,
			"status", status,
			"duration", time.Since(start).Round(time.Millisecond).String(),
		)
		return err
	}
}
