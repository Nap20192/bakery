package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"bakery/internal/app"
	orderdomain "bakery/internal/domain/order"
	authuc "bakery/internal/services/auth/usecase/auth"
	orderuc "bakery/internal/services/order/usecase/order"
	"bakery/pkg/rabbitmq/consumer"

	tele "gopkg.in/telebot.v3"
)

type baseBot struct {
	tele           *tele.Bot
	orderSvc       orderuc.UseCase
	authSvc        authuc.UseCase
	rbacSvc        *authuc.RBAC
	departmentSvc  *app.DepartmentService
	monitorSvc     *app.MonitorService
	syncSvc        *app.SyncService
	techCardSvc    *app.TechCardService
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
	orderSvc orderuc.UseCase,
	authSvc authuc.UseCase,
	rbacSvc *authuc.RBAC,
	departmentSvc *app.DepartmentService,
	monitorSvc *app.MonitorService,
	syncSvc *app.SyncService,
	techCardSvc *app.TechCardService,
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
			orderSvc:       orderSvc,
			authSvc:        authSvc,
			rbacSvc:        rbacSvc,
			departmentSvc:  departmentSvc,
			monitorSvc:     monitorSvc,
			syncSvc:        syncSvc,
			techCardSvc:    techCardSvc,
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

// ConsumeOrderEvents runs the RabbitMQ consumer that delivers order events to
// the workshop chat. Blocks until ctx is cancelled.
func (b *OrderBot) ConsumeOrderEvents(ctx context.Context) error {
	if b == nil || b.eventConsumer == nil {
		<-ctx.Done()
		return nil
	}
	return b.eventConsumer.StartConsumer(ctx, b.handleOrderEvent)
}

type orderEventEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type orderEventPayload struct {
	Order orderdomain.Order `json:"order"`
}

func (b *OrderBot) handleOrderEvent(ctx context.Context, body []byte) error {
	var env orderEventEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decode order event envelope: %w", err)
	}
	var payload orderEventPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("decode order event payload: %w", err)
	}
	order := payload.Order
	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)

	var message string
	switch env.Type {
	case orderdomain.EventOrderCreated:
		message = responses.OrderSummary(order, fromName, toName)
	case orderdomain.EventOrderUpdated:
		message = responses.OrderUpdated(order, fromName, toName)
	default:
		slog.WarnContext(ctx, "unknown order event type", "type", env.Type)
		return nil
	}
	if err := b.notifyWorkshop(ctx, message); err != nil {
		return fmt.Errorf("notify workshop: %w", err)
	}
	return nil
}
