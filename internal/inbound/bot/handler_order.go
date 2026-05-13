package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"bakery/internal/app"
	orderdomain "bakery/internal/domain/order"
	"bakery/internal/pkg/enum"
	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

const workshopDepartmentCode = "pekari"

func (b *OrderBot) handleTemplate(c tele.Context) error {
	ctx := requestContext(c)
	template, err := b.orderSvc.GetTemplate(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get order template failed", "error", err)
		return sendText(c, "Не удалось получить шаблон заказа.")
	}
	return sendHTML(c, responses.Template(template))
}

func (b *OrderBot) handleCancel(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.clearSession(sender.ID)
	return sendText(c, "Текущий заказ отменен.", b.actionMarkup(c))
}

func (b *OrderBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	sender := c.Sender()
	if sender != nil && b.isWaitingTemplate(sender.ID) {
		return b.createTemplateFromMessage(c, text)
	}
	if handled, err := b.handleActionText(c, text); handled {
		return err
	}
	if sender != nil && b.isWaitingDelete(sender.ID) {
		return b.deletePositionFromMessage(c, text)
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
	switch strings.TrimSpace(text) {
	case actionChooseShop:
		return true, b.handleDepartmentShop(c)
	case actionChooseWorkshop:
		return true, b.handleDepartmentWorkshop(c)
	case actionTemplates:
		return true, b.handleTemplates(c)
	case actionOrders:
		return true, b.handleOrders(c)
	case actionSubmitOrder:
		return true, b.handleConfirm(c)
	case actionUpdateOrder:
		return true, b.handleUpdateOrder(c)
	case actionDeletePosition:
		return true, b.handleDeletePosition(c)
	case actionCancelOrder:
		return true, b.handleCancel(c)
	case actionAddTemplate:
		if err := b.ensureActionPermission(c, app.PermissionTemplateManage); err != nil {
			return true, err
		}
		return true, b.handleAddTemplate(c)
	case actionSync:
		if err := b.ensureActionPermission(c, app.PermissionSync); err != nil {
			return true, err
		}
		return true, b.handleSync(c)
	case actionHelp:
		return true, b.handleStart(c)
	default:
		return false, nil
	}
}

func (b *OrderBot) handleDeletePosition(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	var hasItems bool
	b.updateSession(sender.ID, func(s *session) {
		hasItems = len(s.items) > 0
		s.waitingDelete = hasItems
	})
	if !hasItems {
		return sendText(c, "В текущем заказе нет позиций для удаления.", b.actionMarkup(c))
	}
	return sendText(c, "Отправьте код позиции, которую нужно удалить.\nНапример: 15647", b.actionMarkup(c))
}

func (b *OrderBot) deletePositionFromMessage(c tele.Context, text string) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	code := firstField(text)
	if code == "" {
		return sendText(c, "Отправьте код позиции, например: 15647", b.actionMarkup(c))
	}

	var current session
	var removed bool
	b.updateSession(sender.ID, func(s *session) {
		s.items, removed = removeSessionItemByCode(s.items, code)
		s.waitingDelete = false
		current = *s
	})
	if !removed {
		return sendText(c, fmt.Sprintf("Позиция с кодом %s не найдена в текущем заказе.", code), b.actionMarkup(c))
	}
	return sendHTML(c, responses.OrderDraft(current.editOrderNumber, current.items, current.fulfillmentDate, nil), b.actionMarkup(c))
}

func firstField(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func removeSessionItemByCode(items []orderdomain.OrderItem, code string) ([]orderdomain.OrderItem, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return items, false
	}
	for i, item := range items {
		if item.Code != code {
			continue
		}
		result := make([]orderdomain.OrderItem, 0, len(items)-1)
		result = append(result, items[:i]...)
		result = append(result, items[i+1:]...)
		return result, true
	}
	return items, false
}

func (b *OrderBot) ensureActionPermission(c tele.Context, permission enum.Permission) error {
	user, err := b.authUserFromContext(c)
	if err != nil {
		return err
	}
	if b.rbacSvc == nil || !b.rbacSvc.HasPermission(user.Role, permission) {
		return sendText(c, "Доступ запрещён.")
	}
	return nil
}

func (b *OrderBot) handleBulkOrder(c tele.Context, text string) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
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
		return sendText(c, "Не удалось определить пользователя.")
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
	if err := b.notifyWorkshop(ctx, sender.ID, summary); err != nil {
		slog.WarnContext(ctx, "notify workshop about order failed", "order_number", order.Number, "error", err)
	}

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

func (b *OrderBot) handleCancelCallback(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.clearSession(sender.ID)
	_ = c.Respond()
	return sendText(c, "Текущий заказ отменен.", b.actionMarkup(c))
}

func (b *OrderBot) handleEditOrder(c tele.Context) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
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
		return sendText(c, "Не удалось определить пользователя.")
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
	order, err := b.orderSvc.UpdateOrder(ctx, app.UpdateOrderInput{
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
	if err := b.notifyWorkshop(ctx, sender.ID, summary); err != nil {
		slog.WarnContext(ctx, "notify workshop about order update failed", "order_number", order.Number, "error", err)
	}
	return sendHTML(c, summary, b.actionMarkup(c))
}

func (b *OrderBot) orderDepartmentsForSender(ctx context.Context, c tele.Context) (*int64, *int64) {
	sender := c.Sender()
	if sender == nil {
		return nil, nil
	}
	user, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		if !errors.Is(err, app.ErrAuthUserNotFound) {
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
	case string(app.DepartmentTypeShop):
		fromID := department.ID
		toID := workshop.ID
		return &fromID, &toID, nil
	case string(app.DepartmentTypeWorkshop):
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

func (b *OrderBot) notifyWorkshop(ctx context.Context, senderID int64, message string) error {
	workshop, err := b.departmentSvc.GetByCode(ctx, workshopDepartmentCode)
	if err != nil {
		return fmt.Errorf("get workshop department: %w", err)
	}
	users, err := b.authSvc.ListUsersByDepartmentID(ctx, workshop.ID)
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.TelegramID == nil || *user.TelegramID == senderID {
			continue
		}
		if err := b.sendHTMLToChat(*user.TelegramID, message); err != nil {
			slog.WarnContext(ctx, "send workshop order notification failed", "telegram_id", *user.TelegramID, "error", err)
		}
	}
	return nil
}
