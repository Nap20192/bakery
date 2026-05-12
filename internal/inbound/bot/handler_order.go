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
		return c.Send("Не удалось получить шаблон заказа.")
	}
	return c.Send(responses.Template(template), tele.ModeHTML)
}

func (b *OrderBot) handleCancel(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	return c.Send("Заказ отменен.")
}

func (b *OrderBot) handleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if strings.HasPrefix(text, "/") {
		return c.Send("Неизвестная команда.\n\n/start - список команд")
	}
	if text == "" {
		return c.Send("Отправьте batch-заказ одним сообщением.")
	}
	return b.handleBulkOrder(c, text)
}

func (b *OrderBot) handleBulkOrder(c tele.Context, text string) error {
	ctx := requestContext(c)
	result := b.orderSvc.ValidateBulkOrder(ctx, text)
	if len(result.ValidItems) == 0 {
		return c.Send(responses.ValidationErrors(result.Errors), tele.ModeHTML)
	}

	fromDepartmentID, toDepartmentID := b.orderDepartmentsForSender(ctx, c)
	if fromDepartmentID == nil && toDepartmentID == nil {
		return c.Send("Заказ создается от магазина в цех. Выберите магазин через /start.")
	}
	b.updateSession(c.Sender().ID, func(s *session) {
		s.items = result.ValidItems
		s.fromDepartmentID = fromDepartmentID
		s.toDepartmentID = toDepartmentID
		s.fulfillmentDate = result.FulfillmentDate
	})

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Отправить", "confirm"),
			markup.Data("Отмена", "cancel_cb"),
		),
	)
	return c.Send(responses.BulkOrderCheck(result), tele.ModeHTML, markup)
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
	})

	_ = c.Respond()
	if len(items) == 0 {
		return c.Send("Заказ пустой или уже отправлен.")
	}
	if fromDepartmentID == nil && toDepartmentID == nil {
		fromDepartmentID, toDepartmentID = b.orderDepartmentsForSender(ctx, c)
	}

	order, err := b.orderSvc.CreateOrder(ctx, orderdomain.CreateOrderInput{
		Items:            items,
		FromDepartmentID: fromDepartmentID,
		ToDepartmentID:   toDepartmentID,
		FulfillmentDate:  fulfillmentDate,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create order failed", "error", err)
		return c.Send("Не удалось создать заказ. Проверьте заказ и попробуйте снова.")
	}

	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	summary := responses.OrderSummary(order, fromName, toName)
	if err := b.notifyWorkshop(ctx, c.Sender().ID, summary); err != nil {
		slog.WarnContext(ctx, "notify workshop about order failed", "order_number", order.Number, "error", err)
	}

	slog.InfoContext(applog.WithOrderNumber(ctx, order.Number), "order created", "items", len(items))
	return c.Send(summary, tele.ModeHTML)
}

func (b *OrderBot) handleCancelCallback(c tele.Context) error {
	b.clearSession(c.Sender().ID)
	_ = c.Respond()
	return c.Send("Заказ отменен.")
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
		if _, err := b.tele.Send(tele.ChatID(*user.TelegramID), message, tele.ModeHTML); err != nil {
			slog.WarnContext(ctx, "send workshop order notification failed", "telegram_id", *user.TelegramID, "error", err)
		}
	}
	return nil
}
