package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"bakery/internal/pkg/enum"
	accessdomain "bakery/internal/services/auth/domain"
	monitoringdomain "bakery/internal/services/monitor/domain"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleOrders(c tele.Context) error {
	ctx := requestContext(c)
	orders, mode, err := b.listVisibleOrders(c, orderListLimit)
	if err != nil {
		if errors.Is(err, errOrderLocationRequired) {
			return sendText(c, msgUserNoShopDepartment, b.actionMarkup(c))
		}
		slog.ErrorContext(ctx, "list orders failed", "error", err)
		return sendText(c, "Не удалось получить заказы.")
	}
	if len(orders) == 0 {
		if mode == ordersModeShop {
			return sendText(c, "Заказов нет.", b.actionMarkup(c))
		}
		return sendText(c, "Ничего не найдено.", b.actionMarkup(c))
	}

	if mode == ordersModeShop {
		return sendText(c, "Последние заказы текущего магазина", b.ordersInlineMarkup(orders))
	}
	if err := sendText(c, "Фильтры заказов", b.actionMarkup(c)); err != nil {
		return err
	}
	return sendText(c, "Заказы после фильтрации", b.ordersInlineMarkup(orders))
}

func (b *OrderBot) listVisibleOrders(c tele.Context, limit int32) ([]orderdomain.Order, ordersMode, error) {
	user, ok := b.currentUser(c)
	if !ok {
		return nil, "", errOrderLocationRequired
	}

	filter := b.currentOrderFilter(c)
	mode := b.ordersMode(c, user)
	if mode == ordersModeShop {
		if user.DepartmentID == nil {
			return nil, mode, errOrderLocationRequired
		}
		filter.FromDepartmentID = cloneInt64Ptr(user.DepartmentID)
	}

	result, err := b.orderSvc.ListOrders(requestContext(c), orderuc.ListOrdersInput{
		Limit:            limit,
		FromDepartmentID: filter.FromDepartmentID,
		FulfillmentDate:  filter.FulfillmentDate,
	})
	if err != nil {
		return nil, mode, err
	}
	return result.Orders, mode, nil
}

var errOrderLocationRequired = fmt.Errorf("order location is required")

const orderListLimit int32 = 10

func (b *OrderBot) handleMonitorFilteredOrdersCallback(c tele.Context) error {
	ctx := requestContext(c)
	orders, _, err := b.listVisibleOrders(c, orderListLimit)
	if err != nil {
		if errors.Is(err, errOrderLocationRequired) {
			return sendText(c, msgUserNoShopDepartment, b.actionMarkup(c))
		}
		slog.ErrorContext(ctx, "list orders for filtered monitor failed", "error", err)
		return sendText(c, "Не удалось получить заказы.")
	}
	if len(orders) == 0 {
		return sendText(c, "Ничего не найдено.")
	}
	numbers := make([]string, 0, len(orders))
	for _, order := range orders {
		numbers = append(numbers, order.Number)
	}
	_ = c.Respond()
	return b.sendBatchMonitorReports(ctx, c, numbers)
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
	return b.sendMonitorReports(ctx, c, order, monitoringdomain.DefaultMonitorCodes)
}

func (b *OrderBot) handleCopyOrderCallback(c tele.Context) error {
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
	return sendHTML(c, responses.OrderCopy(order))
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
	return sendHTML(c, responses.OrderView(order, fromName, toName), b.orderActionsMarkup(c, order.Number))
}

func (b *OrderBot) ordersInlineMarkup(orders []orderdomain.Order) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(orders)+1)
	numbers := make([]string, 0, len(orders))
	for _, order := range orders {
		numbers = append(numbers, order.Number)
		button := markup.Data(orderListButtonText(order), "open_order", order.Number)
		if webAppButton, ok := b.miniAppButton(markup, orderListButtonText(order), miniAppModeView, order.Number, nil); ok {
			button = webAppButton
		}
		rows = append(rows, markup.Row(button))
	}
	monitorButton := markup.Data("Посчитать тесто по заказам", "monitor_filtered_orders")
	if webAppButton, ok := b.miniAppButton(markup, "Посчитать тесто по заказам", miniAppModeMonitor, "", numbers); ok {
		monitorButton = webAppButton
	}
	rows = append(rows, markup.Row(monitorButton))
	markup.Inline(rows...)
	return markup
}

