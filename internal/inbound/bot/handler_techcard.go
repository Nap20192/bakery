package bot

import (
	"log/slog"
	"strings"
	"time"

	applog "bakery/pkg/logger"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleTechCard(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 1 {
		return c.Send("Формат: /techcard code")
	}
	ctx = applog.WithProductCode(ctx, args[0])
	card, err := b.techCardSvc.GetByCode(ctx, args[0], time.Now().UTC())
	if err != nil {
		slog.WarnContext(ctx, "get tech card failed", "error", err)
		return c.Send("Не удалось получить техкарту по этому коду.")
	}
	return c.Send(responses.TechCard(card), tele.ModeHTML)
}
