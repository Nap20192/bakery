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
	b.clearSession(c.Sender().ID)
	return sendText(c, "Текущий заказ отменен.")
}

func (b *OrderBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if c.Sender() != nil && b.isWaitingTemplate(c.Sender().ID) {
		return b.createTemplateFromMessage(c, text)
	}
	if strings.HasPrefix(text, "/") {
		return sendText(c, "Неизвестная команда.\n\n/help - список команд и правила заказа")
	}
	if text == "" {
		return sendText(c, "Отправьте позиции заказа одним сообщением.")
	}
	return b.handleBulkOrder(c, text)
}

func (b *OrderBot) handleBulkOrder(c tele.Context, text string) error {
	ctx := requestContext(c)
	result := b.orderSvc.ValidateBulkOrder(ctx, text)
	if len(result.ValidItems) == 0 && len(result.Errors) > 0 {
		return sendHTML(c, responses.ValidationErrors(result.Errors))
	}

	fromDepartmentID, toDepartmentID := b.orderDepartmentsForSender(ctx, c)
	if fromDepartmentID == nil && toDepartmentID == nil {
		return sendText(c, "Заказ создается от магазина в цех. Выберите магазин через /start.")
	}
	var current session
	b.updateSession(c.Sender().ID, func(s *session) {
		s.items = mergeSessionItems(s.items, result.ValidItems)
		s.fromDepartmentID = fromDepartmentID
		s.toDepartmentID = toDepartmentID
		if hasFulfillmentDateLine(text) || s.fulfillmentDate.IsZero() {
			s.fulfillmentDate = result.FulfillmentDate
		}
		current = *s
	})

	return sendHTML(c, responses.OrderDraft(current.editOrderNumber, current.items, current.fulfillmentDate, result.Errors), b.orderDraftMarkup(current.editOrderNumber))
}

func hasFulfillmentDateLine(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if _, ok, _ := orderdomain.ParseFulfillmentDateLine(strings.TrimSpace(line)); ok {
			return true
		}
	}
	return false
}

func (b *OrderBot) handleConfirm(c tele.Context) error {
	ctx := requestContext(c)
	var items []orderdomain.OrderItem
	var fromDepartmentID *int64
	var toDepartmentID *int64
	var fulfillmentDate time.Time
	b.updateSession(c.Sender().ID, func(s *session) {
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
		CreatedByUsername: b.createdByUsername(ctx, c),
		FulfillmentDate:   fulfillmentDate,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create order failed", "error", err)
		return sendText(c, "Не удалось создать заказ. Проверьте заказ и попробуйте снова.")
	}

	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	summary := responses.OrderSummary(order, fromName, toName)
	if err := b.notifyWorkshop(ctx, c.Sender().ID, summary); err != nil {
		slog.WarnContext(ctx, "notify workshop about order failed", "order_number", order.Number, "error", err)
	}

	slog.InfoContext(applog.WithOrderNumber(ctx, order.Number), "order created", "items", len(items))
	return sendHTML(c, summary)
}

func (b *OrderBot) createdByUsername(ctx context.Context, c tele.Context) string {
	if c == nil || c.Sender() == nil {
		return ""
	}
	if username := strings.TrimSpace(c.Sender().Username); username != "" {
		return username
	}
	user, err := b.authSvc.GetUserByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		return ""
	}
	if user.TelegramUsername != nil && strings.TrimSpace(*user.TelegramUsername) != "" {
		return strings.TrimSpace(*user.TelegramUsername)
	}
	return strings.TrimSpace(user.Username)
}

func (b *OrderBot) handleCancelCallback(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	_ = c.Respond()
	return sendText(c, "Текущий заказ отменен.")
}

func (b *OrderBot) handleEditOrder(c tele.Context) error {
	ctx := requestContext(c)
	number := strings.TrimSpace(c.Callback().Data)
	if number == "" {
		return sendText(c, "Не удалось определить заказ.")
	}
	order, err := b.orderSvc.GetOrderByNumber(ctx, number)
	if err != nil {
		slog.WarnContext(ctx, "get order for edit failed", "order_number", number, "error", err)
		return sendText(c, "Заказ не найден.")
	}
	b.updateSession(c.Sender().ID, func(s *session) {
		s.items = append([]orderdomain.OrderItem(nil), order.Items...)
		s.fromDepartmentID = cloneInt64Ptr(order.FromDepartmentID)
		s.toDepartmentID = cloneInt64Ptr(order.ToDepartmentID)
		s.fulfillmentDate = order.FulfillmentDate
		s.editOrderNumber = order.Number
	})
	_ = c.Respond()
	return sendHTML(c, responses.OrderDraft(order.Number, order.Items, order.FulfillmentDate, nil), b.orderDraftMarkup(order.Number))
}

func (b *OrderBot) handleUpdateOrder(c tele.Context) error {
	ctx := requestContext(c)
	var items []orderdomain.OrderItem
	var fromDepartmentID *int64
	var toDepartmentID *int64
	var fulfillmentDate time.Time
	var orderNumber string
	b.updateSession(c.Sender().ID, func(s *session) {
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
		CreatedByUsername: b.createdByUsername(ctx, c),
		FulfillmentDate:   fulfillmentDate,
	})
	if err != nil {
		slog.ErrorContext(ctx, "update order failed", "order_number", orderNumber, "error", err)
		return sendText(c, "Не удалось обновить заказ.")
	}
	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	summary := responses.OrderUpdated(order, fromName, toName)
	if err := b.notifyWorkshop(ctx, c.Sender().ID, summary); err != nil {
		slog.WarnContext(ctx, "notify workshop about order update failed", "order_number", order.Number, "error", err)
	}
	return sendHTML(c, summary)
}

func (b *OrderBot) orderDraftMarkup(orderNumber string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	if strings.TrimSpace(orderNumber) != "" {
		markup.Inline(
			markup.Row(
				markup.Data("Обновить заказ", "update_order"),
				markup.Data("Отмена", "cancel_cb"),
			),
			markup.Row(markup.Data("Шаблоны", "open_templates")),
		)
		return markup
	}
	markup.Inline(
		markup.Row(
			markup.Data("Отправить заказ", "submit_order"),
			markup.Data("Отмена", "cancel_cb"),
		),
		markup.Row(markup.Data("Шаблоны", "open_templates")),
	)
	return markup
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
