package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"bakery/internal/pkg/enum"
	authuc "bakery/internal/services/auth/usecase/auth"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"
	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

const workshopDepartmentCode = "pekari"

func (b *OrderBot) handleCancel(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	b.clearSession(sender.ID)
	return sendText(c, "Текущий заказ отменен.", b.actionMarkup(c))
}

func (b *OrderBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if handled, err := b.handleActionText(c, text); handled {
		return err
	}
	if strings.HasPrefix(text, "/") {
		return sendText(c, "Неизвестная команда.\n\n/help - список команд и правила заказа")
	}
	if text == "" {
		return sendText(c, "Отправьте позиции заказа одним сообщением.")
	}
	return b.handleBulkOrder(c, text)
}

func (b *OrderBot) handleActionText(c tele.Context, text string) (bool, error) {
	if handled, err := b.handleOrdersReplyText(c, text); handled {
		return true, err
	}
	switch strings.TrimSpace(text) {
	case actionTemplates:
		return true, b.handleTemplates(c)
	case actionOrders:
		return true, b.handleOrders(c)
	case actionSubmitOrder:
		return true, b.handleConfirm(c)
	case actionUpdateOrder:
		return true, b.handleUpdateOrder(c)
	case actionCancelOrder:
		return true, b.handleCancel(c)
	case actionSync:
		if err := b.ensureActionPermission(c, enum.PermissionSync); err != nil {
			return true, err
		}
		return true, b.handleSync(c)
	default:
		return false, nil
	}
}

func (b *OrderBot) handleCurrentOrder(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	var current session
	b.mu.Lock()
	if s := b.sessions[sender.ID]; s != nil {
		current = *s
	}
	b.mu.Unlock()
	if len(current.items) == 0 {
		return sendText(c, "Текущий заказ пустой.", b.actionMarkup(c))
	}
	return sendHTML(c, responses.OrderDraft(current.editOrderNumber, current.items, current.fulfillmentDate, nil), b.actionMarkup(c))
}

func (b *OrderBot) ensureActionPermission(c tele.Context, permission enum.Permission) error {
	user, err := b.authUserFromContext(c)
	if err != nil {
		return err
	}
	if b.rbacSvc == nil || !b.rbacSvc.HasPermission(user.Role, permission) {
		return sendText(c, msgAccessDenied)
	}
	return nil
}

func (b *OrderBot) handleBulkOrder(c tele.Context, text string) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	result := b.orderSvc.ValidateBulkOrder(ctx, text)
	if len(result.ValidItems) == 0 && len(result.Errors) > 0 {
		return sendHTML(c, responses.ValidationErrors(result.Errors))
	}

	fromDepartmentID, toDepartmentID := b.sessionDepartments(sender.ID)
	if fromDepartmentID == nil && toDepartmentID == nil {
		fromDepartmentID, toDepartmentID = b.orderDepartmentsForSender(ctx, c)
	}
	if fromDepartmentID == nil && toDepartmentID == nil {
		return sendText(c, "Заказ создается от магазина в цех. Выберите магазин через /start.")
	}
	var current session
	b.updateSession(sender.ID, func(s *session) {
		s.items = mergeSessionItems(s.items, result.ValidItems)
		s.fromDepartmentID = fromDepartmentID
		s.toDepartmentID = toDepartmentID
		if hasFulfillmentDateLine(text) || s.fulfillmentDate.IsZero() {
			s.fulfillmentDate = result.FulfillmentDate
		}
		current = *s
	})

	return sendHTML(c, responses.OrderDraft(current.editOrderNumber, current.items, current.fulfillmentDate, result.Errors), b.actionMarkup(c))
}

func hasFulfillmentDateLine(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if _, ok, _ := orderdomain.ParseFulfillmentDateLine(strings.TrimSpace(line)); ok {
			return true
		}
	}
	return false
}

func (b *OrderBot) sessionDepartments(uid int64) (*int64, *int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[uid]
	if s == nil {
		return nil, nil
	}
	return cloneInt64Ptr(s.fromDepartmentID), cloneInt64Ptr(s.toDepartmentID)
}

