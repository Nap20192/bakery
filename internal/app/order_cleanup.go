package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *OrderService) DeleteOrdersOlderThan(ctx context.Context, now time.Time, retention time.Duration) (int64, error) {
	if retention <= 0 {
		return 0, fmt.Errorf("order retention must be positive")
	}
	cutoff := pgtype.Timestamptz{
		Time:  now.UTC().Add(-retention),
		Valid: true,
	}
	count, err := s.queries.DeleteOrdersCreatedBefore(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old orders: %w", err)
	}
	return count, nil
}

func (s *OrderService) RunCleanupTicker(ctx context.Context, interval time.Duration, retention time.Duration) error {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if retention <= 0 {
		retention = 31 * 24 * time.Hour
	}

	run := func() {
		deleted, err := s.DeleteOrdersOlderThan(ctx, time.Now(), retention)
		if err != nil {
			slog.ErrorContext(ctx, "old orders cleanup failed", "error", err)
			return
		}
		slog.InfoContext(ctx, "old orders cleanup finished", "deleted", deleted, "retention", retention.String())
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			run()
		}
	}
}
