package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"bakery/internal/app"
	orderdomain "bakery/internal/domain/order"
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

	departments := b.orderListDepartments(ctx, orders)
	return sendHTML(c, responses.OrdersList(orders, departments), b.ordersListMarkup(orders))
}

func (b *OrderBot) handleOrder(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 1 {
		return sendText(c, "Формат: /order order_number")
	}

	return b.sendOrderByNumber(ctx, c, args[0])
}

func (b *OrderBot) handleOpenOrderCallback(c tele.Context) error {
	ctx := requestContext(c)
	number := strings.TrimSpace(c.Callback().Data)
	if number == "" {
		return sendText(c, "Не удалось определить заказ.")
	}
	_ = c.Respond()
	return b.sendOrderByNumber(ctx, c, number)
}

func (b *OrderBot) handleMonitorOrderCallback(c tele.Context) error {
	ctx := requestContext(c)
	number := strings.TrimSpace(c.Callback().Data)
	if number == "" {
		return sendText(c, "Не удалось определить заказ.")
	}
	ctx = applog.WithOrderNumber(ctx, number)
	order, err := b.orderSvc.GetOrderByNumber(ctx, number)
	if err != nil {
		slog.WarnContext(ctx, "order lookup failed", "error", err)
		return sendText(c, fmt.Sprintf("Заказ %s не найден.", number))
	}
	_ = c.Respond()
	return b.sendMonitorReports(ctx, c, order, defaultMonitorCodes)
}

func (b *OrderBot) sendOrderByNumber(ctx context.Context, c tele.Context, number string) error {
	ctx = applog.WithOrderNumber(ctx, number)
	order, err := b.orderSvc.GetOrderByNumber(ctx, number)
	if err != nil {
		slog.WarnContext(ctx, "order lookup failed", "error", err)
		return sendText(c, fmt.Sprintf("Заказ %s не найден.", number))
	}

	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	return sendHTML(c, responses.OrderView(order, fromName, toName), b.orderActionsMarkup(order.Number))
}

func (b *OrderBot) ordersListMarkup(orders []orderdomain.Order) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(orders))
	for _, order := range orders {
		rows = append(rows, markup.Row(markup.Data(order.Number, "open_order", order.Number)))
	}
	markup.Inline(rows...)
	return markup
}

func (b *OrderBot) orderListDepartments(ctx context.Context, orders []orderdomain.Order) map[int64]string {
	departments := make(map[int64]string)
	for _, order := range orders {
		b.addOrderDepartmentName(ctx, departments, order.FromDepartmentID)
		b.addOrderDepartmentName(ctx, departments, order.ToDepartmentID)
	}
	return departments
}

func (b *OrderBot) addOrderDepartmentName(ctx context.Context, departments map[int64]string, id *int64) {
	if id == nil {
		return
	}
	if _, ok := departments[*id]; ok {
		return
	}
	departments[*id] = b.departmentDisplayName(ctx, id)
}

func (b *OrderBot) orderActionsMarkup(orderNumber string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Изменить", "edit_order", orderNumber),
			markup.Data("Калькуляция", "monitor_order", orderNumber),
		),
	)
	return markup
}
