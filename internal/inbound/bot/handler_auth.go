package bot

import (
	"fmt"
	"log/slog"
	"strings"

	accessdomain "bakery/internal/domain/access"

	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleLogin(c tele.Context) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	args := strings.Fields(c.Message().Payload)
	if len(args) != 2 {
		return sendText(c, "Формат: /login username password")
	}

	user, err := b.authSvc.LoginTelegramUser(ctx, sender.ID, sender.Username, args[0], args[1])
	if err != nil {
		slog.WarnContext(ctx, "login failed", "username", args[0], "error", err)
		return sendText(c, "Не удалось войти: проверьте username и password.")
	}

	c.Set(authUserContextKey, user)
	return sendText(c, fmt.Sprintf("Вход выполнен. Роль: %s", user.Role), b.actionMarkup(c))
}

func (b *OrderBot) handleLogout(c tele.Context) error {
	ctx := requestContext(c)
	sender := c.Sender()
	if sender == nil {
		return sendText(c, "Не удалось определить пользователя.")
	}
	if err := b.authSvc.LogoutTelegramUser(ctx, sender.ID); err != nil {
		slog.ErrorContext(ctx, "logout failed", "error", err)
		return sendText(c, "Не удалось выполнить выход. Попробуйте позже.")
	}
	c.Set(authUserContextKey, nil)
	return sendText(c, "Вы вышли.", b.actionMarkup(c))
}

func (b *OrderBot) handleAddUser(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 2 {
		return sendText(c, "Формат: /adduser username password")
	}

	user, err := b.authSvc.CreateUserWithPassword(ctx, accessdomain.PasswordAuthUserInput{
		Username: args[0],
		Password: args[1],
		Role:     accessdomain.RoleAdmin,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create admin user failed", "username", args[0], "error", err)
		return sendText(c, "Не удалось создать пользователя. Проверьте данные и попробуйте снова.")
	}

	return sendText(c, fmt.Sprintf("Администратор %s создан.", user.Username), b.actionMarkup(c))
}
