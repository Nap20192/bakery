package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"bakery/internal/app"
	accessdomain "bakery/internal/domain/access"
	orderdomain "bakery/internal/domain/order"
	"bakery/internal/pkg/enum"
	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleOrders(c tele.Context) error {
	ctx := requestContext(c)
	orders, mode, err := b.listVisibleOrders(c, orderListLimit)
	if err != nil {
		if err == errOrderLocationRequired {
			return sendText(c, "Сначала выберите локацию через /start.", b.actionMarkup(c))
		}
		slog.ErrorContext(ctx, "list orders failed", "error", err)
		return sendText(c, "Не удалось получить заказы. Попробуйте позже.")
	}
	if len(orders) == 0 {
		if mode == ordersModeShop {
			return sendText(c, "Ваших последних заказов пока нет.", b.actionMarkup(c))
		}
		return sendText(c, "Заказов по выбранным фильтрам нет.", b.ordersFilterReplyMarkup(ctx))
	}

	if mode == ordersModeShop {
		return sendText(c, "Последние заказы текущего магазина", b.ordersInlineMarkup(orders))
	}
	if err := sendText(c, "Фильтры заказов", b.ordersFilterReplyMarkup(ctx)); err != nil {
		return err
	}
	return sendText(c, "Заказы после фильтрации", b.ordersInlineMarkup(orders))
}

func (b *OrderBot) listVisibleOrders(c tele.Context, limit int32) ([]orderdomain.Order, ordersMode, error) {
	user, ok := b.currentUser(c)
	if !ok || user.DepartmentID == nil {
		return nil, "", errOrderLocationRequired
	}

	filter := b.currentOrderFilter(c)
	mode := b.ordersMode(c, user)
	if mode == ordersModeShop {
		filter.FromDepartmentID = cloneInt64Ptr(user.DepartmentID)
	}

	result, err := b.orderSvc.ListOrders(requestContext(c), app.ListOrdersInput{
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
		if err == errOrderLocationRequired {
			return sendText(c, "Сначала выберите локацию через /start.", b.actionMarkup(c))
		}
		slog.ErrorContext(ctx, "list orders for filtered monitor failed", "error", err)
		return sendText(c, "Не удалось получить заказы для расчёта.")
	}
	if len(orders) == 0 {
		return sendText(c, "Заказов по выбранным фильтрам нет.")
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
	return b.sendMonitorReports(ctx, c, order, defaultMonitorCodes)
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
	for _, order := range orders {
		rows = append(rows, markup.Row(markup.Data(orderListButtonText(order), "open_order", order.Number)))
	}
	rows = append(rows, markup.Row(markup.Data("Посчитать тесто по заказам", "monitor_filtered_orders")))
	markup.Inline(rows...)
	return markup
}

func orderListButtonText(order orderdomain.Order) string {
	if order.FulfillmentDate.IsZero() {
		return order.Number
	}
	return fmt.Sprintf("%s / %s", order.Number, order.FulfillmentDate.Format("02.01.2006"))
}

const (
	orderFilterTodayText     = "Сегодня"
	orderFilterTomorrowText  = "Завтра"
	orderFilterAllDatesText  = "Все даты"
	orderFilterAllShopsText  = "Все магазины"
	orderReplyShopButtonSize = 2
)

func (b *OrderBot) ordersFilterReplyMarkup(ctx context.Context) *tele.ReplyMarkup {
	rows := make([][]string, 0, 4)
	rows = append(rows, []string{orderFilterTodayText, orderFilterTomorrowText, orderFilterAllDatesText})
	rows = append(rows, b.orderShopFilterReplyRows(ctx)...)
	return replyKeyboard(rows...)
}

func (b *OrderBot) orderShopFilterReplyRows(ctx context.Context) [][]string {
	shops, err := b.departmentSvc.ListByType(ctx, enum.DepartmentTypeShop)
	if err != nil {
		slog.WarnContext(ctx, "list shop departments for order filters failed", "error", err)
		return nil
	}
	buttons := make([]string, 0, len(shops)+1)
	buttons = append(buttons, orderFilterAllShopsText)
	for _, shop := range shops {
		buttons = append(buttons, shop.Name)
	}
	rows := make([][]string, 0, (len(buttons)+1)/orderReplyShopButtonSize)
	for len(buttons) > 0 {
		end := orderReplyShopButtonSize
		if len(buttons) < end {
			end = len(buttons)
		}
		rows = append(rows, buttons[:end])
		buttons = buttons[end:]
	}
	return rows
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
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FulfillmentDate = date
		s.monitorOrders = nil
	})
	return b.handleOrders(c)
}

func (b *OrderBot) clearOrderReplyDateFilter(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FulfillmentDate = time.Time{}
		s.monitorOrders = nil
	})
	return b.handleOrders(c)
}

func (b *OrderBot) setOrderReplyShopFilter(c tele.Context, shopID int64) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FromDepartmentID = &shopID
		s.monitorOrders = nil
	})
	return b.handleOrders(c)
}

