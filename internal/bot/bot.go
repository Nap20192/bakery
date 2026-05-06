package bot

import (
	"fmt"
	"sync"
	"time"

	"bakery/internal/app"

	tele "gopkg.in/telebot.v3"
)

type baseBot struct {
	tele       *tele.Bot
	orderSvc   *app.OrderService
	authSvc    *app.AuthService
	groupSvc   *app.GroupService
	monitorSvc *app.MonitorService
}

type OrderBot struct {
	*baseBot
	mu       sync.Mutex
	sessions map[int64]*session
}

func NewOrderBot(
	token string,
	orderSvc *app.OrderService,
	authSvc *app.AuthService,
	groupSvc *app.GroupService,
	monitorSvc *app.MonitorService,
) (*OrderBot, error) {
	b, err := newTelegramBot(token)
	if err != nil {
		return nil, fmt.Errorf("orderbot: new: %w", err)
	}

	bot := &OrderBot{
		baseBot: &baseBot{
			tele:       b,
			orderSvc:   orderSvc,
			authSvc:    authSvc,
			groupSvc:   groupSvc,
			monitorSvc: monitorSvc,
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

func (b *OrderBot) Stop() {
	b.tele.Stop()
}
