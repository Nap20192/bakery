package bot

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

type baseBot struct {
	tele           *tele.Bot
	authSvc        AuthBackend
	departmentSvc  DepartmentBackend
	miniAppURL     string
	workshopChatID int64
}

type OrderBot struct {
	*baseBot
	mu       sync.Mutex
	sessions map[int64]*session
}

func NewOrderBot(
	token string,
	authSvc AuthBackend,
	departmentSvc DepartmentBackend,
	miniAppURL string,
	workshopChatID int64,
) (*OrderBot, error) {
	miniAppURL = strings.TrimSpace(miniAppURL)
	if err := ValidateMiniAppURL(miniAppURL); err != nil {
		return nil, err
	}

	b, err := newTelegramBot(token)
	if err != nil {
		return nil, fmt.Errorf("orderbot: new: %w", err)
	}

	bot := &OrderBot{
		baseBot: &baseBot{
			tele:           b,
			authSvc:        authSvc,
			departmentSvc:  departmentSvc,
			miniAppURL:     miniAppURL,
			workshopChatID: workshopChatID,
		},
		sessions: make(map[int64]*session),
	}
	bot.register()
	return bot, nil
}

func ValidateMiniAppURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("orderbot: mini app URL must be a valid https URL")
	}
	return nil
}

func IsRejectedToken(err error) bool {
	return errors.Is(err, tele.ErrUnauthorized) || errors.Is(err, tele.ErrNotFound)
}

func newTelegramBot(token string) (*tele.Bot, error) {
	return tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
}

func (b *OrderBot) Start() {
	go b.cleanupLoop()
	b.tele.Start()
}

func (b *OrderBot) Name() string {
	if b == nil || b.tele == nil || b.tele.Me == nil {
		return ""
	}
	return b.tele.Me.FirstName
}

func (b *OrderBot) Username() string {
	if b == nil || b.tele == nil || b.tele.Me == nil {
		return ""
	}
	return b.tele.Me.Username
}

func (b *OrderBot) Stop() {
	b.tele.Stop()
}
