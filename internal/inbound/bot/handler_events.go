package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	orderdomain "bakery/internal/services/order/domain"
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
	default:
		slog.WarnContext(ctx, "unknown order event type", "type", env.Type)
		return nil
	}
	b.notifyOrder(ctx, order, message)
	return nil
}
