package bot

import (
	"fmt"
	"log/slog"
	"strings"

	"bakery/internal/app"
	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleOrders(c tele.Context) error {
	ctx := requestContext(c)
	result, err := b.orderSvc.ListOrders(ctx, app.ListOrdersInput{Limit: 10})
	if err != nil {
		slog.ErrorContext(ctx, "list orders failed", "error", err)
		return c.Send("Не удалось получить заказы. Попробуйте позже.")
	}
	orders := result.Orders
	if len(orders) == 0 {
		return c.Send("Заказов пока нет.")
	}

	return c.Send(responses.OrdersList(orders), tele.ModeHTML)
}

func (b *OrderBot) handleOrder(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 1 {
		return c.Send("Формат: /order order_number")
	}

	ctx = applog.WithOrderNumber(ctx, args[0])
	order, err := b.orderSvc.GetOrderByNumber(ctx, args[0])
	if err != nil {
		slog.WarnContext(ctx, "order lookup failed", "error", err)
		return c.Send(fmt.Sprintf("Заказ %s не найден.", args[0]))
	}

	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	return c.Send(responses.OrderSummary(order, fromName, toName), tele.ModeHTML)
}
