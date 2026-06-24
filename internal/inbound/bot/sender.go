package bot

import (
	"log/slog"
	"strings"

	tele "gopkg.in/telebot.v3"
)

const telegramMessageLimit = 3600

type messageOptions struct {
	parseMode tele.ParseMode
	markup    *tele.ReplyMarkup
}

type chunkSender func(text string, options *tele.SendOptions) error

func sendHTML(c tele.Context, message string, markup ...*tele.ReplyMarkup) error {
	return sendTelegramChunks(contextSender(c), message, messageOptions{
		parseMode: tele.ModeHTML,
		markup:    firstMarkup(markup),
	})
}

func sendText(c tele.Context, message string, markup ...*tele.ReplyMarkup) error {
	return sendTelegramChunks(contextSender(c), message, messageOptions{
		markup: firstMarkup(markup),
	})
}

func (b *OrderBot) sendHTMLToChat(chatID int64, message string, markup ...*tele.ReplyMarkup) error {
	slog.Debug("bot send to chat: start", "component", "bot.sender", "chat_id", chatID, "bytes", len(message))
	err := sendTelegramChunks(botSender(b.tele, chatID), message, messageOptions{
		parseMode: tele.ModeHTML,
		markup:    firstMarkup(markup),
	})
	if err != nil {
		slog.Error("bot send to chat: failed", "component", "bot.sender", "chat_id", chatID, "error", err)
	} else {
		slog.Debug("bot send to chat: ok", "component", "bot.sender", "chat_id", chatID)
	}
	return err
}

func contextSender(c tele.Context) chunkSender {
	return func(text string, options *tele.SendOptions) error {
		if options == nil {
			return c.Send(text)
		}
		return c.Send(text, options)
	}
}

func botSender(bot *tele.Bot, chatID int64) chunkSender {
	return func(text string, options *tele.SendOptions) error {
		var err error
		if options == nil {
			_, err = bot.Send(tele.ChatID(chatID), text)
		} else {
			_, err = bot.Send(tele.ChatID(chatID), text, options)
		}
		slog.Debug("bot.Send chunk result",
			"component", "bot.sender",
			"chat_id", chatID,
			"bytes", len(text),
			"ok", err == nil,
			"error", errString(err),
		)
		return err
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func sendTelegramChunks(send chunkSender, message string, options messageOptions) error {
	chunks := splitTelegramMessage(message)
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	for i, chunk := range chunks {
		if err := send(chunk, options.forChunk(i == len(chunks)-1)); err != nil {
			return err
		}
	}
	return nil
}

func (o messageOptions) forChunk(last bool) *tele.SendOptions {
	if o.parseMode == "" && (!last || o.markup == nil) {
		return nil
	}
	opts := &tele.SendOptions{ParseMode: o.parseMode}
	if last {
		opts.ReplyMarkup = o.markup
	}
	return opts
}

func firstMarkup(markups []*tele.ReplyMarkup) *tele.ReplyMarkup {
	if len(markups) == 0 {
		return nil
	}
	return markups[0]
}

func splitTelegramMessage(message string) []string {
	message = strings.TrimSpace(message)
	if strings.HasPrefix(message, "<pre>") && strings.HasSuffix(message, "</pre>") {
		return splitPreMessage(strings.TrimSuffix(strings.TrimPrefix(message, "<pre>"), "</pre>"))
	}
	if len(message) <= telegramMessageLimit {
		return []string{message}
	}

	var chunks []string
	var current strings.Builder
	for _, line := range strings.Split(message, "\n") {
		nextLen := current.Len() + len(line) + 1
		if current.Len() > 0 && nextLen > telegramMessageLimit {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if len(line) > telegramMessageLimit {
			for len(line) > telegramMessageLimit {
				chunks = append(chunks, line[:telegramMessageLimit])
				line = line[telegramMessageLimit:]
			}
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}

func splitPreMessage(body string) []string {
	const preOverhead = len("<pre></pre>")
	limit := telegramMessageLimit - preOverhead
	if len(body)+preOverhead <= telegramMessageLimit {
		return []string{"<pre>" + body + "</pre>"}
	}
	rawChunks := splitPlainText(body, limit)
	chunks := make([]string, 0, len(rawChunks))
	for _, chunk := range rawChunks {
		chunks = append(chunks, "<pre>"+chunk+"</pre>")
	}
	return chunks
}

func splitPlainText(message string, limit int) []string {
	var chunks []string
	var current strings.Builder
	for _, line := range strings.Split(message, "\n") {
		nextLen := current.Len() + len(line) + 1
		if current.Len() > 0 && nextLen > limit {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if len(line) > limit {
			for len(line) > limit {
				chunks = append(chunks, line[:limit])
				line = line[limit:]
			}
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}
