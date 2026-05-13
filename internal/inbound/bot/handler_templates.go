package bot

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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

	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(templates))
	var text strings.Builder
	text.WriteString("Выберите шаблон:\n")
	currentTheme := ""
	for _, template := range templates {
		if template.Theme != currentTheme {
			currentTheme = template.Theme
			text.WriteString("\n")
			text.WriteString(currentTheme)
			text.WriteString("\n")
		}
		text.WriteString("- ")
		text.WriteString(template.Name)
		text.WriteString("\n")
		rows = append(rows, markup.Row(markup.Data(template.Name, "template_use", strconv.FormatInt(template.ID, 10))))
	}
	markup.Inline(rows...)
	if err := sendText(c, text.String(), markup); err != nil {
		return err
	}
	return b.sendActionMenu(c)
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
	if err := sendHTML(c, responses.Template(template.Body)); err != nil {
		return err
	}
	return b.sendActionMenu(c)
}

func (b *OrderBot) handleAddTemplate(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.updateSession(sender.ID, func(s *session) {
		s.waitingTemplate = true
	})
	if err := sendText(c, "Отправьте шаблон одним сообщением.\nПервая строка - НАЗВАНИЕ заглавными буквами.\nДальше строки: код название 0"); err != nil {
		return err
	}
	return b.sendActionMenu(c)
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
	if err := sendText(c, fmt.Sprintf("Шаблон сохранен: %s", template.Name)); err != nil {
		return err
	}
	return b.sendActionMenu(c)
}

func (b *OrderBot) isWaitingTemplate(uid int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[uid]
	return s != nil && s.waitingTemplate
}
