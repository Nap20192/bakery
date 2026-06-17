package syncsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	sqlcrepo "bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"
	"bakery/internal/pkg/enum"
	"bakery/internal/pkg/helpers"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	client   *iiko.Client
	db       *pgxpool.Pool
	queries  *sqlcrepo.Queries
	interval time.Duration
}

const (
	emptyJSON = "{}"
	nullJSON  = "null"
)

func New(client *iiko.Client, db *pgxpool.Pool, queries *sqlcrepo.Queries, interval time.Duration) *Service {
	return &Service{
		client:   client,
		db:       db,
		queries:  queries,
		interval: interval,
	}
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
	syncDate := latestSyncDate()
	if err := s.client.Auth(); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	defer func() {
		if err := s.client.Logout(); err != nil {
			slog.Warn("iiko logout failed", "error", err)
		}
	}()

	var catalog *iiko.NomenclatureResponse
	var charts *iiko.ChartResultDto
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
	if err := s.SaveSnapshot(ctx, catalog, charts, syncDate); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

func (s *Service) SaveSnapshot(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto, syncDate string) error {
	if s.db == nil || s.queries == nil {
		return fmt.Errorf("missing sync repository dependencies")
	}
	if catalog == nil {
		return fmt.Errorf("empty product catalog")
	}
	if charts == nil {
		return fmt.Errorf("empty assembly charts")
	}

	run, err := s.queries.CreateIikoSyncRun(ctx, sqlcrepo.CreateIikoSyncRunParams{
		Source:        string(enum.IikoSyncSourceGetAll),
		DateFrom:      syncDate,
		DateTo:        syncDate,
		KnownRevision: int64(charts.KnownRevision),
		Status:        string(enum.SyncStatusRunning),
		Error:         "",
		StartedAt:     helpers.TimestamptzNow(),
		FinishedAt:    pgtype.Timestamptz{},
	})
	if err != nil {
		return fmt.Errorf("create sync run: %w", err)
	}

	saveStart := time.Now()
	if err := s.saveSnapshotData(ctx, catalog, charts); err != nil {
		if finishErr := s.finishSyncRun(ctx, run.ID, int64(charts.KnownRevision), enum.SyncStatusError, err.Error()); finishErr != nil {
			slog.Warn("finish sync run failed", "error", finishErr)
		}
		return err
	}
	slog.InfoContext(ctx, "iiko snapshot saved",
		"duration", time.Since(saveStart).Round(time.Millisecond).String(),
		"products", len(catalog.Products),
		"assembly_charts", len(charts.AssemblyCharts),
		"assembly_items", countAssemblyItems(charts.AssemblyCharts),
		"prepared_charts", len(charts.PreparedCharts),
		"prepared_items", countPreparedItems(charts.PreparedCharts),
	)

	return s.finishSyncRun(ctx, run.ID, int64(charts.KnownRevision), enum.SyncStatusOK, "")
}

func (s *Service) finishSyncRun(ctx context.Context, runID int64, revision int64, status enum.SyncStatus, message string) error {
	_, err := s.queries.FinishIikoSyncRun(ctx, sqlcrepo.FinishIikoSyncRunParams{
		KnownRevision: revision,
		Status:        string(status),
		Error:         message,
		FinishedAt:    helpers.TimestamptzNow(),
		ID:            runID,
	})
	return err
}

func (s *Service) saveSnapshotData(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(ctx); err != nil {
				slog.Warn("rollback iiko sync tx failed", "error", err)
			}
		}
	}()

	q := s.queries.WithTx(tx)
	updatedAt := helpers.TimestamptzNow()

	for _, product := range catalog.Products {
		if err := upsertIikoProduct(ctx, q, product, updatedAt); err != nil {
			return err
		}
	}
	for _, chart := range charts.AssemblyCharts {
		if err := upsertIikoAssemblyChart(ctx, q, chart, updatedAt); err != nil {
			return err
		}
		if err := q.DeleteIikoAssemblyChartItemsByChartID(ctx, chart.ID); err != nil {
			return fmt.Errorf("delete assembly chart items %s: %w", chart.ID, err)
		}
		for _, item := range chart.Items {
			if err := insertIikoAssemblyChartItem(ctx, q, chart.ID, item); err != nil {
				return err
			}
		}
	}
	for _, chart := range charts.PreparedCharts {
		if err := upsertIikoPreparedChart(ctx, q, chart, updatedAt); err != nil {
			return err
		}
		if err := q.DeleteIikoPreparedChartItemsByChartID(ctx, chart.ID); err != nil {
			return fmt.Errorf("delete prepared chart items %s: %w", chart.ID, err)
		}
		for _, item := range chart.Items {
			if err := insertIikoPreparedChartItem(ctx, q, chart.ID, item); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	committed = true
	return nil
}

func upsertIikoProduct(ctx context.Context, q *sqlcrepo.Queries, product iiko.Product, updatedAt pgtype.Timestamptz) error {
	err := q.UpsertIikoProduct(ctx, sqlcrepo.UpsertIikoProductParams{
		ID:          product.ID.String(),
		Code:        product.Code,
		Name:        product.Name,
		Type:        optionalString(product.Type),
		MeasureUnit: product.MeasureUnit,
		RawJson:     emptyJSON,
		UpdatedAt:   updatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert product %s: %w", product.ID.String(), err)
	}
	return nil
}

func upsertIikoAssemblyChart(ctx context.Context, q *sqlcrepo.Queries, chart iiko.AssemblyChartDto, updatedAt pgtype.Timestamptz) error {
	err := q.UpsertIikoAssemblyChart(ctx, sqlcrepo.UpsertIikoAssemblyChartParams{
		ID:                                   chart.ID,
		AssembledProductID:                   chart.AssembledProductID,
		DateFrom:                             chart.DateFrom,
		DateTo:                               chart.DateTo,
		AssembledAmount:                      chart.AssembledAmount,
		ProductWriteoffStrategy:              string(chart.ProductWriteoffStrategy),
		ProductSizeAssemblyStrategy:          string(chart.ProductSizeAssemblyStrategy),
		EffectiveDirectWriteoffStoreSpecJson: nullJSON,
		RawJson:                              encodeJSON(chart),
		UpdatedAt:                            updatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert assembly chart %s: %w", chart.ID, err)
	}
	return nil
}

func insertIikoAssemblyChartItem(ctx context.Context, q *sqlcrepo.Queries, chartID string, item iiko.AssemblyChartItem) error {
	err := q.InsertIikoAssemblyChartItem(ctx, sqlcrepo.InsertIikoAssemblyChartItemParams{
		ID:                  item.ID,
		ChartID:             chartID,
		SortWeight:          item.SortWeight,
		ProductID:           item.ProductID,
		ProductSizeSpecJson: nullJSON,
		StoreSpecJson:       nullJSON,
		AmountIn:            item.AmountIn,
		AmountMiddle:        item.AmountMiddle,
		AmountOut:           item.AmountOut,
		AmountIn1:           item.AmountIn1,
		AmountOut1:          item.AmountOut1,
		AmountIn2:           item.AmountIn2,
		AmountOut2:          item.AmountOut2,
		AmountIn3:           item.AmountIn3,
		AmountOut3:          item.AmountOut3,
		PackageCount:        item.PackageCount,
		PackageTypeID:       item.PackageTypeID,
		RawJson:             emptyJSON,
	})
	if err != nil {
		return fmt.Errorf("insert assembly chart item %s: %w", item.ID, err)
	}
	return nil
}

func upsertIikoPreparedChart(ctx context.Context, q *sqlcrepo.Queries, chart iiko.PreparedChartDto, updatedAt pgtype.Timestamptz) error {
	err := q.UpsertIikoPreparedChart(ctx, sqlcrepo.UpsertIikoPreparedChartParams{
		ID:                                   chart.ID,
		AssembledProductID:                   chart.AssembledProductID,
		DateFrom:                             chart.DateFrom,
		DateTo:                               chart.DateTo,
		ProductSizeAssemblyStrategy:          string(chart.ProductSizeAssemblyStrategy),
		EffectiveDirectWriteoffStoreSpecJson: nullJSON,
		RawJson:                              encodeJSON(chart),
		UpdatedAt:                            updatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert prepared chart %s: %w", chart.ID, err)
	}
	return nil
}

func insertIikoPreparedChartItem(ctx context.Context, q *sqlcrepo.Queries, chartID string, item iiko.PreparedChartItem) error {
	err := q.InsertIikoPreparedChartItem(ctx, sqlcrepo.InsertIikoPreparedChartItemParams{
		ID:                  item.ID,
		PreparedChartID:     chartID,
		SortWeight:          item.SortWeight,
		ProductID:           item.ProductID,
		ProductSizeSpecJson: nullJSON,
		StoreSpecJson:       nullJSON,
		Amount:              item.Amount,
		RawJson:             emptyJSON,
	})
	if err != nil {
		return fmt.Errorf("insert prepared chart item %s: %w", item.ID, err)
	}
	return nil
}

func encodeJSON(value any) string {
	if value == nil {
		return "null"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func countAssemblyItems(charts []iiko.AssemblyChartDto) int {
	count := 0
	for _, chart := range charts {
		count += len(chart.Items)
	}
	return count
}

func countPreparedItems(charts []iiko.PreparedChartDto) int {
	count := 0
	for _, chart := range charts {
		count += len(chart.Items)
	}
	return count
}

func latestSyncDate() string {
	return time.Now().Format("2006-01-02")
}
