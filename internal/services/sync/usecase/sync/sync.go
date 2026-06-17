package syncuc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"bakery/internal/outbound/iiko"

	"golang.org/x/sync/errgroup"
)

type Service struct {
	client   IikoClient
	repo     Repository
	interval time.Duration
}

var _ UseCase = (*Service)(nil)

func NewService(client IikoClient, repo Repository, interval time.Duration) *Service {
	return &Service{client: client, repo: repo, interval: interval}
}

func (s *Service) Run(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.interval <= 0 {
		s.interval = 6 * time.Hour
	}
	s.syncOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.syncOnce(ctx)
		}
	}
}

func (s *Service) syncOnce(ctx context.Context) {
	start := time.Now()
	if err := s.SyncOnce(ctx); err != nil {
		slog.Error("iiko sync failed", "error", err)
		return
	}
	slog.Info("iiko sync completed", "duration", time.Since(start).Round(time.Millisecond).String())
}

func (s *Service) SyncOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	syncDate := time.Now().Format("2006-01-02")
	if err := s.client.Auth(); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	defer func() {
		if err := s.client.Logout(); err != nil {
			slog.Warn("iiko logout failed", "error", err)
		}
	}()

	var (
		catalog *iiko.NomenclatureResponse
		charts  *iiko.ChartResultDto
	)
	fetchStart := time.Now()
	group, _ := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		catalog, err = s.client.ListProductsWithCategories()
		if err != nil {
			return fmt.Errorf("list products: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		var err error
		charts, err = s.client.AssemblyChartsGetAll(syncDate, syncDate, false, true)
		if err != nil {
			return fmt.Errorf("get assembly charts: %w", err)
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return err
	}
	slog.InfoContext(ctx, "iiko sync fetched",
		"date", syncDate,
		"duration", time.Since(fetchStart).Round(time.Millisecond).String(),
		"products", len(catalog.Products),
		"assembly_charts", len(charts.AssemblyCharts),
		"prepared_charts", len(charts.PreparedCharts),
	)
	if err := s.repo.SaveSnapshot(ctx, catalog, charts, syncDate); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}
