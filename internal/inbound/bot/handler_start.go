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

	if _, err := b.authSvc.GetUserByTelegramID(requestContext(c), sender.ID); err == nil {
		return sendHTML(c, responses.Start(), b.actionMarkup(c))
	}
	b.updateSession(sender.ID, func(s *session) { s.awaitingPassword = true })
	return sendText(c, "Введите пароль для входа:")
}

func (b *OrderBot) handleHelp(c tele.Context) error {
	return sendHTML(c, responses.Help(), b.actionMarkup(c))
}

func (b *OrderBot) isAwaitingPassword(c tele.Context) bool {
	sender := c.Sender()
	if sender == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.sessions[sender.ID]
	return s != nil && s.awaitingPassword
}

// handlePasswordEntry verifies a password against the account whose
// telegram_username matches the sender, binds the sender's telegram_id, then
// the user is identified by telegram_id on subsequent interactions.
func (b *OrderBot) handlePasswordEntry(c tele.Context, password string) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	user, err := b.authSvc.AuthenticateTelegram(ctx, sender.ID, sender.Username, strings.TrimSpace(password))
	if err != nil {
		return sendText(c, "Неверный пароль или пользователь не заведён. Обратитесь к администратору.")
	}
	b.updateSession(sender.ID, func(s *session) { s.awaitingPassword = false })
	c.Set(authUserContextKey, user)
	return sendHTML(c, responses.Start(), b.actionMarkup(c))
}
