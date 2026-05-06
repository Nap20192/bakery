package bot

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"bakery/internal/app"
	"bakery/internal/domain"

	tele "gopkg.in/telebot.v3"
)

type baseBot struct {
	tele      *tele.Bot
	orderSvc  *app.OrderService
}

type OrderBot struct {
	*baseBot
	mu           sync.Mutex
	sessions     map[int64]*session
	productNames []string
}

type AdminBot struct {
	*baseBot
}

func NewOrderBot(
	token string,
	orderSvc *app.OrderService,
	doughRepo domain.DoughRepository,
) (*OrderBot, error) {
	b, err := newTelegramBot(token)
	if err != nil {
		return nil, fmt.Errorf("orderbot: new: %w", err)
	}

	names, err := productRepo.List()
	if err != nil {
		return nil, fmt.Errorf("orderbot: load products: %w", err)
	}
	sort.Strings(names)

	bot := &OrderBot{
		baseBot: &baseBot{
			tele:      b,
			orderSvc:  orderSvc,
			orderRepo: orderRepo,
			doughRepo: doughRepo,
		},
		sessions:     make(map[int64]*session),
		productNames: names,
	}
	bot.register()
	return bot, nil
}

func NewAdminBot(
	token string,
	orderSvc *app.OrderService,
	orderRepo domain.OrderRepository,
	doughRepo domain.DoughRepository,
) (*AdminBot, error) {
	b, err := newTelegramBot(token)
	if err != nil {
		return nil, fmt.Errorf("adminbot: new: %w", err)
	}
	bot := &AdminBot{
		baseBot: &baseBot{
			tele:      b,
			orderSvc:  orderSvc,
			orderRepo: orderRepo,
			doughRepo: doughRepo,
		},
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

func (b *AdminBot) Start() {
	b.tele.Start()
}

func (b *AdminBot) Stop() {
	b.tele.Stop()
}