func orderListButtonText(order orderdomain.Order) string {
	if order.FulfillmentDate.IsZero() {
		return order.Number
	}
	return fmt.Sprintf("%s / %s", order.Number, order.FulfillmentDate.Format("02.01.2006"))
}

func (b *OrderBot) handleOrdersReplyText(c tele.Context, text string) (bool, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false, nil
	}

	user, ok := b.currentUser(c)
	if !ok || b.ordersMode(c, user) != ordersModeWorkshop {
		return false, nil
	}
	switch text {
	case orderFilterTodayText:
		return true, b.setOrderReplyDateFilter(c, time.Now())
	case orderFilterTomorrowText:
		return true, b.setOrderReplyDateFilter(c, time.Now().AddDate(0, 0, 1))
	case orderFilterAllDatesText:
		return true, b.clearOrderReplyDateFilter(c)
	case orderFilterAllShopsText:
		return true, b.clearOrderReplyShopFilter(c)
	}

	shopID, ok := b.shopIDByReplyName(c, text)
	if !ok {
		return false, nil
	}
	return true, b.setOrderReplyShopFilter(c, shopID)
}

func (b *OrderBot) setOrderReplyDateFilter(c tele.Context, date time.Time) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FulfillmentDate = date
	})
	return b.handleOrders(c)
}

func (b *OrderBot) clearOrderReplyDateFilter(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FulfillmentDate = time.Time{}
	})
	return b.handleOrders(c)
}

func (b *OrderBot) setOrderReplyShopFilter(c tele.Context, shopID int64) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FromDepartmentID = &shopID
	})
	return b.handleOrders(c)
}

func (b *OrderBot) clearOrderReplyShopFilter(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FromDepartmentID = nil
	})
	return b.handleOrders(c)
}

func (b *OrderBot) shopIDByReplyName(c tele.Context, name string) (int64, bool) {
	shops, err := b.departmentSvc.ListByType(requestContext(c), enum.DepartmentTypeShop)
	if err != nil {
		slog.WarnContext(requestContext(c), "list shop departments for order reply filter failed", "error", err)
		return 0, false
	}
	for _, shop := range shops {
		if strings.EqualFold(strings.TrimSpace(shop.Name), strings.TrimSpace(name)) {
			return shop.ID, true
		}
	}
	return 0, false
}

type ordersMode string

const (
	ordersModeShop     ordersMode = "shop"
	ordersModeWorkshop ordersMode = "workshop"
)

func (b *OrderBot) ordersMode(c tele.Context, user accessdomain.AuthUser) ordersMode {
	if b.userDepartmentType(c, user) == string(enum.DepartmentTypeShop) {
		return ordersModeShop
	}
	return ordersModeWorkshop
}

func (b *OrderBot) currentOrderFilter(c tele.Context) orderFilter {
	if c.Sender() == nil {
		return orderFilter{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[c.Sender().ID]
	if s == nil {
		return orderFilter{}
	}
	return orderFilter{
		FromDepartmentID: cloneInt64Ptr(s.orderFilter.FromDepartmentID),
		FulfillmentDate:  s.orderFilter.FulfillmentDate,
	}
}

func (b *OrderBot) orderActionsMarkup(c tele.Context, orderNumber string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	user, ok := b.currentUser(c)
	if ok && b.ordersMode(c, user) == ordersModeShop {
		editButton := markup.Data("Изменить", "edit_order", orderNumber)
		if webAppButton, available := b.miniAppButton(markup, "Изменить", miniAppModeEdit, orderNumber, nil); available {
			editButton = webAppButton
		}
		markup.Inline(
			markup.Row(
				editButton,
				markup.Data("Копировать", "copy_order", orderNumber),
			),
		)
		return markup
	}
	monitorButton := markup.Data("Калькуляция", "monitor_order", orderNumber)
	if webAppButton, available := b.miniAppButton(markup, "Калькуляция", miniAppModeMonitor, orderNumber, nil); available {
		monitorButton = webAppButton
	}
	markup.Inline(
		markup.Row(
			monitorButton,
			markup.Data("Копировать", "copy_order", orderNumber),
		),
	)
	return markup
}
