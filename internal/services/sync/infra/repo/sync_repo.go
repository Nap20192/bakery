// Package syncrepo is the persistence adapter of the iiko sync service. It owns
// the sync-run lifecycle and the transactional snapshot upserts.
package syncrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/outbound/iiko"
	"bakery/internal/pkg/enum"
	"bakery/internal/pkg/helpers"
	syncuc "bakery/internal/services/sync/usecase/sync"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	emptyJSON = "{}"
	nullJSON  = "null"
)

type SyncRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

var _ syncuc.Repository = (*SyncRepository)(nil)

func New(db *pgxpool.Pool, queries *sqlc.Queries) *SyncRepository {
	return &SyncRepository{db: db, queries: queries}
}

func (r *SyncRepository) SaveSnapshot(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto, syncDate string) error {
	if r.db == nil || r.queries == nil {
		return fmt.Errorf("missing sync repository dependencies")
	}
	if catalog == nil {
		return fmt.Errorf("empty product catalog")
	}
	if charts == nil {
		return fmt.Errorf("empty assembly charts")
	}

	run, err := r.queries.CreateIikoSyncRun(ctx, sqlc.CreateIikoSyncRunParams{
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

	if err := r.saveSnapshotData(ctx, catalog, charts); err != nil {
		if finishErr := r.finishSyncRun(ctx, run.ID, int64(charts.KnownRevision), enum.SyncStatusError, err.Error()); finishErr != nil {
			slog.Warn("finish sync run failed", "error", finishErr)
		}
		return err
	}
	return r.finishSyncRun(ctx, run.ID, int64(charts.KnownRevision), enum.SyncStatusOK, "")
}

func (r *SyncRepository) finishSyncRun(ctx context.Context, runID, revision int64, status enum.SyncStatus, message string) error {
	_, err := r.queries.FinishIikoSyncRun(ctx, sqlc.FinishIikoSyncRunParams{
		KnownRevision: revision,
		Status:        string(status),
		Error:         message,
		FinishedAt:    helpers.TimestamptzNow(),
		ID:            runID,
	})
	if err != nil {
		return fmt.Errorf("finish sync run: %w", err)
	}
	return nil
}

func (r *SyncRepository) saveSnapshotData(ctx context.Context, catalog *iiko.NomenclatureResponse, charts *iiko.ChartResultDto) error {
	tx, err := r.db.Begin(ctx)
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

	q := r.queries.WithTx(tx)
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

func upsertIikoProduct(ctx context.Context, q *sqlc.Queries, product iiko.Product, updatedAt pgtype.Timestamptz) error {
	if err := q.UpsertIikoProduct(ctx, sqlc.UpsertIikoProductParams{
		ID:          product.ID.String(),
		Code:        product.Code,
		Name:        product.Name,
		Type:        optionalString(product.Type),
		MeasureUnit: product.MeasureUnit,
		RawJson:     emptyJSON,
		UpdatedAt:   updatedAt,
	}); err != nil {
		return fmt.Errorf("upsert product %s: %w", product.ID.String(), err)
	}
	return nil
}

func upsertIikoAssemblyChart(ctx context.Context, q *sqlc.Queries, chart iiko.AssemblyChartDto, updatedAt pgtype.Timestamptz) error {
	if err := q.UpsertIikoAssemblyChart(ctx, sqlc.UpsertIikoAssemblyChartParams{
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
	}); err != nil {
		return fmt.Errorf("upsert assembly chart %s: %w", chart.ID, err)
	}
	return nil
}

func insertIikoAssemblyChartItem(ctx context.Context, q *sqlc.Queries, chartID string, item iiko.AssemblyChartItem) error {
	if err := q.InsertIikoAssemblyChartItem(ctx, sqlc.InsertIikoAssemblyChartItemParams{
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
	}); err != nil {
		return fmt.Errorf("insert assembly chart item %s: %w", item.ID, err)
	}
	return nil
}

func upsertIikoPreparedChart(ctx context.Context, q *sqlc.Queries, chart iiko.PreparedChartDto, updatedAt pgtype.Timestamptz) error {
	if err := q.UpsertIikoPreparedChart(ctx, sqlc.UpsertIikoPreparedChartParams{
		ID:                                   chart.ID,
		AssembledProductID:                   chart.AssembledProductID,
		DateFrom:                             chart.DateFrom,
		DateTo:                               chart.DateTo,
		ProductSizeAssemblyStrategy:          string(chart.ProductSizeAssemblyStrategy),
		EffectiveDirectWriteoffStoreSpecJson: nullJSON,
		RawJson:                              encodeJSON(chart),
		UpdatedAt:                            updatedAt,
	}); err != nil {
		return fmt.Errorf("upsert prepared chart %s: %w", chart.ID, err)
	}
	return nil
}

func insertIikoPreparedChartItem(ctx context.Context, q *sqlc.Queries, chartID string, item iiko.PreparedChartItem) error {
	if err := q.InsertIikoPreparedChartItem(ctx, sqlc.InsertIikoPreparedChartItemParams{
		ID:                  item.ID,
		PreparedChartID:     chartID,
		SortWeight:          item.SortWeight,
		ProductID:           item.ProductID,
		ProductSizeSpecJson: nullJSON,
		StoreSpecJson:       nullJSON,
		Amount:              item.Amount,
		RawJson:             emptyJSON,
	}); err != nil {
		return fmt.Errorf("insert prepared chart item %s: %w", item.ID, err)
	}
	return nil
}

func encodeJSON(value any) string {
	if value == nil {
		return nullJSON
	}
	data, err := json.Marshal(value)
	if err != nil {
		return emptyJSON
	}
	return string(data)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
