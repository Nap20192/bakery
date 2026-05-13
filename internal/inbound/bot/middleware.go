package bot

import (
	"errors"
	"fmt"
	"log/slog"

	"bakery/internal/app"
	accessdomain "bakery/internal/domain/access"
	"bakery/internal/pkg/enum"

	tele "gopkg.in/telebot.v3"
)

const authUserContextKey = "auth_user"

func (b *baseBot) requirePermissions(permissions ...enum.Permission) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
		if b.rbacSvc == nil {
			return sendText(c, "Сервис прав доступа недоступен.")
		}
			user, err := b.authUserFromContext(c)
			if err != nil {
				return err
			}
			for _, perm := range permissions {
				if !b.rbacSvc.HasPermission(user.Role, perm) {
					return sendText(c, fmt.Sprintf("Доступ запрещён: недостаточно прав (%s).", perm))
				}
			}
			return next(c)
		}
	}
}

func (b *baseBot) authUserFromContext(c tele.Context) (accessdomain.AuthUser, error) {
	if raw := c.Get(authUserContextKey); raw != nil {
		if user, ok := raw.(accessdomain.AuthUser); ok {
			return user, nil
		}
	}
	if b.authSvc == nil {
		return accessdomain.AuthUser{}, sendText(c, "Сервис авторизации недоступен.")
	}
	sender := c.Sender()
	if sender == nil {
		return accessdomain.AuthUser{}, sendText(c, "Не удалось определить пользователя.")
	}
	ctx := requestContext(c)
	user, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		if errors.Is(err, app.ErrAuthUserNotFound) {
			return accessdomain.AuthUser{}, sendText(c, "Пользователь не зарегистрирован. Обратитесь к администратору.")
		}
		slog.ErrorContext(ctx, "auth user lookup failed", "error", err)
		return accessdomain.AuthUser{}, sendText(c, "Ошибка авторизации.")
	}
	c.Set(authUserContextKey, user)
	return user, nil
}
