package orderrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/apperr"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"

	"github.com/jackc/pgx/v5"
)

// Журнал отработок. Документ хранит факт выпечки по позициям заказов;
// produced_quantity в самих заказах — проекция журнала (последний лист
// побеждает), пересчитываемая здесь при каждом изменении.

func (r *OrderRepository) SaveProductionSheet(ctx context.Context, input orderuc.SaveProductionSheetInput) (orderdomain.ProductionSheet, error) {
	var sheetID int64
	err := r.withTx(ctx, func(q sqlc.Querier) error {
		affected := make(map[int64]struct{})

		if input.SheetID == 0 {
			row, err := q.CreateProductionSheet(ctx, input.ProducedByUsername)
			if err != nil {
				return fmt.Errorf("create production sheet: %w", err)
			}
			sheetID = row.ID
		} else {
			sheetID = input.SheetID
			prev, err := q.ListProductionSheetOrderIDs(ctx, sheetID)
			if err != nil {
				return fmt.Errorf("list production sheet orders: %w", err)
			}
			for _, id := range prev {
				affected[id] = struct{}{}
			}
			if err := q.DeleteProductionSheetItems(ctx, sheetID); err != nil {
				return fmt.Errorf("delete production sheet items: %w", err)
			}
			if err := q.TouchProductionSheet(ctx, sheetID); err != nil {
				return fmt.Errorf("touch production sheet: %w", err)
			}
		}

		for _, order := range input.Orders {
			row, err := q.GetOrderByNumber(ctx, strings.TrimSpace(order.Number))
			if err != nil {
				return orderuc.ErrProductionOrderNotFound
			}
			// Заказ принадлежит максимум одному листу: пересечение с чужим
			// документом — конфликт, пусть правят существующую отработку.
			existingSheet, err := q.GetOrderProductionSheetID(ctx, row.ID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("check order production sheet: %w", err)
			}
			if err == nil && existingSheet != sheetID {
				return apperr.Conflict("order.production_exists",
					fmt.Sprintf("По заказу %s уже есть отработка №%d — измените её в журнале.", row.Number, existingSheet))
			}
			affected[row.ID] = struct{}{}
			for _, item := range order.Items {
				if err := q.InsertProductionSheetItem(ctx, sqlc.InsertProductionSheetItemParams{
					SheetID:          sheetID,
					OrderID:          row.ID,
					ProductName:      item.ProductName,
					ProducedQuantity: item.ProducedQuantity,
					Reason:           item.Reason,
				}); err != nil {
					return fmt.Errorf("insert production sheet item: %w", err)
				}
			}
		}

		return r.syncOrdersProduction(ctx, q, affected, input.ProducedByUsername)
	})
	if err != nil {
		return orderdomain.ProductionSheet{}, err
	}
	return r.GetProductionSheet(ctx, sheetID)
}

func (r *OrderRepository) DeleteProductionSheet(ctx context.Context, id int64, byUsername string) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		prev, err := q.ListProductionSheetOrderIDs(ctx, id)
		if err != nil {
			return fmt.Errorf("list production sheet orders: %w", err)
		}
		if err := q.DeleteProductionSheet(ctx, id); err != nil {
			return fmt.Errorf("delete production sheet: %w", err)
		}
		affected := make(map[int64]struct{}, len(prev))
		for _, orderID := range prev {
			affected[orderID] = struct{}{}
		}
		return r.syncOrdersProduction(ctx, q, affected, byUsername)
	})
}

func (r *OrderRepository) ListProductionSheets(ctx context.Context) ([]orderdomain.ProductionSheet, error) {
	rows, err := r.queries.ListProductionSheets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list production sheets: %w", err)
	}
	sheets := make([]orderdomain.ProductionSheet, 0, len(rows))
	for _, row := range rows {
		sheets = append(sheets, orderdomain.ProductionSheet{
			ID:                row.ID,
			CreatedByUsername: row.CreatedByUsername,
			CreatedAt:         row.CreatedAt.Time,
			UpdatedAt:         row.UpdatedAt.Time,
			OrderNumbers:      row.OrderNumbers,
			ItemCount:         row.ItemCount,
		})
	}
	return sheets, nil
}

