package bot

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"bakery/internal/app"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleTemplates(c tele.Context) error {
	ctx := requestContext(c)
	templates, err := b.orderSvc.ListOrderTemplates(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "list order templates failed", "error", err)
		return sendText(c, "Не удалось получить шаблоны.")
	}
	if len(templates) == 0 {
		return sendText(c, "Шаблонов пока нет.")
	}

	canDelete := b.canManageTemplates(c)
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(templates)+1)
	rows = append(rows, markup.Row(markup.Data("Все шаблоны", "template_all")))
	for _, template := range templates {
		templateID := strconv.FormatInt(template.ID, 10)
		if canDelete {
			rows = append(rows, markup.Row(
				markup.Data(template.Name, "template_use", templateID),
				markup.Data("Удалить", "template_delete_confirm", templateID),
			))
			continue
		}
		rows = append(rows, markup.Row(markup.Data(template.Name, "template_use", templateID)))
	}
	markup.Inline(rows...)
	return sendText(c, "Шаблоны", markup)
}

func (b *OrderBot) handleTemplateAll(c tele.Context) error {
	ctx := requestContext(c)
	template, err := b.orderSvc.CombinedOrderTemplate(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "get combined order template failed", "error", err)
		return sendText(c, "Не удалось получить общий шаблон.")
	}
	if strings.TrimSpace(template) == "" {
		return sendText(c, "Шаблонов пока нет.")
	}
	_ = c.Respond()
	return sendHTML(c, responses.Template(template), b.actionMarkup(c))
}

func (b *OrderBot) handleTemplateUse(c tele.Context) error {
	ctx := requestContext(c)
	id, err := strconv.ParseInt(strings.TrimSpace(c.Callback().Data), 10, 64)
	if err != nil || id <= 0 {
		return sendText(c, "Не удалось определить шаблон.")
	}
	template, err := b.orderSvc.GetOrderTemplate(ctx, id)
	if err != nil {
		slog.WarnContext(ctx, "get order template failed", "template_id", id, "error", err)
		return sendText(c, "Шаблон не найден.")
	}
	_ = c.Respond()
	return sendHTML(c, responses.Template(template.Body), b.actionMarkup(c))
}

func (b *OrderBot) handleTemplateDeleteConfirm(c tele.Context) error {
	ctx := requestContext(c)
	id, err := templateIDFromCallback(c)
	if err != nil {
		return sendText(c, "Не удалось определить шаблон.")
	}
	template, err := b.orderSvc.GetOrderTemplate(ctx, id)
	if err != nil {
		slog.WarnContext(ctx, "get order template before delete failed", "template_id", id, "error", err)
		return sendText(c, "Шаблон не найден.")
	}

	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("Удалить", "template_delete", strconv.FormatInt(template.ID, 10)),
			markup.Data("Отмена", "open_templates"),
		),
	)
	_ = c.Respond()
	return sendText(c, fmt.Sprintf("Удалить шаблон %s?", template.Name), markup)
}

func (b *OrderBot) handleTemplateDelete(c tele.Context) error {
	ctx := requestContext(c)
	id, err := templateIDFromCallback(c)
	if err != nil {
		return sendText(c, "Не удалось определить шаблон.")
	}
	if err := b.orderSvc.DeleteOrderTemplate(ctx, id); err != nil {
		slog.ErrorContext(ctx, "delete order template failed", "template_id", id, "error", err)
		return sendText(c, "Не удалось удалить шаблон.")
	}
	_ = c.Respond()
	return sendText(c, "Шаблон удален.", b.actionMarkup(c))
}

func (b *OrderBot) handleAddTemplate(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.waitingTemplate = true
	})
	return sendText(c, "Отправьте шаблон одним сообщением.\nПервая строка - НАЗВАНИЕ заглавными буквами.\nДальше строки: код название 0", b.actionMarkup(c))
}

func (b *OrderBot) createTemplateFromMessage(c tele.Context, text string) error {
	ctx := requestContext(c)
	user, err := b.authUserFromContext(c)
	if err != nil {
		return err
	}
	template, validation, err := b.orderSvc.CreateOrderTemplate(ctx, &user.ID, text)
	if err != nil {
		slog.ErrorContext(ctx, "create order template failed", "error", err)
		return sendText(c, "Не удалось сохранить шаблон.")
	}
	if len(validation.Errors) > 0 {
		return sendHTML(c, responses.ValidationErrors(validation.Errors))
	}
	sender := c.Sender()
	if sender != nil {
		b.updateSession(sender.ID, func(s *session) {
			s.waitingTemplate = false
		})
	}
	return sendText(c, fmt.Sprintf("Шаблон сохранен: %s", template.Name), b.actionMarkup(c))
}

func (b *OrderBot) isWaitingTemplate(uid int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[uid]
	return s != nil && s.waitingTemplate
}

func (b *OrderBot) canManageTemplates(c tele.Context) bool {
	user, ok := b.currentUser(c)
	if !ok || b.rbacSvc == nil {
		return false
	}
	return b.rbacSvc.HasPermission(user.Role, app.PermissionTemplateManage)
}

func templateIDFromCallback(c tele.Context) (int64, error) {
	if c.Callback() == nil {
		return 0, fmt.Errorf("missing callback")
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Callback().Data), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid template id")
	}
	return id, nil
}
