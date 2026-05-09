package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"bakery/internal/app"
	accessdomain "bakery/internal/domain/access"

	tele "gopkg.in/telebot.v3"
)

const authUserContextKey = "auth_user"

const (
	permTechCard = "techcard:view"
)

var rolePermissions = map[string]map[string]struct{}{
	accessdomain.RoleAdmin: {
		permTechCard: {},
	},
}

func (b *baseBot) requirePermissions(permissions ...string) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user, err := b.authUserFromContext(c)
			if err != nil {
				return err
			}
			for _, perm := range permissions {
				if !userHasPermission(user.Role, perm) {
					return c.Send(fmt.Sprintf("Доступ запрещён: недостаточно прав (%s).", perm))
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
		return accessdomain.AuthUser{}, c.Send("Сервис авторизации недоступен.")
	}
	sender := c.Sender()
	if sender == nil {
		return accessdomain.AuthUser{}, c.Send("Не удалось определить пользователя.")
	}
	ctx := requestContext(c)
	user, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		if errors.Is(err, app.ErrAuthUserNotFound) {
			return accessdomain.AuthUser{}, c.Send("Пользователь не зарегистрирован. Обратитесь к администратору.")
		}
		slog.ErrorContext(ctx, "auth user lookup failed", "error", err)
		return accessdomain.AuthUser{}, c.Send("Ошибка авторизации.")
	}
	c.Set(authUserContextKey, user)
	return user, nil
}

func userHasPermission(role string, permission string) bool {
	role = accessdomain.NormalizeRole(role)
	permission = strings.TrimSpace(permission)
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = perms[permission]
	return ok
}
