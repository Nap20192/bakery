// Package orderoutbox publishes order domain events that were persisted to the
// transactional outbox. The relay polls unpublished rows, publishes each to the
// event bus, and stamps it published — providing at-least-once delivery
// independent of the request that produced the event.
package orderoutbox

import (
	"context"
	"log/slog"
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultInterval = 2 * time.Second
	defaultBatch    = 100
)

// Publisher publishes a stored event to the bus.
type Publisher interface {
	PublishStored(ctx context.Context, eventType string, payload []byte, createdAt time.Time, correlationID string) error
}

// Relay drains the order_outbox table on an interval.
type Relay struct {
	queries   *sqlc.Queries
	publisher Publisher
	interval  time.Duration
	batch     int32
}

func NewRelay(queries *sqlc.Queries, db *pgxpool.Pool, publisher Publisher, interval time.Duration) *Relay {
	_ = db // reserved for future SELECT ... FOR UPDATE SKIP LOCKED batching across replicas
	if interval <= 0 {
		interval = defaultInterval
	}
	return &Relay{queries: queries, publisher: publisher, interval: interval, batch: defaultBatch}
}

// Run drains the outbox until the context is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.drain(ctx); err != nil {
				slog.ErrorContext(ctx, "outbox drain failed", "component", "order.outbox", "error", err)
			}
		}
	}
}

// drain publishes every currently-unpublished event. Publish-then-mark gives
// at-least-once delivery: a crash after publish but before mark re-delivers,
// which the bus consumers tolerate.
func (r *Relay) drain(ctx context.Context) error {
	rows, err := r.queries.ListUnpublishedOrderOutboxEvents(ctx, r.batch)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := r.publisher.PublishStored(ctx, row.EventType, row.Payload, row.CreatedAt.Time, row.CorrelationID); err != nil {
			// Stop on first failure; remaining rows stay unpublished and are
			// retried on the next tick, preserving order.
			return err
		}
		if err := r.queries.MarkOrderOutboxEventPublished(ctx, row.ID); err != nil {
			return err
		}
	}
	return nil
}
