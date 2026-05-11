package bot

import (
	"log/slog"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleOrders(c tele.Context) error {
	ctx := requestContext(c)
	orders, err := b.orderSvc.ListOrders(ctx, 10)
	if err != nil {
		slog.ErrorContext(ctx, "list orders failed", "error", err)
		return c.Send("Не удалось получить заказы. Попробуйте позже.")
	}
	if len(orders) == 0 {
		return c.Send("Заказов пока нет.")
	}

	return c.Send(responses.OrdersList(orders), tele.ModeHTML)
}
