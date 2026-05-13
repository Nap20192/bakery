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
		return sendText(c, "Не удалось получить заказы. Попробуйте позже.")
	}
	orders := result.Orders
	if len(orders) == 0 {
		return sendText(c, "Заказов пока нет.")
	}

	return sendHTML(c, responses.OrdersList(orders), b.actionMarkup(c))
}

func (b *OrderBot) handleOrder(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 1 {
		return sendText(c, "Формат: /order order_number")
	}

	ctx = applog.WithOrderNumber(ctx, args[0])
	order, err := b.orderSvc.GetOrderByNumber(ctx, args[0])
	if err != nil {
		slog.WarnContext(ctx, "order lookup failed", "error", err)
		return sendText(c, fmt.Sprintf("Заказ %s не найден.", args[0]))
	}

	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("Изменить", "edit_order", order.Number)))
	return sendHTML(c, responses.OrderSummary(order, fromName, toName), markup)
}
