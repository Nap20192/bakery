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
	"github.com/jackc/pgx/v5/pgconn"
)

const productionSheetOrderUniqueConstraint = "production_sheet_orders_order_id_key"

// Журнал отработок — единственное место, где живёт факт выпечки. Лист
// фиксирует партию: сохранённый выбор заказов (production_sheet_orders) плюс
// отклонения по позициям (production_sheet_items). Заказ при отработке не
// изменяется: факт декорирует его позиции при чтении
// (GetOrderProductionFacts + decorateProductionFacts). Наружу уходят события
// produced/production_cleared — только для заказов, чей видимый факт реально
// изменился.

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
		}

		// Заказы из входа резолвятся и проверяются до любых записей — так
		// «прежний факт» можно снять одним запросом по всем затронутым.
		orderIDs := make(map[string]int64, len(input.Orders))
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
			orderIDs[order.Number] = row.ID
			affected[row.ID] = struct{}{}
		}

		before, err := productionFactsByOrder(ctx, q, affected)
		if err != nil {
			return err
		}

		if input.SheetID != 0 {
			if err := q.DeleteProductionSheetOrders(ctx, sheetID); err != nil {
				return fmt.Errorf("delete production sheet orders: %w", err)
			}
			if err := q.DeleteProductionSheetItems(ctx, sheetID); err != nil {
				return fmt.Errorf("delete production sheet items: %w", err)
			}
			if err := q.DeleteProductionSheetLoads(ctx, sheetID); err != nil {
				return fmt.Errorf("delete production sheet loads: %w", err)
			}
			if err := q.TouchProductionSheet(ctx, sheetID); err != nil {
				return fmt.Errorf("touch production sheet: %w", err)
			}
		}
		for _, order := range input.Orders {
			// Выбор партии сохраняется целиком — и для заказов без отклонений.
			if err := q.InsertProductionSheetOrder(ctx, sqlc.InsertProductionSheetOrderParams{
				SheetID: sheetID,
				OrderID: orderIDs[order.Number],
			}); err != nil {
				return productionSheetOrderInsertError(err, order.Number)
			}
			for _, item := range order.Items {
				if err := q.InsertProductionSheetLoad(ctx, sqlc.InsertProductionSheetLoadParams{
					SheetID:        sheetID,
					OrderID:        orderIDs[order.Number],
					ProductName:    item.ProductName,
					LoadedQuantity: *item.LoadedQuantity,
				}); err != nil {
					return fmt.Errorf("insert production sheet load: %w", err)
				}
				if !item.IsDeviation {
					continue
				}
				if err := q.InsertProductionSheetItem(ctx, sqlc.InsertProductionSheetItemParams{
					SheetID:          sheetID,
					OrderID:          orderIDs[order.Number],
					ProductName:      item.ProductName,
					ProducedQuantity: item.ProducedQuantity,
					Reason:           item.Reason,
				}); err != nil {
					return fmt.Errorf("insert production sheet item: %w", err)
				}
			}
		}

		return r.notifyProductionChanges(ctx, q, affected, before, input.ProducedByUsername)
	})
	if err != nil {
		return orderdomain.ProductionSheet{}, err
	}
	return r.GetProductionSheet(ctx, sheetID)
}

func productionSheetOrderInsertError(err error, orderNumber string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == productionSheetOrderUniqueConstraint {
		return apperr.Conflict("order.production_exists",
			fmt.Sprintf("По заказу %s уже есть отработка — измените её в журнале.", orderNumber))
	}
	return fmt.Errorf("insert production sheet order: %w", err)
}

func (r *OrderRepository) DeleteProductionSheet(ctx context.Context, id int64, byUsername string) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		prev, err := q.ListProductionSheetOrderIDs(ctx, id)
		if err != nil {
			return fmt.Errorf("list production sheet orders: %w", err)
		}
		affected := make(map[int64]struct{}, len(prev))
		for _, orderID := range prev {
			affected[orderID] = struct{}{}
		}
		before, err := productionFactsByOrder(ctx, q, affected)
		if err != nil {
			return err
		}
		if err := q.DeleteProductionSheet(ctx, id); err != nil {
			return fmt.Errorf("delete production sheet: %w", err)
		}
		return r.notifyProductionChanges(ctx, q, affected, before, byUsername)
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
	// Партия листа — сохранённый выбор, включая заказы без отклонений.
	orderNumbers, err := r.queries.ListProductionSheetOrderNumbers(ctx, id)
	if err != nil {
		return orderdomain.ProductionSheet{}, fmt.Errorf("list production sheet order numbers %d: %w", id, err)
	}
	sheet := orderdomain.ProductionSheet{
		ID:                row.ID,
		CreatedByUsername: row.CreatedByUsername,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
		OrderNumbers:      orderNumbers,
		Items:             make([]orderdomain.ProductionSheetItem, 0, len(itemRows)),
		ItemCount:         int64(len(itemRows)),
	}
	for _, item := range itemRows {
		sheet.Items = append(sheet.Items, orderdomain.ProductionSheetItem{
			OrderNumber:      item.OrderNumber,
			ProductName:      item.ProductName,
			LoadedQuantity:   item.LoadedQuantity,
			ProducedQuantity: item.ProducedQuantity,
			Reason:           item.Reason,
		})
	}
	return sheet, nil
}

// productionFact — видимое значение отработки одной позиции (для сравнения
// «до/после» изменения журнала).
type productionFact struct {
	Quantity float64
	Reason   string
}

// productionFactsByOrder снимает текущий факт по заказам одним запросом:
// map[orderID]map[normalized product name]fact.
func productionFactsByOrder(ctx context.Context, q sqlc.Querier, orderIDs map[int64]struct{}) (map[int64]map[string]productionFact, error) {
	ids := make([]int64, 0, len(orderIDs))
	for id := range orderIDs {
		ids = append(ids, id)
	}
	result := make(map[int64]map[string]productionFact, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := q.GetOrderProductionFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load production facts: %w", err)
	}
	for _, row := range rows {
		facts := result[row.OrderID]
		if facts == nil {
			facts = make(map[string]productionFact)
			result[row.OrderID] = facts
		}
		facts[row.ProductKey] = productionFact{Quantity: row.ProducedQuantity, Reason: row.Reason}
	}
	return result, nil
}

// notifyProductionChanges сравнивает факт до/после изменения журнала и кладёт
// события produced/production_cleared в outbox только для заказов, чей
// видимый факт изменился. Историю заказа отработка не пишет — она не является
// изменением заказа.
func (r *OrderRepository) notifyProductionChanges(
	ctx context.Context,
	q sqlc.Querier,
	affected map[int64]struct{},
	before map[int64]map[string]productionFact,
	byUsername string,
) error {
	after, err := productionFactsByOrder(ctx, q, affected)
	if err != nil {
		return err
	}
	for orderID := range affected {
		if equalFacts(before[orderID], after[orderID]) {
			continue
		}
		row, err := q.GetOrderByID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %d for production notification: %w", orderID, err)
		}
		order, err := r.hydrateOrder(ctx, q, row)
		if err != nil {
			return err
		}
		if len(after[orderID]) > 0 {
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

func equalFacts(a, b map[string]productionFact) bool {
	if len(a) != len(b) {
		return false
	}
	for key, fact := range a {
		if other, ok := b[key]; !ok || other != fact {
			return false
		}
	}
	return true
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
