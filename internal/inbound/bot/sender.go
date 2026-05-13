package bot

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

const telegramMessageLimit = 3600

func sendHTML(c tele.Context, message string, opts ...interface{}) error {
	return sendTelegramChunks(func(text string, chunkOpts ...interface{}) error {
		return c.Send(text, chunkOpts...)
	}, message, []interface{}{tele.ModeHTML}, opts...)
}

func sendText(c tele.Context, message string, opts ...interface{}) error {
	return sendTelegramChunks(func(text string, chunkOpts ...interface{}) error {
		return c.Send(text, chunkOpts...)
	}, message, nil, opts...)
}

func (b *OrderBot) sendHTMLToChat(chatID int64, message string) error {
	return sendTelegramChunks(func(text string, opts ...interface{}) error {
		_, err := b.tele.Send(tele.ChatID(chatID), text, opts...)
		return err
	}, message, []interface{}{tele.ModeHTML}, nil)
}

func sendTelegramChunks(send func(string, ...interface{}) error, message string, baseOpts []interface{}, finalOpts ...interface{}) error {
	chunks := splitTelegramMessage(message)
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	for i, chunk := range chunks {
		chunkOpts := append([]interface{}{}, baseOpts...)
		if i == len(chunks)-1 {
			chunkOpts = append(chunkOpts, finalOpts...)
		}
		if err := send(chunk, chunkOpts...); err != nil {
			return err
		}
	}
	return nil
}

func splitTelegramMessage(message string) []string {
	message = strings.TrimSpace(message)
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
