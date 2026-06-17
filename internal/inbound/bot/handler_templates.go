package bot

import (
	"log/slog"
	"strings"

	"bakery/internal/pkg/enum"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleTemplates(c tele.Context) error {
	if err := b.ensureTemplatesAvailable(c); err != nil {
		return err
	}

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
	rows := make([]tele.Row, 0, len(templates)+1)
	rows = append(rows, markup.Row(markup.Data("Все шаблоны", "template_all")))
	for _, template := range templates {
		rows = append(rows, markup.Row(markup.Data(template.Name, "template_use", template.Name)))
	}
	markup.Inline(rows...)
	return sendText(c, "Шаблоны", markup)
}

func (b *OrderBot) handleTemplateAll(c tele.Context) error {
	if err := b.ensureTemplatesAvailable(c); err != nil {
		return err
	}

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
	if err := b.ensureTemplatesAvailable(c); err != nil {
		return err
	}

	ctx := requestContext(c)
	theme := strings.TrimSpace(c.Callback().Data)
	if theme == "" {
		return sendText(c, "Не удалось определить шаблон.")
	}
	template, err := b.orderSvc.GetOrderTemplate(ctx, theme)
	if err != nil {
		slog.WarnContext(ctx, "get order template failed", "template", theme, "error", err)
		return sendText(c, "Шаблон не найден.")
	}
	_ = c.Respond()
	return sendHTML(c, responses.Template(template.Body), b.actionMarkup(c))
}

func (b *OrderBot) ensureTemplatesAvailable(c tele.Context) error {
	user, ok := b.currentUser(c)
	if !ok || b.userDepartmentType(c, user) != string(enum.DepartmentTypeWorkshop) {
		return nil
	}
	return sendText(c, "В цеху шаблоны не используются. Откройте последние заказы и фильтры.", b.actionMarkup(c))
}
