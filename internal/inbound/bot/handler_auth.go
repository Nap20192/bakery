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
	args := strings.Fields(c.Message().Payload)
	if len(args) != 2 {
		return c.Send("Формат: /login username password")
	}

	user, err := b.authSvc.LoginTelegramUser(ctx, c.Sender().ID, c.Sender().Username, args[0], args[1])
	if err != nil {
		slog.WarnContext(ctx, "login failed", "username", args[0], "error", err)
		return c.Send("Не удалось войти: проверьте username и password.")
	}

	c.Set(authUserContextKey, user)
	return c.Send(fmt.Sprintf("Вход выполнен. Роль: %s", user.Role))
}

func (b *OrderBot) handleLogout(c tele.Context) error {
	ctx := requestContext(c)
	if c.Sender() == nil {
		return c.Send("Не удалось определить пользователя.")
	}
	if err := b.authSvc.LogoutTelegramUser(ctx, c.Sender().ID); err != nil {
		slog.ErrorContext(ctx, "logout failed", "error", err)
		return c.Send("Не удалось выполнить выход. Попробуйте позже.")
	}
	c.Set(authUserContextKey, nil)
	return c.Send("Вы вышли.")
}

func (b *OrderBot) handleAddUser(c tele.Context) error {
	ctx := requestContext(c)
	args := strings.Fields(c.Message().Payload)
	if len(args) != 2 {
		return c.Send("Формат: /adduser username password")
	}

	user, err := b.authSvc.CreateUserWithPassword(ctx, accessdomain.PasswordAuthUserInput{
		Username: args[0],
		Password: args[1],
		Role:     accessdomain.RoleAdmin,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create admin user failed", "username", args[0], "error", err)
		return c.Send("Не удалось создать пользователя. Проверьте данные и попробуйте снова.")
	}

	return c.Send(fmt.Sprintf("Администратор %s создан.", user.Username))
}
