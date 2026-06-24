package bot

import (
	"log/slog"
	"strings"

	tele "gopkg.in/telebot.v3"
)

// handleStart begins the password gate. The user is identified by their
// Telegram username; once found, the bot waits for the password.
func (b *OrderBot) handleStart(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	b.resetSession(sender.ID)

	username := strings.TrimSpace(sender.Username)
	if username == "" {
		return sendText(c, "Добавьте username в настройках Telegram и нажмите /start.")
	}

	ctx := requestContext(c)
	user, err := b.authSvc.GetUserByTelegramUsername(ctx, username)
	if err != nil {
		return sendText(c, "Вы не зарегистрированы. Обратитесь к администратору.")
	}

	b.updateSession(sender.ID, func(s *session) {
		s.awaitingPassword = true
		s.username = user.Username
	})
	return sendText(c, "Введите пароль:")
}

// handleText treats free text as the password when the user is in the password
// gate. On success the Telegram account is bound (the user is authorized).
func (b *OrderBot) handleText(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return nil
	}
	sess := b.getSession(sender.ID)
	if !sess.awaitingPassword {
		return sendText(c, "Нажмите /start, чтобы войти.")
	}

	ctx := requestContext(c)
	password := strings.TrimSpace(c.Text())
	if _, err := b.authSvc.VerifyPassword(ctx, sess.username, password); err != nil {
		return sendText(c, "Неверный пароль. Попробуйте ещё раз или нажмите /start.")
	}

	// Bind the Telegram account to the user record — this is the authorization.
	if _, err := b.authSvc.AuthenticateTelegram(ctx, sender.ID, sender.Username); err != nil {
		slog.WarnContext(ctx, "bind telegram after password failed", "error", err)
	}
	b.resetSession(sender.ID)

	if markup := b.openAppMarkup(); markup != nil {
		return sendHTML(c, "Вы авторизованы ✅", markup)
	}
	return sendHTML(c, "Вы авторизованы ✅")
}

// openAppMarkup builds an inline keyboard with the mini app button, or nil when
// no mini app URL is configured.
func (b *OrderBot) openAppMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	if btn, ok := b.miniAppButton(markup, "Открыть приложение", "", "", nil); ok {
		markup.Inline(markup.Row(btn))
		return markup
	}
	return nil
}
