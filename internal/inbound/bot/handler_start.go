package bot

import (
	tele "gopkg.in/telebot.v3"
)

func (b *OrderBot) handleStart(c tele.Context) error {
	if sender := c.Sender(); sender != nil {
		b.resetSession(sender.ID)
	}
	return sendHTML(c, responses.Start(), b.actionMarkup(c))
}

func (b *OrderBot) handleHelp(c tele.Context) error {
	return sendHTML(c, responses.Help(), b.actionMarkup(c))
}
