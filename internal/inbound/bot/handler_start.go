package bot

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleStart(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.resetSession(sender.ID)

	ctx := requestContext(c)
	if _, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID); err == nil {
		return sendHTML(c, responses.Start(), b.actionMarkup(c))
	}
	if strings.TrimSpace(sender.Username) == "" {
		return sendText(c, "У аккаунта Telegram должен быть username. Укажите username в Telegram и попросите администратора добавить его пользователю.")
	}
	user, err := b.authSvc.AuthenticateTelegram(ctx, sender.ID, sender.Username)
	if err != nil {
		return sendText(c, "Пользователь не заведён. Обратитесь к администратору.")
	}
	c.Set(authUserContextKey, user)
	return sendHTML(c, responses.Start(), b.actionMarkup(c))
}

func (b *OrderBot) handleHelp(c tele.Context) error {
	return sendHTML(c, responses.Help(), b.actionMarkup(c))
}
