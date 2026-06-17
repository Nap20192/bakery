package bot

import (
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestSendTelegramChunksUsesTypedOptions(t *testing.T) {
	var got []*tele.SendOptions
	err := sendTelegramChunks(func(_ string, opts *tele.SendOptions) error {
		got = append(got, opts)
		return nil
	}, "message", messageOptions{parseMode: tele.ModeHTML})
	if err != nil {
		t.Fatalf("sendTelegramChunks returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("chunks = %d, want 1", len(got))
	}
	if got[0] == nil || got[0].ParseMode != tele.ModeHTML {
		t.Fatalf("options = %#v", got[0])
	}
}

func TestSendTelegramChunksAttachesMarkupOnlyToLastChunk(t *testing.T) {
	markup := &tele.ReplyMarkup{}
	var got []*tele.SendOptions
	err := sendTelegramChunks(func(_ string, opts *tele.SendOptions) error {
		got = append(got, opts)
		return nil
	}, stringsOfLen(telegramMessageLimit+1), messageOptions{
		parseMode: tele.ModeHTML,
		markup:    markup,
	})
	if err != nil {
		t.Fatalf("sendTelegramChunks returned error: %v", err)
	}
	if len(got) < 2 {
		t.Fatalf("chunks = %d, want at least 2", len(got))
	}
	for i, opts := range got[:len(got)-1] {
		if opts == nil || opts.ReplyMarkup != nil {
			t.Fatalf("chunk %d options = %#v, want parse mode without markup", i, opts)
		}
	}
	if got[len(got)-1] == nil || got[len(got)-1].ReplyMarkup != markup {
		t.Fatalf("last options = %#v, want markup", got[len(got)-1])
	}
}

func TestSplitTelegramMessageKeepsPreTagsBalanced(t *testing.T) {
	chunks := splitTelegramMessage("<pre>" + stringsOfLen(telegramMessageLimit+1) + "</pre>")
	if len(chunks) < 2 {
		t.Fatalf("chunks = %d, want at least 2", len(chunks))
	}
	for _, chunk := range chunks {
		if !strings.HasPrefix(chunk, "<pre>") || !strings.HasSuffix(chunk, "</pre>") {
			t.Fatalf("chunk has unbalanced pre tags: %q", chunk)
		}
		if len(chunk) > telegramMessageLimit {
			t.Fatalf("chunk len = %d, limit %d", len(chunk), telegramMessageLimit)
		}
	}
}

func TestPrivateChatOnlyBlocksGroupChats(t *testing.T) {
	var called bool
	handler := (&baseBot{}).privateChatOnly(func(tele.Context) error {
		called = true
		return nil
	})

	if err := handler(newTestContext(tele.ChatPrivate)); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !called {
		t.Fatal("handler was not called for private chat")
	}

	called = false
	if err := handler(newTestContext(tele.ChatGroup)); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if called {
		t.Fatal("handler was called for group chat")
	}
}

func stringsOfLen(size int) string {
	var b strings.Builder
	for b.Len() < size {
		b.WriteByte('x')
	}
	return b.String()
}

func newTestContext(chatType tele.ChatType) tele.Context {
	return (&tele.Bot{}).NewContext(tele.Update{
		Message: &tele.Message{
			Chat: &tele.Chat{Type: chatType},
		},
	})
}
