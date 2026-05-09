package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	sqlcrepo "bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SyncService struct {
	client   *iiko.Client
	db       *pgxpool.Pool
	queries  *sqlcrepo.Queries
	interval time.Duration
}

const (
	iikoSyncSource  = "getAll"
	syncStatusRun   = "running"
	syncStatusOK    = "ok"
	syncStatusError = "error"
)

func NewSyncService(client *iiko.Client, db *pgxpool.Pool, queries *sqlcrepo.Queries, interval time.Duration) *SyncService {
	return &SyncService{
		client:   client,
		db:       db,
		queries:  queries,
		interval: interval,
	}
}

func (s *SyncService) Run(ctx context.Context) error {
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

func (s *SyncService) syncOnce(ctx context.Context) {
	start := time.Now()
	if err := s.SyncOnce(ctx); err != nil {
		slog.Error("iiko sync failed", "error", err)
		return
	}
	slog.Info("iiko sync completed", "duration", time.Since(start).Round(time.Millisecond).String())
}

func (s *SyncService) SyncOnce(ctx context.Context) error {
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

	catalog, err := s.client.ListProductsWithCategories()
	if err != nil {
		return fmt.Errorf("list products: %w", err)
	}
	charts, err := s.client.AssemblyChartsGetAll(syncDate, syncDate, false, true)
	if err != nil {
		return fmt.Errorf("get assembly charts: %w", err)
	}
	if err := s.SaveSnapshot(ctx, catalog, charts, syncDate); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

func (s *SyncService) SaveSnapshot(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto, syncDate string) error {
	if s.db == nil || s.queries == nil {
		return fmt.Errorf("missing sync repository dependencies")
	}
	if catalog == nil {
		return fmt.Errorf("empty product catalog")
	}
	if charts == nil {
		return fmt.Errorf("empty assembly charts")
	}

	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	run, err := s.queries.CreateIikoSyncRun(ctx, sqlcrepo.CreateIikoSyncRunParams{
		Source:        iikoSyncSource,
		DateFrom:      syncDate,
		DateTo:        syncDate,
		KnownRevision: int64(charts.KnownRevision),
		Status:        syncStatusRun,
		Error:         "",
		StartedAt:     startedAt,
		FinishedAt:    nil,
	})
	if err != nil {
		return fmt.Errorf("create sync run: %w", err)
	}

	if err := s.saveSnapshotData(ctx, catalog, charts); err != nil {
		if finishErr := s.finishSyncRun(ctx, run.ID, int64(charts.KnownRevision), syncStatusError, err.Error()); finishErr != nil {
			slog.Warn("finish sync run failed", "error", finishErr)
		}
		return err
	}

	return s.finishSyncRun(ctx, run.ID, int64(charts.KnownRevision), syncStatusOK, "")
}

func (s *SyncService) finishSyncRun(ctx context.Context, runID int64, revision int64, status string, message string) error {
	finishedAt := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.queries.FinishIikoSyncRun(ctx, sqlcrepo.FinishIikoSyncRunParams{
		KnownRevision: revision,
		Status:        status,
		Error:         message,
		FinishedAt:    &finishedAt,
		ID:            runID,
	})
	return err
}

func (s *SyncService) saveSnapshotData(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto) error {
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
	updatedAt := time.Now().UTC().Format(time.RFC3339Nano)

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

func upsertIikoProduct(ctx context.Context, q *sqlcrepo.Queries, product iiko.Product, updatedAt string) error {
	_, err := q.UpsertIikoProduct(ctx, sqlcrepo.UpsertIikoProductParams{
		ID:          product.ID.String(),
		Code:        product.Code,
		Name:        product.Name,
		Type:        optionalString(product.Type),
		MeasureUnit: product.MeasureUnit,
		RawJson:     encodeJSON(product),
		UpdatedAt:   updatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert product %s: %w", product.ID.String(), err)
	}
	return nil
}

func upsertIikoAssemblyChart(ctx context.Context, q *sqlcrepo.Queries, chart iiko.AssemblyChartDto, updatedAt string) error {
	_, err := q.UpsertIikoAssemblyChart(ctx, sqlcrepo.UpsertIikoAssemblyChartParams{
		ID:                                   chart.ID,
		AssembledProductID:                   chart.AssembledProductID,
		DateFrom:                             chart.DateFrom,
		DateTo:                               chart.DateTo,
		AssembledAmount:                      chart.AssembledAmount,
		ProductWriteoffStrategy:              string(chart.ProductWriteoffStrategy),
		ProductSizeAssemblyStrategy:          string(chart.ProductSizeAssemblyStrategy),
		EffectiveDirectWriteoffStoreSpecJson: encodeJSON(chart.EffectiveDirectWriteoffStoreSpecification),
		RawJson:                              encodeJSON(chart),
		UpdatedAt:                            updatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert assembly chart %s: %w", chart.ID, err)
	}
	return nil
}

func insertIikoAssemblyChartItem(ctx context.Context, q *sqlcrepo.Queries, chartID string, item iiko.AssemblyChartItem) error {
	_, err := q.InsertIikoAssemblyChartItem(ctx, sqlcrepo.InsertIikoAssemblyChartItemParams{
		ID:                  item.ID,
		ChartID:             chartID,
		SortWeight:          item.SortWeight,
		ProductID:           item.ProductID,
		ProductSizeSpecJson: encodeJSON(item.ProductSizeSpecification),
		StoreSpecJson:       encodeJSON(item.StoreSpecification),
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
		RawJson:             encodeJSON(item),
	})
	if err != nil {
		return fmt.Errorf("insert assembly chart item %s: %w", item.ID, err)
	}
	return nil
}

func upsertIikoPreparedChart(ctx context.Context, q *sqlcrepo.Queries, chart iiko.PreparedChartDto, updatedAt string) error {
	_, err := q.UpsertIikoPreparedChart(ctx, sqlcrepo.UpsertIikoPreparedChartParams{
		ID:                                   chart.ID,
		AssembledProductID:                   chart.AssembledProductID,
		DateFrom:                             chart.DateFrom,
		DateTo:                               chart.DateTo,
		ProductSizeAssemblyStrategy:          string(chart.ProductSizeAssemblyStrategy),
		EffectiveDirectWriteoffStoreSpecJson: encodeJSON(chart.EffectiveDirectWriteoffStoreSpecification),
		RawJson:                              encodeJSON(chart),
		UpdatedAt:                            updatedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert prepared chart %s: %w", chart.ID, err)
	}
	return nil
}

func insertIikoPreparedChartItem(ctx context.Context, q *sqlcrepo.Queries, chartID string, item iiko.PreparedChartItem) error {
	_, err := q.InsertIikoPreparedChartItem(ctx, sqlcrepo.InsertIikoPreparedChartItemParams{
		ID:                  item.ID,
		PreparedChartID:     chartID,
		SortWeight:          item.SortWeight,
		ProductID:           item.ProductID,
		ProductSizeSpecJson: encodeJSON(item.ProductSizeSpecification),
		StoreSpecJson:       encodeJSON(item.StoreSpecification),
		Amount:              item.Amount,
		RawJson:             encodeJSON(item),
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

func latestSyncDate() string {
	return time.Now().Format("2006-01-02")
}
