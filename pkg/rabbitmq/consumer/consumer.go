package consumer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"bakery/internal/pkg/enum"
	"bakery/pkg/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeKind       = "fanout"
	exchangeDurable    = true
	exchangeAutoDelete = false
	exchangeInternal   = false
	exchangeNoWait     = false

	queueDurable    = true
	queueAutoDelete = false
	queueExclusive  = false
	queueNoWait     = false

	prefetchCount  = 5
	prefetchSize   = 0
	prefetchGlobal = false

	consumeAutoAck   = false
	consumeExclusive = false
	consumeNoLocal   = false
	consumeNoWait    = false

	bindingKey     = ""
	consumerTag    = "bakery-bot"
	workerPoolSize = 8
)

// Handler processes a raw message body. Returning an error nacks the message.
type Handler func(ctx context.Context, body []byte) error

// Consumer subscribes to the order-events exchange via a durable queue.
type Consumer struct {
	exchangeName   string
	queueName      string
	bindingKey     string
	consumerTag    string
	workerPoolSize int
	conn           *rabbitmq.Conn
}

func New(conn *rabbitmq.Conn) *Consumer {
	return &Consumer{
		conn:           conn,
		exchangeName:   enum.ExchangeOrderEvents.String(),
		queueName:      enum.QueueBotOrderEvents.String(),
		bindingKey:     bindingKey,
		consumerTag:    consumerTag,
		workerPoolSize: workerPoolSize,
	}
}

// StartConsumer runs until ctx is cancelled or the channel closes.
func (c *Consumer) StartConsumer(ctx context.Context, handler Handler) error {
	if handler == nil {
		return fmt.Errorf("rabbitmq consumer handler is nil")
	}
	ch, err := c.createChannel()
	if err != nil {
		return err
	}
	defer func() {
		if err := ch.Close(); err != nil {
			slog.WarnContext(ctx, "close rabbitmq consume channel failed", "error", err)
		}
	}()

	deliveries, err := ch.Consume(
		c.queueName, c.consumerTag,
		consumeAutoAck, consumeExclusive, consumeNoLocal, consumeNoWait, nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	notifyClose := ch.NotifyClose(make(chan *amqp.Error, 1))
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var ackMu sync.Mutex
	for i := 0; i < c.workerPoolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.consume(workerCtx, deliveries, handler, &ackMu)
		}()
	}

	select {
	case <-ctx.Done():
		cancel()
		ackMu.Lock()
		if err := ch.Cancel(c.consumerTag, false); err != nil {
			slog.WarnContext(ctx, "cancel rabbitmq consumer failed", "error", err)
		}
		ackMu.Unlock()
		wg.Wait()
		return nil
	case chanErr := <-notifyClose:
		cancel()
		wg.Wait()
		if chanErr == nil {
			return nil
		}
		return chanErr
	}
}

func (c *Consumer) consume(ctx context.Context, deliveries <-chan amqp.Delivery, handler Handler, ackMu *sync.Mutex) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}
			slog.DebugContext(ctx, "rabbitmq message received",
				"component", "rabbitmq.consumer",
				"queue", c.queueName,
				"message_id", delivery.MessageId,
				"type", delivery.Type,
				"bytes", len(delivery.Body),
				"body", string(delivery.Body),
			)
			if err := handler(ctx, delivery.Body); err != nil {
				ackMu.Lock()
				if nackErr := delivery.Nack(false, false); nackErr != nil {
					slog.WarnContext(ctx, "nack rabbitmq message failed", "error", nackErr)
				}
				ackMu.Unlock()
				slog.WarnContext(ctx, "rabbitmq message handler failed", "error", err)
				continue
			}
			ackMu.Lock()
			if err := delivery.Ack(false); err != nil {
				slog.WarnContext(ctx, "ack rabbitmq message failed", "error", err)
			} else {
				slog.DebugContext(ctx, "rabbitmq message acked",
					"component", "rabbitmq.consumer",
					"queue", c.queueName,
					"message_id", delivery.MessageId,
				)
			}
			ackMu.Unlock()
		}
	}
}

func (c *Consumer) createChannel() (*amqp.Channel, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	if err := ch.ExchangeDeclare(
		c.exchangeName, exchangeKind,
		exchangeDurable, exchangeAutoDelete, exchangeInternal, exchangeNoWait, nil,
	); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}
	queue, err := ch.QueueDeclare(
		c.queueName, queueDurable, queueAutoDelete, queueExclusive, queueNoWait, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(queue.Name, c.bindingKey, c.exchangeName, queueNoWait, nil); err != nil {
		return nil, fmt.Errorf("bind queue: %w", err)
	}
	if err := ch.Qos(prefetchCount, prefetchSize, prefetchGlobal); err != nil {
		return nil, fmt.Errorf("qos: %w", err)
	}
	slog.Info("rabbitmq consumer ready", "queue", queue.Name, "exchange", c.exchangeName)
	return ch, nil
}
