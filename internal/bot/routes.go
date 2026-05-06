package bot

import (
	"log"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) register() {
	bt := b.tele
	bt.Use(b.logMiddleware)

	bt.Handle("/start", b.handleStart)
	bt.Handle("/template", b.handleTemplate, b.requirePermissions(permCreateOrder))
	bt.Handle("/cancel", b.handleCancel, b.requirePermissions(permCreateOrder))

	bt.Handle("\fconfirm", b.handleConfirm, b.requirePermissions(permCreateOrder))
	bt.Handle("\fcancel_cb", b.handleCancelCallback, b.requirePermissions(permCreateOrder))

	bt.Handle("/orders", b.handleOrders, b.requirePermissions(permViewOrders))
	bt.Handle("/accept", b.handleAcceptOrder, b.requirePermissions(permAcceptOrder))
	bt.Handle("/delete", b.handleDeleteOrder, b.requirePermissions(permDeleteOrder))
	bt.Handle("/close", b.handleCloseOrder, b.requirePermissions(permCloseOrder))
	bt.Handle("/reports", b.handleReports, b.requirePermissions(permViewReports))
	bt.Handle("/groups_add", b.handleAddGroup, b.requirePermissions(permManageGroups))
	bt.Handle("/users_add", b.handleAddUser, b.requirePermissions(permManageUsers))
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
		log.Printf("[%s] uid=%d @%s %q → %s (%s)",
			kind, sender.ID, sender.Username, payload, status,
			time.Since(start).Round(time.Millisecond),
		)
		return err
	}
}
