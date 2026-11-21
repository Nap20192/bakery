package orderoutbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/correlation"
)

const (
	defaultInterval = 2 * time.Second
	defaultBatch    = 100
)

type Handler interface {
	HandleOrderEvent(ctx context.Context, eventType string, payload []byte) error
}

type Relay struct {
	queries  *sqlc.Queries
	handler  Handler
	interval time.Duration
	batch    int64
}

func NewRelay(queries *sqlc.Queries, handler Handler, interval time.Duration) *Relay {
	if interval <= 0 {
		interval = defaultInterval
	}
	return &Relay{queries: queries, handler: handler, interval: interval, batch: defaultBatch}
}

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

func (r *Relay) drain(ctx context.Context) error {
	rows, err := r.queries.ListUnpublishedOrderOutboxEvents(ctx, r.batch)
	if err != nil {
		return fmt.Errorf("list unpublished outbox events: %w", err)
	}
	for _, row := range rows {
		eventCtx := correlation.WithID(ctx, row.CorrelationID)
		if err := r.handler.HandleOrderEvent(eventCtx, row.EventType, []byte(row.Payload)); err != nil {
			return fmt.Errorf("handle outbox event %d: %w", row.ID, err)
		}
		if err := r.queries.MarkOrderOutboxEventPublished(ctx, row.ID); err != nil {
			return fmt.Errorf("mark outbox event %d published: %w", row.ID, err)
		}
	}
	return nil
}
