package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"bakery/internal/pkg/enum"
	authuc "bakery/internal/services/auth/usecase/auth"
	orderdomain "bakery/internal/services/order/domain"

	tele "gopkg.in/telebot.v3"
)

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
	slog.DebugContext(ctx, "bot received order event",
		"component", "bot.consumer",
		"type", env.Type,
		"order_number", order.Number,
		"created_by", order.CreatedByUsername,
	)
	fromName := b.departmentDisplayName(ctx, order.FromDepartmentID)
	toName := b.departmentDisplayName(ctx, order.ToDepartmentID)

	var message string
	switch env.Type {
	case orderdomain.EventOrderCreated:
		message = responses.OrderSummary(order, fromName, toName)
	case orderdomain.EventOrderUpdated:
		message = responses.OrderUpdated(order, fromName, toName)
	case orderdomain.EventOrderCancelled:
		message = responses.OrderCancelled(order, fromName, toName)
	case orderdomain.EventOrderRestored:
		message = responses.OrderRestored(order, fromName, toName)
	default:
		slog.WarnContext(ctx, "unknown order event type", "type", env.Type)
		return nil
	}
	b.notifyOrder(ctx, order, message)
	return nil
}

func (b *OrderBot) departmentDisplayName(ctx context.Context, departmentID *int64) string {
	if departmentID == nil {
		return ""
	}
	department, err := b.departmentSvc.GetByID(ctx, *departmentID)
	if err != nil {
		slog.WarnContext(ctx, "get department display name failed", "department_id", *departmentID, "error", err)
		return fmt.Sprintf("#%d", *departmentID)
	}
	return department.Name
}

// notifyOrder delivers an order notification to the order's creator, every
// baker, and the workshop group chat. Best-effort: per-recipient failures are
// logged, not propagated (a failed send must not requeue the event).
func (b *OrderBot) notifyOrder(ctx context.Context, order orderdomain.Order, message string) {
	recipients := make(map[int64]struct{})

	if username := strings.TrimSpace(order.CreatedByUsername); username != "" {
		creator, err := b.authSvc.GetUserByTelegramUsername(ctx, username)
		switch {
		case err != nil && !errors.Is(err, authuc.ErrAuthUserNotFound):
			slog.WarnContext(ctx, "resolve order creator failed", "username", username, "error", err)
		case err != nil:
			slog.DebugContext(ctx, "order creator not registered, skipping DM", "component", "bot.notify", "username", username)
		case creator.TelegramID != nil:
			recipients[*creator.TelegramID] = struct{}{}
		}
	}

	bakers, err := b.authSvc.ListUsersByRole(ctx, string(enum.RoleBaker))
	if err != nil {
		slog.WarnContext(ctx, "list bakers for order notification failed", "error", err)
	}
	for _, baker := range bakers {
		if baker.TelegramID != nil {
			recipients[*baker.TelegramID] = struct{}{}
		}
	}

	// DMs are private chats — a WebApp button opens the order inside Telegram.
	dmMarkup := b.orderWebAppMarkup(order.Number)
	for telegramID := range recipients {
		if err := b.sendHTMLToChat(telegramID, message, dmMarkup); err != nil {
			slog.WarnContext(ctx, "send order notification failed", "telegram_id", telegramID, "error", err)
		}
	}

	// Telegram forbids WebApp buttons in groups, so the workshop chat gets a
	// plain URL button instead.
	if b.workshopChatID != 0 {
		if err := b.sendHTMLToChat(b.workshopChatID, message, b.orderURLMarkup(order.Number)); err != nil {
			slog.WarnContext(ctx, "send order notification to group failed", "chat_id", b.workshopChatID, "error", err)
		}
	}
}

// orderAppURL builds the mini app URL pointing at a specific order.
func (b *OrderBot) orderAppURL(orderNumber string) string {
	base := strings.TrimSpace(b.miniAppURL)
	if base == "" {
		return ""
	}
	link, err := url.Parse(base)
	if err != nil {
		return ""
	}
	query := link.Query()
	query.Set("order", strings.TrimSpace(orderNumber))
	link.RawQuery = query.Encode()
	return link.String()
}

// orderWebAppMarkup is an inline WebApp button (private chats only).
func (b *OrderBot) orderWebAppMarkup(orderNumber string) *tele.ReplyMarkup {
	link := b.orderAppURL(orderNumber)
	if link == "" {
		return nil
	}
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.WebApp("🥐 Открыть заказ", &tele.WebApp{URL: link})))
	return markup
}

// orderURLMarkup is an inline URL button (works in group chats).
func (b *OrderBot) orderURLMarkup(orderNumber string) *tele.ReplyMarkup {
	link := b.orderAppURL(orderNumber)
	if link == "" {
		return nil
	}
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.URL("🥐 Открыть заказ", link)))
	return markup
}
