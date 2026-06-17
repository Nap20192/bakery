package bot

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"bakery/internal/app"

	tele "gopkg.in/telebot.v3"
)

type baseBot struct {
	tele           *tele.Bot
	orderSvc       *app.OrderService
	authSvc        *app.AuthService
	rbacSvc        *app.RbacService
	departmentSvc  *app.DepartmentService
	monitorSvc     *app.MonitorService
	syncSvc        *app.SyncService
	techCardSvc    *app.TechCardService
	orderEvents    app.OrderEventBus
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
	orderSvc *app.OrderService,
	authSvc *app.AuthService,
	rbacSvc *app.RbacService,
	departmentSvc *app.DepartmentService,
	monitorSvc *app.MonitorService,
	syncSvc *app.SyncService,
	techCardSvc *app.TechCardService,
	orderEvents app.OrderEventBus,
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
			orderSvc:       orderSvc,
			authSvc:        authSvc,
			rbacSvc:        rbacSvc,
			departmentSvc:  departmentSvc,
			monitorSvc:     monitorSvc,
			syncSvc:        syncSvc,
			techCardSvc:    techCardSvc,
			orderEvents:    orderEvents,
			miniAppURL:     miniAppURL,
			workshopChatID: workshopChatID,
		},
		sessions: make(map[int64]*session),
	}
	bot.register()
	if err := bot.subscribeOrderEvents(); err != nil {
		return nil, err
	}
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

func (b *OrderBot) subscribeOrderEvents() error {
	if b == nil || b.orderEvents == nil {
		return nil
	}
	if err := b.orderEvents.SubscribeOrderCreated(b.handleOrderCreatedEvent); err != nil {
		return fmt.Errorf("subscribe order created: %w", err)
	}
	if err := b.orderEvents.SubscribeOrderUpdated(b.handleOrderUpdatedEvent); err != nil {
		return fmt.Errorf("subscribe order updated: %w", err)
	}
	return nil
}

func (b *OrderBot) handleOrderCreatedEvent(event app.OrderEvent) {
	ctx := context.Background()
	order := event.Order
	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	if err := b.notifyWorkshop(ctx, responses.OrderSummary(order, fromName, toName)); err != nil {
		slog.WarnContext(ctx, "notify workshop about order event failed", "topic", event.Topic, "order_number", order.Number, "error", err)
	}
}

func (b *OrderBot) handleOrderUpdatedEvent(event app.OrderEvent) {
	ctx := context.Background()
	order := event.Order
	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)
	if err := b.notifyWorkshop(ctx, responses.OrderUpdated(order, fromName, toName)); err != nil {
		slog.WarnContext(ctx, "notify workshop about order event failed", "topic", event.Topic, "order_number", order.Number, "error", err)
	}
}