func (b *OrderBot) handleConfirm(c tele.Context) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	var items []orderdomain.OrderItem
	var fromDepartmentID *int64
	var toDepartmentID *int64
	var fulfillmentDate time.Time
	b.updateSession(sender.ID, func(s *session) {
		items = make([]orderdomain.OrderItem, len(s.items))
		copy(items, s.items)
		fromDepartmentID = cloneInt64Ptr(s.fromDepartmentID)
		toDepartmentID = cloneInt64Ptr(s.toDepartmentID)
		fulfillmentDate = s.fulfillmentDate
		s.items = nil
		s.fulfillmentDate = time.Time{}
		s.editOrderNumber = ""
	})

	_ = c.Respond()
	if len(items) == 0 {
		return sendText(c, "Заказ пустой или уже отправлен.")
	}
	if fromDepartmentID == nil && toDepartmentID == nil {
		fromDepartmentID, toDepartmentID = b.orderDepartmentsForSender(ctx, c)
	}

	order, err := b.orderSvc.CreateOrder(ctx, orderdomain.CreateOrderInput{
		Items:             items,
		FromDepartmentID:  fromDepartmentID,
		ToDepartmentID:    toDepartmentID,
		CreatedByUsername: b.createdByUsername(ctx, sender),
		FulfillmentDate:   fulfillmentDate,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create order failed", "error", err)
		return sendText(c, "Не удалось создать заказ. Проверьте заказ и попробуйте снова.")
	}

	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	summary := responses.OrderSummary(order, fromName, toName)

	slog.InfoContext(applog.WithOrderNumber(ctx, order.Number), "order created", "items", len(items))
	return sendHTML(c, summary, b.actionMarkup(c))
}

func (b *OrderBot) createdByUsername(ctx context.Context, sender *tele.User) string {
	if sender == nil {
		return ""
	}
	if username := strings.TrimSpace(sender.Username); username != "" {
		return username
	}
	user, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		return ""
	}
	if user.TelegramUsername != nil && strings.TrimSpace(*user.TelegramUsername) != "" {
		return strings.TrimSpace(*user.TelegramUsername)
	}
	return strings.TrimSpace(user.Username)
}

func (b *OrderBot) handleEditOrder(c tele.Context) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	number := strings.TrimSpace(c.Callback().Data)
	if number == "" {
		return sendText(c, "Не удалось определить заказ.")
	}
	order, err := b.orderSvc.GetOrderByNumber(ctx, number)
	if err != nil {
		slog.WarnContext(ctx, "get order for edit failed", "order_number", number, "error", err)
		return sendText(c, "Заказ не найден.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.items = append([]orderdomain.OrderItem(nil), order.Items...)
		s.fromDepartmentID = cloneInt64Ptr(order.FromDepartmentID)
		s.toDepartmentID = cloneInt64Ptr(order.ToDepartmentID)
		s.fulfillmentDate = order.FulfillmentDate
		s.editOrderNumber = order.Number
	})
	_ = c.Respond()
	return sendHTML(c, responses.OrderDraft(order.Number, order.Items, order.FulfillmentDate, nil), b.actionMarkup(c))
}

func (b *OrderBot) handleUpdateOrder(c tele.Context) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, msgTelegramUserUnknown)
	}
	var items []orderdomain.OrderItem
	var fromDepartmentID *int64
	var toDepartmentID *int64
	var fulfillmentDate time.Time
	var orderNumber string
	b.updateSession(sender.ID, func(s *session) {
		items = append([]orderdomain.OrderItem(nil), s.items...)
		fromDepartmentID = cloneInt64Ptr(s.fromDepartmentID)
		toDepartmentID = cloneInt64Ptr(s.toDepartmentID)
		fulfillmentDate = s.fulfillmentDate
		orderNumber = s.editOrderNumber
		s.items = nil
		s.fulfillmentDate = time.Time{}
		s.editOrderNumber = ""
	})
	_ = c.Respond()
	if orderNumber == "" {
		return sendText(c, "Нет заказа в режиме редактирования.")
	}
	if len(items) == 0 {
		return sendText(c, "Заказ пустой. Добавьте позиции или нажмите отмену.")
	}
	order, err := b.orderSvc.UpdateOrder(ctx, orderuc.UpdateOrderInput{
		Number:            orderNumber,
		Items:             items,
		FromDepartmentID:  fromDepartmentID,
		ToDepartmentID:    toDepartmentID,
		CreatedByUsername: b.createdByUsername(ctx, sender),
		FulfillmentDate:   fulfillmentDate,
	})
	if err != nil {
		slog.ErrorContext(ctx, "update order failed", "order_number", orderNumber, "error", err)
		return sendText(c, "Не удалось обновить заказ.")
	}
	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	summary := responses.OrderUpdated(order, fromName, toName)
	return sendHTML(c, summary, b.actionMarkup(c))
}

