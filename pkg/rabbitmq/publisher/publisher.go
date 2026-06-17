package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"bakery/internal/pkg/enum"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	publishMandatory = false
	publishImmediate = false
	bindingKey       = ""
	messageTypeName  = "domain.event"

	exchangeDurable    = true
	exchangeAutoDelete = false
	exchangeInternal   = false
	exchangeNoWait     = false
)

// Publisher publishes domain events to the order-events fanout exchange.
type Publisher struct {
	exchangeName string
	exchangeKind string
	bindingKey   string
	amqpConn     *amqp.Connection
}

func New(amqpConn *amqp.Connection) *Publisher {
	return &Publisher{
		amqpConn:     amqpConn,
		exchangeName: enum.ExchangeOrderEvents.String(),
		exchangeKind: enum.ExchangeKindFanout.String(),
		bindingKey:   bindingKey,
	}
}

// PublishEvents marshals each event into a transport envelope and publishes it.
func (p *Publisher) PublishEvents(ctx context.Context, events []any) error {
	slog.DebugContext(ctx, "rabbitmq publish batch",
		"component", "rabbitmq.publisher",
		"exchange", p.exchangeName,
		"events", len(events),
	)
	for _, e := range events {
		identity := ""
		if ev, ok := e.(interface{ Identity() string }); ok {
			identity = ev.Identity()
		}
		body, err := json.Marshal(envelope(e))
		if err != nil {
			return fmt.Errorf("marshal event envelope: %w", err)
		}
		slog.DebugContext(ctx, "rabbitmq publishing event",
			"component", "rabbitmq.publisher",
			"exchange", p.exchangeName,
			"type", identity,
			"bytes", len(body),
			"body", string(body),
		)
		if err := p.publish(ctx, body); err != nil {
			return err
		}
		slog.DebugContext(ctx, "rabbitmq event published",
			"component", "rabbitmq.publisher",
			"exchange", p.exchangeName,
			"type", identity,
		)
	}
	return nil
}

func envelope(event any) map[string]any {
	identity := ""
	createdAt := time.Now().UTC()
	if e, ok := event.(interface {
		Identity() string
		CreatedAt() time.Time
	}); ok {
		identity = e.Identity()
		createdAt = e.CreatedAt()
	}
	return map[string]any{
		"type":       identity,
		"created_at": createdAt,
		"payload":    event,
	}
}

func (p *Publisher) publish(ctx context.Context, body []byte) error {
	ch, err := p.amqpConn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() {
		if err := ch.Close(); err != nil {
			slog.WarnContext(ctx, "close rabbitmq publish channel failed", "error", err)
		}
	}()

	if err := ch.ExchangeDeclare(
		p.exchangeName, p.exchangeKind,
		exchangeDurable, exchangeAutoDelete, exchangeInternal, exchangeNoWait, nil,
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	messageID := uuid.New().String()
	if err := ch.PublishWithContext(
		ctx, p.exchangeName, p.bindingKey, publishMandatory, publishImmediate,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    messageID,
			Timestamp:    time.Now(),
			Body:         body,
			Type:         messageTypeName,
		},
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	slog.DebugContext(ctx, "rabbitmq message sent to exchange",
		"component", "rabbitmq.publisher",
		"exchange", p.exchangeName,
		"routing_key", p.bindingKey,
		"message_id", messageID,
		"bytes", len(body),
	)
	return nil
}
