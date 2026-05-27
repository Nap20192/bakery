package bot

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"bakery/internal/app"

	tele "gopkg.in/telebot.v3"
)

type baseBot struct {
	tele          *tele.Bot
	orderSvc      *app.OrderService
	authSvc       *app.AuthService
	rbacSvc       *app.RbacService
	departmentSvc *app.DepartmentService
	monitorSvc    *app.MonitorService
	syncSvc       *app.SyncService
	techCardSvc   *app.TechCardService
	miniAppURL    string
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
	rbacSvc *app.RbacService,
	departmentSvc *app.DepartmentService,
	monitorSvc *app.MonitorService,
	syncSvc *app.SyncService,
	techCardSvc *app.TechCardService,
	miniAppURL string,
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
			tele:          b,
			orderSvc:      orderSvc,
			authSvc:       authSvc,
			rbacSvc:       rbacSvc,
			departmentSvc: departmentSvc,
			monitorSvc:    monitorSvc,
			syncSvc:       syncSvc,
			techCardSvc:   techCardSvc,
			miniAppURL:    miniAppURL,
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
