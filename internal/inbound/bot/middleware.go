package bot

import (
	"log/slog"

	tele "gopkg.in/telebot.v3"
)

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
