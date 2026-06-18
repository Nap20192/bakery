package bot

import (
	"errors"
	"log/slog"

	"bakery/internal/pkg/enum"
	accessdomain "bakery/internal/services/auth/domain"
	authuc "bakery/internal/services/auth/usecase/auth"

	tele "gopkg.in/telebot.v3"
)

const authUserContextKey = "auth_user"

func (b *baseBot) requirePermissions(permissions ...enum.Permission) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if b.rbacSvc == nil {
				return sendText(c, msgRBACUnavailable)
			}
			user, err := b.authUserFromContext(c)
			if err != nil {
				return err
			}
			for _, perm := range permissions {
				if !b.rbacSvc.HasPermission(user.Role, perm) {
					return sendText(c, "Доступ запрещён: у вашего пользователя нет прав на эту команду.")
				}
			}
			return next(c)
		}
	}
}

func (b *baseBot) privateChatOnly(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		chat := c.Chat()
		if chat == nil || chat.Type != tele.ChatPrivate {
			// Group/channel updates are not handled, but log the chat id so the
			// workshop group id can be discovered for TELEGRAM_WORKSHOP_CHAT_ID.
			if chat != nil {
				switch chat.Type {
				case tele.ChatGroup, tele.ChatSuperGroup:
					slog.Info("GROUP CHAT ID — use this for TELEGRAM_WORKSHOP_CHAT_ID",
						"chat_id", chat.ID,
						"chat_type", chat.Type,
						"chat_title", chat.Title,
					)
				default:
					slog.Info("non-private chat update ignored",
						"chat_id", chat.ID,
						"chat_type", chat.Type,
					)
				}
			}
			return nil
		}
		return next(c)
	}
}

func (b *baseBot) authUserFromContext(c tele.Context) (accessdomain.AuthUser, error) {
	if raw := c.Get(authUserContextKey); raw != nil {
		if user, ok := raw.(accessdomain.AuthUser); ok {
			return user, nil
		}
	}
	if b.authSvc == nil {
		return accessdomain.AuthUser{}, sendText(c, msgAuthUnavailable)
	}
	sender := c.Sender()
	if sender == nil {
		return accessdomain.AuthUser{}, sendText(c, msgTelegramUserUnknown)
	}
	ctx := requestContext(c)
	user, err := b.authSvc.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		if errors.Is(err, authuc.ErrAuthUserNotFound) {
			return accessdomain.AuthUser{}, sendText(c, msgUserNotLinked)
		}
		slog.ErrorContext(ctx, "auth user lookup failed", "error", err)
		return accessdomain.AuthUser{}, sendText(c, "Ошибка авторизации.")
	}
	c.Set(authUserContextKey, user)
	return user, nil
}