func (b *OrderBot) orderDepartmentsForSender(ctx context.Context, c tele.Context) (*int64, *int64) {
	sender := c.Sender()
	if sender == nil {
		return nil, nil
	}
	user, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		if !errors.Is(err, authuc.ErrAuthUserNotFound) {
			slog.WarnContext(ctx, "get user department failed", "error", err)
		}
		return nil, nil
	}
	if user.DepartmentID == nil {
		return nil, nil
	}
	fromID, toID, err := b.orderDepartmentsForSelected(ctx, *user.DepartmentID)
	if err != nil {
		slog.WarnContext(ctx, "resolve selected department failed", "department_id", *user.DepartmentID, "error", err)
		return nil, nil
	}
	return fromID, toID
}

func (b *OrderBot) orderDepartmentsForSelected(ctx context.Context, departmentID int64) (*int64, *int64, error) {
	department, err := b.departmentSvc.GetByID(ctx, departmentID)
	if err != nil {
		return nil, nil, err
	}
	workshop, err := b.departmentSvc.GetByCode(ctx, workshopDepartmentCode)
	if err != nil {
		return nil, nil, err
	}

	switch department.Type {
	case string(enum.DepartmentTypeShop):
		fromID := department.ID
		toID := workshop.ID
		return &fromID, &toID, nil
	case string(enum.DepartmentTypeWorkshop):
		return nil, nil, nil
	default:
		return nil, nil, nil
	}
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (b *OrderBot) departmentDisplayName(ctx context.Context, departmentID *int64) string {
	if departmentID == nil {
		return ""
	}
	department, err := b.departmentSvc.GetByID(ctx, *departmentID)
	if err != nil {
		slog.WarnContext(ctx, "get department display name failed", "department_id", *departmentID, "error", err)
		return fmt.Sprintf("#%d", *departmentID)
	}
	return department.Name
}

// notifyOrder delivers an order notification to the order's creator, every
// baker, and the workshop group chat. Best-effort: per-recipient failures are
// logged, not propagated (a failed send must not requeue the event).
func (b *OrderBot) notifyOrder(ctx context.Context, order orderdomain.Order, message string) {
	recipients := make(map[int64]struct{})

	if username := strings.TrimSpace(order.CreatedByUsername); username != "" {
		creator, err := b.authSvc.GetUserByTelegramUsername(ctx, username)
		switch {
		case err != nil && !errors.Is(err, authuc.ErrAuthUserNotFound):
			slog.WarnContext(ctx, "resolve order creator failed", "username", username, "error", err)
		case err != nil:
			slog.DebugContext(ctx, "order creator not registered, skipping DM", "component", "bot.notify", "username", username)
		case creator.TelegramID != nil:
			recipients[*creator.TelegramID] = struct{}{}
			slog.DebugContext(ctx, "notify recipient: creator", "component", "bot.notify", "username", username, "telegram_id", *creator.TelegramID)
		}
	}

	bakers, err := b.authSvc.ListUsersByRole(ctx, string(enum.RoleBaker))
	if err != nil {
		slog.WarnContext(ctx, "list bakers for order notification failed", "error", err)
	}
	bakerCount := 0
	for _, baker := range bakers {
		if baker.TelegramID != nil {
			recipients[*baker.TelegramID] = struct{}{}
			bakerCount++
			slog.DebugContext(ctx, "notify recipient: baker", "component", "bot.notify", "telegram_id", *baker.TelegramID)
		}
	}

	slog.DebugContext(ctx, "notifying order",
		"component", "bot.notify",
		"order_number", order.Number,
		"dm_recipients", len(recipients),
		"bakers", bakerCount,
		"group_chat_id", b.workshopChatID,
	)

	for telegramID := range recipients {
		if err := b.sendHTMLToChat(telegramID, message); err != nil {
			slog.WarnContext(ctx, "send order notification failed", "telegram_id", telegramID, "error", err)
			continue
		}
		slog.DebugContext(ctx, "order notification sent (dm)", "component", "bot.notify", "telegram_id", telegramID, "order_number", order.Number)
	}

	if b.workshopChatID != 0 {
		if err := b.sendHTMLToChat(b.workshopChatID, message); err != nil {
			slog.WarnContext(ctx, "send order notification to group failed", "chat_id", b.workshopChatID, "error", err)
			return
		}
		slog.DebugContext(ctx, "order notification sent (group)", "component", "bot.notify", "chat_id", b.workshopChatID, "order_number", order.Number)
	}
}
