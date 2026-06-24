package bot

import (
	"fmt"
	"html"
	"log/slog"
	"strings"

	accessdomain "bakery/internal/services/auth/domain"

	tele "gopkg.in/telebot.v3"
)

// handleStart is the only entry point. If the user is already authorized it
// just re-sends their info; otherwise it begins the password gate.
func (b *OrderBot) handleStart(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	ctx := requestContext(c)

	// Already authorized — do not ask for the password again, just re-send info.
	if user, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID); err == nil {
		b.resetSession(sender.ID)
		return b.sendUserInfo(c, user)
	}

	b.resetSession(sender.ID)
	username := strings.TrimSpace(sender.Username)
	if username == "" {
		return sendText(c, "Добавьте username в настройках Telegram и нажмите /start.")
	}
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

// handleText treats free text as the password when the user is in the gate.
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
	user, err := b.authSvc.VerifyPassword(ctx, sess.username, password)
	if err != nil {
		return sendText(c, "Неверный пароль. Попробуйте ещё раз или нажмите /start.")
	}

	// Bind the Telegram account to the user record — this is the authorization.
	if bound, err := b.authSvc.AuthenticateTelegram(ctx, sender.ID, sender.Username); err != nil {
		slog.WarnContext(ctx, "bind telegram after password failed", "error", err)
	} else {
		user = bound
	}
	b.resetSession(sender.ID)
	return b.sendUserInfo(c, user)
}

// sendUserInfo sends the account info with an inline button that opens the mini
// app inside Telegram (WebApp button, so the app receives Telegram initData).
func (b *OrderBot) sendUserInfo(c tele.Context, user accessdomain.AuthUser) error {
	if markup := b.openAppMarkup(); markup != nil {
		return sendHTML(c, b.userInfoMessage(user), markup)
	}
	return sendHTML(c, b.userInfoMessage(user))
}

func (b *OrderBot) openAppMarkup() *tele.ReplyMarkup {
	url := strings.TrimSpace(b.miniAppURL)
	if url == "" {
		return nil
	}
	markup := &tele.ReplyMarkup{}
	btn := markup.WebApp("🥐 Открыть приложение", &tele.WebApp{URL: url})
	markup.Inline(markup.Row(btn))
	return markup
}

// userInfoMessage renders the authorized user's account info plus a short guide
// on what to do next (no password — it is stored as an irreversible hash).
func (b *OrderBot) userInfoMessage(user accessdomain.AuthUser) string {
	var sb strings.Builder
	sb.WriteString("<b>Вы авторизованы ✅</b>\n\n")
	fmt.Fprintf(&sb, "Логин: <code>%s</code>\n", html.EscapeString(user.Username))
	fmt.Fprintf(&sb, "Роль: %s\n", html.EscapeString(roleTitle(user.Role)))
	if user.TelegramUsername != nil && strings.TrimSpace(*user.TelegramUsername) != "" {
		fmt.Fprintf(&sb, "Telegram: @%s\n", html.EscapeString(strings.TrimPrefix(strings.TrimSpace(*user.TelegramUsername), "@")))
	}

	switch accessdomain.NormalizeRole(user.Role) {
	case accessdomain.RoleShop:
		sb.WriteString("\n<b>Как создавать заказы:</b>\n")
		sb.WriteString("1. Откройте приложение (ссылка ниже).\n")
		sb.WriteString("2. Выберите магазин, из которого отправляете заказ, и дату.\n")
		sb.WriteString("3. Укажите количество у нужных блюд (можно «Вставить списком»).\n")
		sb.WriteString("4. Нажмите «Создать заказ» — он уйдёт в цех, уведомление придёт сюда.\n")
	case accessdomain.RoleBaker:
		sb.WriteString("\n<b>Что делать дальше:</b>\n")
		sb.WriteString("Откройте приложение — там все заказы и расчёт теста.\n")
	default:
		sb.WriteString("\n<b>Что делать дальше:</b>\n")
		sb.WriteString("Откройте приложение — заказы, пользователи и блюда.\n")
	}

	return sb.String()
}

func roleTitle(role string) string {
	switch accessdomain.NormalizeRole(role) {
	case accessdomain.RoleAdmin:
		return "Администратор"
	case accessdomain.RoleShop:
		return "Магазин"
	case accessdomain.RoleBaker:
		return "Цех"
	default:
		return role
	}
}
