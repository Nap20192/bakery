package bot

import (
	"fmt"
	"log/slog"
	"strings"

	"bakery/internal/app"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleStart(c tele.Context) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Магазин", "dept_shop"),
			markup.Data("Цех", "dept_workshop"),
		),
		markup.Row(
			markup.Data("Шаблоны", "open_templates"),
			markup.Data("Последние заказы", "open_orders"),
		),
	)
	return sendHTML(c, responses.Start(), markup)
}

func (b *OrderBot) handleDepartmentShop(c tele.Context) error {
	ctx := requestContext(c)
	departments, err := b.departmentSvc.ListByType(ctx, app.DepartmentTypeShop)
	if err != nil {
		slog.ErrorContext(ctx, "list shop departments failed", "error", err)
		return sendText(c, "Не удалось получить список магазинов.")
	}
	if len(departments) == 0 {
		return sendText(c, "Магазины не настроены.")
	}

	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(departments))
	for _, department := range departments {
		rows = append(rows, markup.Row(markup.Data(department.Name, "dept_select", department.Code)))
	}
	markup.Inline(rows...)
	_ = c.Respond()
	return sendText(c, "Выберите магазин:", markup)
}

func (b *OrderBot) handleDepartmentWorkshop(c tele.Context) error {
	return b.saveDepartmentByCode(c, workshopDepartmentCode)
}

func (b *OrderBot) handleDepartmentSelect(c tele.Context) error {
	code := strings.TrimSpace(c.Callback().Data)
	if code == "" {
		return sendText(c, "Не удалось определить выбранную локацию.")
	}
	return b.saveDepartmentByCode(c, code)
}

func (b *OrderBot) saveDepartmentByCode(c tele.Context, code string) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	department, err := b.departmentSvc.GetByCode(ctx, code)
	if err != nil {
		slog.WarnContext(ctx, "department lookup failed", "code", code, "error", err)
		return sendText(c, "Локация не найдена.")
	}
	user, err := b.authSvc.SetTelegramUserDepartment(ctx, sender.ID, sender.Username, department.ID)
	if err != nil {
		slog.ErrorContext(ctx, "save user department failed", "department_id", department.ID, "error", err)
		return sendText(c, "Не удалось сохранить локацию.")
	}
	c.Set(authUserContextKey, user)

	fromID, toID, err := b.orderDepartmentsForSelected(ctx, department.ID)
	if err != nil {
		slog.WarnContext(ctx, "resolve order departments failed", "department_id", department.ID, "error", err)
	}
	b.updateSession(sender.ID, func(s *session) {
		s.fromDepartmentID = fromID
		s.toDepartmentID = toID
	})

	_ = c.Respond()
	if err := sendText(c, fmt.Sprintf("Локация сохранена: %s.", department.Name)); err != nil {
		return err
	}
	return b.sendActionMenu(c)
}