func (b *OrderBot) clearOrderReplyShopFilter(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FromDepartmentID = nil
		s.monitorOrders = nil
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

func (b *OrderBot) ordersFilterMarkup(ctx context.Context, orders []orderdomain.Order) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := orderFilterRows(ctx, b, markup)
	for _, order := range orders {
		rows = append(rows, markup.Row(markup.Data(order.Number, "open_order", order.Number)))
	}
	markup.Inline(rows...)
	return markup
}

type ordersMode string

const (
	ordersModeShop     ordersMode = "shop"
	ordersModeWorkshop ordersMode = "workshop"
)

func (b *OrderBot) ordersMode(c tele.Context, user accessdomain.AuthUser) ordersMode {
	if b.userDepartmentType(c, user) == string(app.DepartmentTypeShop) {
		return ordersModeShop
	}
	return ordersModeWorkshop
}

func (b *OrderBot) handleSelectMonitorOrderCallback(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	number := strings.TrimSpace(c.Callback().Data)
	if number == "" {
		return sendText(c, "Не удалось определить заказ.")
	}
	b.updateSession(sender.ID, func(s *session) {
		if containsString(s.monitorOrders, number) {
			s.monitorOrders = removeString(s.monitorOrders, number)
			return
		}
		s.monitorOrders = append(s.monitorOrders, number)
	})
	_ = c.Respond()
	return b.handleOrders(c)
}

func (b *OrderBot) handleMonitorSelectedOrdersCallback(c tele.Context) error {
	ctx := requestContext(c)
	selected := b.currentMonitorOrders(c)
	if len(selected) == 0 {
		return sendText(c, "Выберите заказы в списке /orders.")
	}
	_ = c.Respond()
	return b.sendBatchMonitorReports(ctx, c, selected)
}

func (b *OrderBot) handleClearMonitorOrdersCallback(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.monitorOrders = nil
	})
	_ = c.Respond()
	return b.handleOrders(c)
}

func orderFilterRows(ctx context.Context, b *OrderBot, markup *tele.ReplyMarkup) []tele.Row {
	rows := []tele.Row{
		markup.Row(
			markup.Data("Сегодня", "order_filter_date", time.Now().Format("2006-01-02")),
			markup.Data("Завтра", "order_filter_date", time.Now().AddDate(0, 0, 1).Format("2006-01-02")),
			markup.Data("Все даты", "order_filter_all_dates"),
		),
	}

	shops, err := b.departmentSvc.ListByType(ctx, enum.DepartmentTypeShop)
	if err != nil {
		slog.WarnContext(ctx, "list shop departments for order filters failed", "error", err)
		return rows
	}
	shopButtons := make([]tele.Btn, 0, len(shops)+1)
	shopButtons = append(shopButtons, markup.Data("Все магазины", "order_filter_all_shops"))
	for _, shop := range shops {
		shopButtons = append(shopButtons, markup.Data(shop.Name, "order_filter_shop", strconv.FormatInt(shop.ID, 10)))
	}
	for len(shopButtons) > 0 {
		end := 2
		if len(shopButtons) < end {
			end = len(shopButtons)
		}
		rows = append(rows, markup.Row(shopButtons[:end]...))
		shopButtons = shopButtons[end:]
	}
	return rows
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

func (b *OrderBot) handleOrderFilterShop(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Callback().Data), 10, 64)
	if err != nil || id <= 0 {
		return sendText(c, "Не удалось определить магазин.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FromDepartmentID = &id
		s.monitorOrders = nil
	})
	_ = c.Respond()
	return b.handleOrders(c)
}

func (b *OrderBot) handleOrderFilterDate(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(c.Callback().Data))
	if err != nil {
		return sendText(c, "Не удалось определить дату.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FulfillmentDate = date
		s.monitorOrders = nil
	})
	_ = c.Respond()
	return b.handleOrders(c)
}

func (b *OrderBot) handleOrderFilterAllShops(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FromDepartmentID = nil
		s.monitorOrders = nil
	})
	_ = c.Respond()
	return b.handleOrders(c)
}

func (b *OrderBot) handleOrderFilterAllDates(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.orderFilter.FulfillmentDate = time.Time{}
		s.monitorOrders = nil
	})
	_ = c.Respond()
	return b.handleOrders(c)
}

func (b *OrderBot) currentMonitorOrders(c tele.Context) []string {
	sender := c.Sender()
	if sender == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[sender.ID]
	if s == nil || len(s.monitorOrders) == 0 {
		return nil
	}
	return append([]string(nil), s.monitorOrders...)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func removeString(values []string, target string) []string {
	result := values[:0]
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
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

func (b *OrderBot) orderActionsMarkup(c tele.Context, orderNumber string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	user, ok := b.currentUser(c)
	if ok && b.ordersMode(c, user) == ordersModeShop {
		markup.Inline(
			markup.Row(
				markup.Data("Изменить", "edit_order", orderNumber),
				markup.Data("Копировать", "copy_order", orderNumber),
			),
		)
		return markup
	}
	markup.Inline(
		markup.Row(
			markup.Data("Калькуляция", "monitor_order", orderNumber),
			markup.Data("Копировать", "copy_order", orderNumber),
		),
	)
	return markup
}
