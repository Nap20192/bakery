package bot

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"bakery/pkg/rabbitmq/consumer"

	tele "gopkg.in/telebot.v3"
)

type baseBot struct {
	tele           *tele.Bot
	authSvc        AuthBackend
	departmentSvc  DepartmentBackend
	eventConsumer  *consumer.Consumer
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
	eventConsumer *consumer.Consumer,
	miniAppURL string,
	workshopChatID int64,
) (*OrderBot, error) {
	miniAppURL = strings.TrimSpace(miniAppURL)
	if miniAppURL != "" {
		parsed, err := url.Parse(miniAppURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return nil, fmt.Errorf("orderbot: mini app URL must be a valid https URL")
		}
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
			eventConsumer:  eventConsumer,
			miniAppURL:     miniAppURL,
			workshopChatID: workshopChatID,
		},
		sessions: make(map[int64]*session),
	}
	bot.register()
	return bot, nil
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