func (r *OrderRepository) GetProductionSheet(ctx context.Context, id int64) (orderdomain.ProductionSheet, error) {
	row, err := r.queries.GetProductionSheet(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return orderdomain.ProductionSheet{}, orderuc.ErrProductionSheetNotFound
		}
		return orderdomain.ProductionSheet{}, fmt.Errorf("get production sheet %d: %w", id, err)
	}
	itemRows, err := r.queries.ListProductionSheetItems(ctx, id)
	if err != nil {
		return orderdomain.ProductionSheet{}, fmt.Errorf("list production sheet items %d: %w", id, err)
	}
	sheet := orderdomain.ProductionSheet{
		ID:                row.ID,
		CreatedByUsername: row.CreatedByUsername,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
		Items:             make([]orderdomain.ProductionSheetItem, 0, len(itemRows)),
		ItemCount:         int64(len(itemRows)),
	}
	seenOrders := make(map[string]struct{})
	for _, item := range itemRows {
		sheet.Items = append(sheet.Items, orderdomain.ProductionSheetItem{
			OrderNumber:      item.OrderNumber,
			ProductName:      item.ProductName,
			ProducedQuantity: item.ProducedQuantity,
			Reason:           item.Reason,
		})
		if _, ok := seenOrders[item.OrderNumber]; !ok {
			seenOrders[item.OrderNumber] = struct{}{}
			sheet.OrderNumbers = append(sheet.OrderNumbers, item.OrderNumber)
		}
	}
	return sheet, nil
}

// syncOrdersProduction перепроецирует журнал на затронутые заказы: обновляет
// produced_quantity, пишет диф в историю и кладёт события produced/cleared в
// outbox. Заказы без фактических изменений пропускаются.
func (r *OrderRepository) syncOrdersProduction(ctx context.Context, q sqlc.Querier, orderIDs map[int64]struct{}, byUsername string) error {
	for orderID := range orderIDs {
		row, err := q.GetOrderByID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %d for production sync: %w", orderID, err)
		}
		before, err := q.GetOrderItemsByOrderID(ctx, orderID)
		if err != nil {
			return err
		}
		if err := q.ApplyOrderProduction(ctx, orderID); err != nil {
			return fmt.Errorf("apply order production %d: %w", orderID, err)
		}
		if err := q.ClearUncoveredOrderProduction(ctx, orderID); err != nil {
			return fmt.Errorf("clear uncovered production %d: %w", orderID, err)
		}
		after, err := q.GetOrderItemsByOrderID(ctx, orderID)
		if err != nil {
			return err
		}

		historyItems := producedDiff(before, after)
		if len(historyItems) == 0 {
			continue
		}
		if err := r.createOrderHistory(ctx, q, orderID, byUsername, historyItems); err != nil {
			return err
		}

		order, err := r.hydrateOrder(ctx, q, row)
		if err != nil {
			return err
		}
		hasProduction := false
		for _, item := range order.Items {
			if item.ProducedQuantity != nil {
				hasProduction = true
				break
			}
		}
		if hasProduction {
			order.RecordProduced(byUsername)
		} else {
			order.RecordProductionCleared(byUsername)
		}
		if err := r.persistOutbox(ctx, q, order.Number, order.PullDomainEvents()); err != nil {
			return err
		}
	}
	return nil
}

// producedDiff собирает записи истории по изменившемуся факту выпечки.
func producedDiff(before, after []sqlc.GetOrderItemsByOrderIDRow) []orderdomain.OrderHistoryItem {
	prev := make(map[int64]*float64, len(before))
	for _, item := range before {
		prev[item.ID] = item.ProducedQuantity
	}
	diff := make([]orderdomain.OrderHistoryItem, 0)
	for _, item := range after {
		old := prev[item.ID]
		if equalProduced(old, item.ProducedQuantity) {
			continue
		}
		diff = append(diff, orderdomain.OrderHistoryItem{
			ChangeType:  orderdomain.ChangeTypeProduced,
			ProductCode: item.ProductCode,
			ProductName: item.ProductName,
			OldQuantity: old,
			NewQuantity: item.ProducedQuantity,
		})
	}
	return diff
}

func equalProduced(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// withTx runs fn inside a transaction (or directly without a pool, as in
// tests), committing on success.
func (r *OrderRepository) withTx(ctx context.Context, fn func(q sqlc.Querier) error) error {
	if r.db == nil {
		return fn(r.queries)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := fn(r.queries.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	committed = true
	return nil
}
