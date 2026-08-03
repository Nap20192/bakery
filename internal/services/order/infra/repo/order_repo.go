// Package orderrepo is the persistence adapter of the order service. It
// implements the orderuc.Repository port over sqlc + pgx and contains all SQL
// and transaction handling. The use-case layer depends on the port, not on
// this package (dependency inversion); the composition root binds them.
package orderrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/apperr"
	"bakery/internal/pkg/correlation"
	"bakery/internal/pkg/helpers"
	sharedkernel "bakery/internal/pkg/sharedkernel"
	orderdomain "bakery/internal/services/order/domain"
	orderuc "bakery/internal/services/order/usecase/order"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderRepository persists orders, history and the dish catalog.
type OrderRepository struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
	domain  *orderdomain.OrderService
}

var _ orderuc.Repository = (*OrderRepository)(nil)

func New(queries *sqlc.Queries, db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		queries: queries,
		db:      db,
		domain:  orderdomain.NewOrderService(),
	}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, input orderuc.CreateOrderRepositoryInput) (orderdomain.Order, error) {
	if r.db == nil {
		return r.createOrderWithQueries(ctx, r.queries, input)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("begin order tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	order, err := r.createOrderWithQueries(ctx, r.queries.WithTx(tx), input)
	if err != nil {
		return orderdomain.Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return orderdomain.Order{}, fmt.Errorf("commit order tx: %w", err)
	}
	committed = true
	return order, nil
}

func (r *OrderRepository) createOrderWithQueries(ctx context.Context, q sqlc.Querier, input orderuc.CreateOrderRepositoryInput) (orderdomain.Order, error) {
	if err := q.CreateOrderCounterDay(ctx, sqlc.CreateOrderCounterDayParams{Day: input.CounterDay, DepartmentID: input.Source.ID, CategoryID: input.Category.ID}); err != nil {
		return orderdomain.Order{}, fmt.Errorf("init order counter: %w", err)
	}
	counter, err := q.NextOrderCounter(ctx, sqlc.NextOrderCounterParams{Day: input.CounterDay, DepartmentID: input.Source.ID, CategoryID: input.Category.ID})
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("increment order counter: %w", err)
	}

	number := r.domain.BuildOrderNumber(input.Source.Code, input.Source.Name, input.Category.Letter, input.CreatedAt, counter)
	row, err := q.CreateOrder(ctx, sqlc.CreateOrderParams{
		Number:            number,
		Location:          input.Input.Location,
		FromDepartmentID:  input.Input.FromDepartmentID,
		ToDepartmentID:    input.Input.ToDepartmentID,
		CategoryID:        &input.Category.ID,
		CreatedAt:         helpers.Timestamptz(input.CreatedAt),
		FulfillmentDate:   helpers.DateOf(input.FulfillmentDate),
		CreatedByUsername: strings.TrimSpace(input.Input.CreatedByUsername),
		Comments:          marshalComments(input.Input.Comments),
	})
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("create order: %w", err)
	}

	if err := r.createOrderItems(ctx, q, row.ID, input.Input.Items); err != nil {
		return orderdomain.Order{}, err
	}

	order := orderFromRow(row, input.Input.Items, nil)
	category := input.Category
	order.Category = &category
	order.RecordCreated()
	if err := r.persistOutbox(ctx, q, order.Number, order.PullDomainEvents()); err != nil {
		return orderdomain.Order{}, err
	}
	return order, nil
}

// persistOutbox writes the aggregate's domain events to the order_outbox table
// inside the caller's transaction, so events are committed atomically with the
// order change. A relay later publishes them to RabbitMQ.
func (r *OrderRepository) persistOutbox(ctx context.Context, q sqlc.Querier, aggregateID string, events []sharedkernel.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}
	correlationID := correlation.FromContext(ctx)
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal outbox event: %w", err)
		}
		if _, err := q.InsertOrderOutboxEvent(ctx, sqlc.InsertOrderOutboxEventParams{
			AggregateID:   aggregateID,
			EventType:     event.Identity(),
			Payload:       payload,
			CorrelationID: correlationID,
		}); err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
	}
	return nil
}

func (r *OrderRepository) UpdateOrder(ctx context.Context, input orderuc.UpdateOrderRepositoryInput) (orderdomain.Order, error) {
	if r.db == nil {
		return r.updateOrderWithQueries(ctx, r.queries, input)
	}
	return r.updateOrderTx(ctx, input)
}

func (r *OrderRepository) updateOrderTx(ctx context.Context, input orderuc.UpdateOrderRepositoryInput) (orderdomain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("begin order tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	order, err := r.updateOrderWithQueries(ctx, r.queries.WithTx(tx), input)
	if err != nil {
		return orderdomain.Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return orderdomain.Order{}, fmt.Errorf("commit order tx: %w", err)
	}
	committed = true
	return order, nil
}

func (r *OrderRepository) updateOrderWithQueries(ctx context.Context, q sqlc.Querier, input orderuc.UpdateOrderRepositoryInput) (orderdomain.Order, error) {
	row, err := q.UpdateOrder(ctx, sqlc.UpdateOrderParams{
		FromDepartmentID: input.FromDepartmentID,
		ToDepartmentID:   input.ToDepartmentID,
		FulfillmentDate:  helpers.DateOf(input.FulfillmentDate),
		Number:           input.Number,
		Comments:         marshalComments(input.Comments),
	})
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("update order: %w", err)
	}
	if err := q.DeleteOrderItemsByOrderID(ctx, row.ID); err != nil {
		return orderdomain.Order{}, fmt.Errorf("delete order items: %w", err)
	}
	if err := r.createOrderItems(ctx, q, row.ID, input.Items); err != nil {
		return orderdomain.Order{}, err
	}
	if len(input.HistoryItems) > 0 {
		if err := r.createOrderHistory(ctx, q, row.ID, input.ChangedByUsername, input.HistoryItems); err != nil {
			return orderdomain.Order{}, err
		}
	}
	history, err := r.listOrderHistory(ctx, q, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	order := orderFromRow(row, input.Items, history)
	if err := r.attachCategory(ctx, &order); err != nil {
		return orderdomain.Order{}, err
	}
	order.RecordUpdated()
	if err := r.persistOutbox(ctx, q, order.Number, order.PullDomainEvents()); err != nil {
		return orderdomain.Order{}, err
	}
	return order, nil
}

func (r *OrderRepository) GetOrderByNumber(ctx context.Context, number string) (orderdomain.Order, error) {
	order, err := r.queries.GetOrderByNumber(ctx, number)
	if err != nil {
		return orderdomain.Order{}, err
	}
	items, err := loadOrderItems(ctx, r.queries, order.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	history, err := r.listOrderHistory(ctx, r.queries, order.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	result := orderFromRow(order, items, history)
	if err := r.attachCategory(ctx, &result); err != nil {
		return orderdomain.Order{}, err
	}
	if err := attachProductionSheet(ctx, r.queries, order.ID, &result); err != nil {
		return orderdomain.Order{}, err
	}
	return result, nil
}

func (r *OrderRepository) ListOrders(ctx context.Context, input orderuc.ListOrdersInput) (orderuc.ListOrdersResult, error) {
	var fulfillmentDate, fulfillmentFrom, fulfillmentTo pgtype.Date
	if !input.FulfillmentDate.IsZero() {
		fulfillmentDate = helpers.DateOf(input.FulfillmentDate)
	}
	if !input.FulfillmentFrom.IsZero() {
		fulfillmentFrom = helpers.DateOf(input.FulfillmentFrom)
	}
	if !input.FulfillmentTo.IsZero() {
		fulfillmentTo = helpers.DateOf(input.FulfillmentTo)
	}
	total, err := r.queries.CountOrders(ctx, sqlc.CountOrdersParams{
		FromDepartmentID: input.FromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
		FulfillmentFrom:  fulfillmentFrom,
		FulfillmentTo:    fulfillmentTo,
		CategoryID:       input.CategoryID,
	})
	if err != nil {
		return orderuc.ListOrdersResult{}, err
	}
	rows, err := r.queries.ListOrders(ctx, sqlc.ListOrdersParams{
		FromDepartmentID: input.FromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
		FulfillmentFrom:  fulfillmentFrom,
		FulfillmentTo:    fulfillmentTo,
		CategoryID:       input.CategoryID,
		OrderLimit:       input.Limit,
		OrderOffset:      input.Offset,
	})
	if err != nil {
		return orderuc.ListOrdersResult{}, err
	}
	orderIDs := make([]int64, len(rows))
	for i, row := range rows {
		orderIDs[i] = row.ID
	}
	itemsByOrder, err := r.orderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return orderuc.ListOrdersResult{}, err
	}
	result := make([]orderdomain.Order, 0, len(rows))
	for _, row := range rows {
		result = append(result, orderFromRow(row, itemsByOrder[row.ID], nil))
	}
	if err := r.attachCategories(ctx, result); err != nil {
		return orderuc.ListOrdersResult{}, err
	}
	if err := r.attachProductionSheets(ctx, orderIDs, result); err != nil {
		return orderuc.ListOrdersResult{}, err
	}
	return orderuc.ListOrdersResult{
		Orders: result,
		Total:  total,
		Limit:  input.Limit,
		Offset: input.Offset,
	}, nil
}

// attachProductionSheets resolves production-sheet membership for a list in
// one query. A sheet with zero deviations still marks the order as worked.
func (r *OrderRepository) attachProductionSheets(ctx context.Context, ids []int64, orders []orderdomain.Order) error {
	if len(orders) == 0 {
		return nil
	}
	byID := make(map[int64]*orderdomain.Order, len(orders))
	for i := range orders {
		byID[ids[i]] = &orders[i]
	}
	rows, err := r.queries.ListOrderProductionSheetIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("list order production sheets: %w", err)
	}
	for _, row := range rows {
		if order := byID[row.OrderID]; order != nil {
			sheetID := row.SheetID
			order.ProductionSheetID = &sheetID
		}
	}
	return nil
}

func (r *OrderRepository) GetDepartmentByID(ctx context.Context, id int64) (orderuc.Department, error) {
	department, err := r.queries.GetDepartmentByID(ctx, id)
	if err != nil {
		return orderuc.Department{}, err
	}
	return orderuc.Department{
		ID:   department.ID,
		Code: department.Code,
		Name: department.Name,
		Type: department.Type,
	}, nil
}

func (r *OrderRepository) DishExistsByCode(ctx context.Context, code string) (bool, error) {
	count, err := r.queries.DishExistsByCode(ctx, code)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *OrderRepository) ResolveDishCatalogItem(ctx context.Context, name string) (orderuc.DishCatalogItem, error) {
	rows, err := r.queries.ListDishCatalogItemsByName(ctx, strings.TrimSpace(name))
	if err != nil {
		return orderuc.DishCatalogItem{}, fmt.Errorf("list dish catalog items by name: %w", err)
	}
	if len(rows) == 0 {
		return orderuc.DishCatalogItem{}, orderuc.ErrDishCatalogItemNotFound
	}

	byCode := make(map[string]sqlc.DishCatalog, len(rows))
	for _, row := range rows {
		byCode[strings.TrimSpace(row.Code)] = row
	}
	if len(byCode) > 1 {
		return orderuc.DishCatalogItem{}, orderuc.ErrDishCatalogItemAmbiguous
	}
	for _, row := range byCode {
		// Theme is unused by the resolve caller (it only reads Code/Name), so
		// skip the extra group lookup here.
		return dishCatalogItem(row, ""), nil
	}
	return orderuc.DishCatalogItem{}, orderuc.ErrDishCatalogItemNotFound
}

func (r *OrderRepository) ListDishCatalog(ctx context.Context) ([]orderuc.DishCatalogItem, error) {
	rows, err := r.queries.ListDishCatalogItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dish catalog items: %w", err)
	}
	names, err := r.groupNames(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]orderuc.DishCatalogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dishCatalogItem(row, themeFor(names, row.GroupID)))
	}
	return items, nil
}

func (r *OrderRepository) UpsertDishCatalogItem(ctx context.Context, item orderuc.DishCatalogItem) error {
	now := helpers.TimestamptzNow()
	groupID, err := r.resolveDishGroupID(ctx, item.Theme, item.CategoryID)
	if err != nil {
		return err
	}
	_, err = r.queries.UpsertDishCatalogItem(ctx, sqlc.UpsertDishCatalogItemParams{
		Code:       item.Code,
		Name:       item.Name,
		GroupID:    groupID,
		CategoryID: item.CategoryID,
		SortOrder:  item.SortOrder,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	return err
}

func (r *OrderRepository) SearchAvailableDishes(ctx context.Context, query string, limit int) ([]orderdomain.AvailableDish, error) {
	var q *string
	if query != "" {
		q = &query
	}
	if limit < 0 || limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	rows, err := r.queries.SearchIikoDishes(ctx, sqlc.SearchIikoDishesParams{Query: q, Lim: int32(limit)})
	if err != nil {
		return nil, fmt.Errorf("search iiko dishes: %w", err)
	}
	dishes := make([]orderdomain.AvailableDish, 0, len(rows))
	for _, row := range rows {
		dishes = append(dishes, orderdomain.AvailableDish{Code: row.Code, Name: row.Name, Unit: row.MeasureUnit})
	}
	return dishes, nil
}

func (r *OrderRepository) SetDishCatalogSortOrder(ctx context.Context, code string, sortOrder int64) error {
	return r.queries.SetDishCatalogSortOrder(ctx, sqlc.SetDishCatalogSortOrderParams{
		Code:      code,
		SortOrder: sortOrder,
		UpdatedAt: helpers.TimestamptzNow(),
	})
}

func (r *OrderRepository) UpdateDishCatalogItem(ctx context.Context, code string, item orderuc.DishCatalogItem) (orderuc.DishCatalogItem, error) {
	groupID, err := r.resolveDishGroupID(ctx, item.Theme, item.CategoryID)
	if err != nil {
		return orderuc.DishCatalogItem{}, err
	}
	row, err := r.queries.UpdateDishCatalogItem(ctx, sqlc.UpdateDishCatalogItemParams{
		Code:       code,
		NewCode:    item.Code,
		Name:       item.Name,
		GroupID:    groupID,
		CategoryID: item.CategoryID,
		SortOrder:  item.SortOrder,
		UpdatedAt:  helpers.TimestamptzNow(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return orderuc.DishCatalogItem{}, orderuc.ErrDishCatalogItemNotFound
		}
		return orderuc.DishCatalogItem{}, fmt.Errorf("update dish catalog item: %w", err)
	}
	return dishCatalogItem(row, strings.TrimSpace(item.Theme)), nil
}

func (r *OrderRepository) DeleteDishCatalogItem(ctx context.Context, code string) error {
	return r.queries.DeleteDishCatalogItem(ctx, code)
}

func (r *OrderRepository) DeleteOrdersOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return r.queries.DeleteOrdersCreatedBefore(ctx, helpers.Timestamptz(cutoff))
}

func (r *OrderRepository) createOrderItems(ctx context.Context, q sqlc.Querier, orderID int64, items []orderdomain.OrderItem) error {
	for _, item := range items {
		if item.ProductionQuantity() <= 0 {
			continue
		}
		var productID *string
		if item.Code != "" {
			product, err := q.GetIikoProductByCode(ctx, item.Code)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("resolve product by code %s: %w", item.Code, err)
			}
			if err == nil {
				productID = &product.ID
			}
		}
		if _, err := q.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
			OrderID:          orderID,
			IikoProductID:    productID,
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		}); err != nil {
			return fmt.Errorf("create order item: %w", err)
		}
	}
	return nil
}

func (r *OrderRepository) createOrderHistory(ctx context.Context, q sqlc.Querier, orderID int64, changedBy string, items []orderdomain.OrderHistoryItem) error {
	history, err := q.CreateOrderHistory(ctx, sqlc.CreateOrderHistoryParams{
		OrderID:           orderID,
		ChangedByUsername: strings.TrimSpace(changedBy),
		ChangedAt:         helpers.TimestamptzNow(),
	})
	if err != nil {
		return fmt.Errorf("create order history: %w", err)
	}
	for _, item := range items {
		if _, err := q.CreateOrderHistoryItem(ctx, sqlc.CreateOrderHistoryItemParams{
			HistoryID:           history.ID,
			ChangeType:          item.ChangeType,
			ProductCode:         item.ProductCode,
			ProductName:         item.ProductName,
			OldQuantity:         item.OldQuantity,
			NewQuantity:         item.NewQuantity,
			OldReservedQuantity: item.OldReservedQuantity,
			NewReservedQuantity: item.NewReservedQuantity,
		}); err != nil {
			return fmt.Errorf("create order history item: %w", err)
		}
	}
	return nil
}

func (r *OrderRepository) listOrderHistory(ctx context.Context, q sqlc.Querier, orderID int64) ([]orderdomain.OrderHistory, error) {
	rows, err := q.ListOrderHistoryByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	result := make([]orderdomain.OrderHistory, 0, len(rows))
	for _, row := range rows {
		itemRows, err := q.ListOrderHistoryItemsByHistoryID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		items := make([]orderdomain.OrderHistoryItem, 0, len(itemRows))
		for _, item := range itemRows {
			items = append(items, orderdomain.OrderHistoryItem{
				ChangeType:          item.ChangeType,
				ProductCode:         item.ProductCode,
				ProductName:         item.ProductName,
				OldQuantity:         item.OldQuantity,
				NewQuantity:         item.NewQuantity,
				OldReservedQuantity: item.OldReservedQuantity,
				NewReservedQuantity: item.NewReservedQuantity,
			})
		}
		result = append(result, orderdomain.OrderHistory{
			ID:                row.ID,
			ChangedByUsername: row.ChangedByUsername,
			ChangedAt:         row.ChangedAt.Time,
			Items:             items,
		})
	}
	return result, nil
}

func orderFromRow(row sqlc.Order, items []orderdomain.OrderItem, history []orderdomain.OrderHistory) orderdomain.Order {
	return orderdomain.Order{
		ID:                  fmt.Sprintf("%d", row.ID),
		Number:              row.Number,
		Location:            row.Location,
		FromDepartmentID:    row.FromDepartmentID,
		ToDepartmentID:      row.ToDepartmentID,
		CategoryID:          row.CategoryID,
		CreatedByUsername:   row.CreatedByUsername,
		Items:               items,
		CreatedAt:           row.CreatedAt.Time,
		FulfillmentDate:     row.FulfillmentDate.Time,
		Comments:            parseComments(row.Comments),
		Favorite:            row.IsFavorite,
		Cancelled:           row.CancelledAt.Valid,
		CancelledByUsername: row.CancelledByUsername,
		History:             history,
	}
}

func (r *OrderRepository) SetOrderFavorite(ctx context.Context, number string, favorite bool) (orderdomain.Order, error) {
	row, err := r.queries.SetOrderFavorite(ctx, sqlc.SetOrderFavoriteParams{Number: number, IsFavorite: favorite})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return orderdomain.Order{}, apperr.NotFound("order.not_found", "Заказ не найден.")
		}
		return orderdomain.Order{}, fmt.Errorf("set order favorite: %w", err)
	}
	items, err := loadOrderItems(ctx, r.queries, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	history, err := r.listOrderHistory(ctx, r.queries, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	order := orderFromRow(row, items, history)
	if err := r.attachCategory(ctx, &order); err != nil {
		return orderdomain.Order{}, err
	}
	return order, nil
}

// CancelOrder soft-cancels the order, recording who cancelled it, and emits an
// order-cancelled event to the outbox in the same transaction.
func (r *OrderRepository) CancelOrder(ctx context.Context, number, by string) (orderdomain.Order, error) {
	return r.withOrderTx(ctx, func(q sqlc.Querier) (orderdomain.Order, error) {
		row, err := q.CancelOrder(ctx, sqlc.CancelOrderParams{
			Number:              number,
			CancelledAt:         helpers.TimestamptzNow(),
			CancelledByUsername: strings.TrimSpace(by),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return orderdomain.Order{}, apperr.NotFound("order.not_found", "Заказ не найден.")
			}
			return orderdomain.Order{}, fmt.Errorf("cancel order: %w", err)
		}
		order, err := r.hydrateOrder(ctx, q, row)
		if err != nil {
			return orderdomain.Order{}, err
		}
		order.RecordCancelled()
		if err := r.persistOutbox(ctx, q, order.Number, order.PullDomainEvents()); err != nil {
			return orderdomain.Order{}, err
		}
		return order, nil
	})
}

// RestoreOrder clears an order's cancelled state and emits an order-restored
// event to the outbox in the same transaction.
func (r *OrderRepository) RestoreOrder(ctx context.Context, number string) (orderdomain.Order, error) {
	return r.withOrderTx(ctx, func(q sqlc.Querier) (orderdomain.Order, error) {
		row, err := q.RestoreOrder(ctx, number)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return orderdomain.Order{}, apperr.NotFound("order.not_found", "Заказ не найден.")
			}
			return orderdomain.Order{}, fmt.Errorf("restore order: %w", err)
		}
		order, err := r.hydrateOrder(ctx, q, row)
		if err != nil {
			return orderdomain.Order{}, err
		}
		order.RecordRestored()
		if err := r.persistOutbox(ctx, q, order.Number, order.PullDomainEvents()); err != nil {
			return orderdomain.Order{}, err
		}
		return order, nil
	})
}

// withOrderTx runs fn inside a transaction (or directly when no pool is set, as
// in tests), committing on success.
func (r *OrderRepository) withOrderTx(ctx context.Context, fn func(q sqlc.Querier) (orderdomain.Order, error)) (orderdomain.Order, error) {
	if r.db == nil {
		return fn(r.queries)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("begin order tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	order, err := fn(r.queries.WithTx(tx))
	if err != nil {
		return orderdomain.Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return orderdomain.Order{}, fmt.Errorf("commit order tx: %w", err)
	}
	committed = true
	return order, nil
}

// hydrateOrder loads an order's items and history for the given row.
func (r *OrderRepository) hydrateOrder(ctx context.Context, q sqlc.Querier, row sqlc.Order) (orderdomain.Order, error) {
	items, err := loadOrderItems(ctx, q, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	history, err := r.listOrderHistory(ctx, q, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	order := orderFromRow(row, items, history)
	if err := r.attachCategory(ctx, &order); err != nil {
		return orderdomain.Order{}, err
	}
	if err := attachProductionSheet(ctx, q, row.ID, &order); err != nil {
		return orderdomain.Order{}, err
	}
	return order, nil
}

// attachProductionSheet resolves the отработка document covering the order,
// so delivery can link заказ → отработка.
func attachProductionSheet(ctx context.Context, q sqlc.Querier, orderID int64, order *orderdomain.Order) error {
	sheetID, err := q.GetOrderProductionSheetID(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get order production sheet: %w", err)
	}
	order.ProductionSheetID = &sheetID
	return nil
}

func categoryFromRow(row sqlc.OrderCategory) orderdomain.OrderCategory {
	return orderdomain.OrderCategory{
		ID:           row.ID,
		Code:         row.Code,
		Letter:       row.Letter,
		Name:         row.Name,
		Color:        row.Color,
		SortOrder:    row.SortOrder,
		MonitorCodes: row.MonitorCodes,
	}
}

// attachCategories resolves Category for the given orders in one lookup, so
// list responses and event snapshots carry the category name/color/letter.
func (r *OrderRepository) attachCategories(ctx context.Context, orders []orderdomain.Order) error {
	rows, err := r.queries.ListOrderCategories(ctx)
	if err != nil {
		return fmt.Errorf("list order categories: %w", err)
	}
	byID := make(map[int64]orderdomain.OrderCategory, len(rows))
	for _, row := range rows {
		byID[row.ID] = categoryFromRow(row)
	}
	for i := range orders {
		if orders[i].CategoryID == nil {
			continue
		}
		if category, ok := byID[*orders[i].CategoryID]; ok {
			orders[i].Category = &category
		}
	}
	return nil
}

func (r *OrderRepository) attachCategory(ctx context.Context, order *orderdomain.Order) error {
	if order.CategoryID == nil {
		return nil
	}
	row, err := r.queries.GetOrderCategoryByID(ctx, *order.CategoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Категория могла быть удалена — заказ остаётся валидным без неё.
		return nil
	}
	if err != nil {
		return fmt.Errorf("get order category %d: %w", *order.CategoryID, err)
	}
	category := categoryFromRow(row)
	order.Category = &category
	return nil
}

func (r *OrderRepository) ListOrderCategories(ctx context.Context) ([]orderdomain.OrderCategory, error) {
	rows, err := r.queries.ListOrderCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list order categories: %w", err)
	}
	categories := make([]orderdomain.OrderCategory, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, categoryFromRow(row))
	}
	return categories, nil
}

func (r *OrderRepository) GetOrderCategoryByID(ctx context.Context, id int64) (orderdomain.OrderCategory, error) {
	row, err := r.queries.GetOrderCategoryByID(ctx, id)
	if err != nil {
		return orderdomain.OrderCategory{}, fmt.Errorf("get order category %d: %w", id, err)
	}
	return categoryFromRow(row), nil
}

func (r *OrderRepository) CreateOrderCategory(ctx context.Context, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error) {
	row, err := r.queries.CreateOrderCategory(ctx, sqlc.CreateOrderCategoryParams{
		Code:         input.Code,
		Letter:       input.Letter,
		Name:         input.Name,
		Color:        input.Color,
		SortOrder:    input.SortOrder,
		MonitorCodes: input.MonitorCodes,
	})
	if err != nil {
		return orderdomain.OrderCategory{}, fmt.Errorf("create order category: %w", err)
	}
	return categoryFromRow(row), nil
}

func (r *OrderRepository) UpdateOrderCategory(ctx context.Context, id int64, input orderdomain.OrderCategory) (orderdomain.OrderCategory, error) {
	row, err := r.queries.UpdateOrderCategory(ctx, sqlc.UpdateOrderCategoryParams{
		ID:           id,
		Letter:       input.Letter,
		Name:         input.Name,
		Color:        input.Color,
		SortOrder:    input.SortOrder,
		MonitorCodes: input.MonitorCodes,
	})
	if err != nil {
		return orderdomain.OrderCategory{}, fmt.Errorf("update order category %d: %w", id, err)
	}
	return categoryFromRow(row), nil
}

func (r *OrderRepository) DeleteOrderCategory(ctx context.Context, id int64) error {
	return r.queries.DeleteOrderCategory(ctx, id)
}

func (r *OrderRepository) CountDishesByCategoryID(ctx context.Context, id int64) (int64, error) {
	return r.queries.CountDishesByCategoryID(ctx, &id)
}

// marshalComments serializes order comments to JSON, returning nil (SQL NULL)
// when there is nothing to store.
func marshalComments(comments orderdomain.OrderComments) []byte {
	if strings.TrimSpace(comments.General) == "" && len(comments.Items) == 0 {
		return nil
	}
	data, err := json.Marshal(comments)
	if err != nil {
		return nil
	}
	return data
}

func parseComments(raw []byte) orderdomain.OrderComments {
	if len(raw) == 0 {
		return orderdomain.OrderComments{}
	}
	var comments orderdomain.OrderComments
	if err := json.Unmarshal(raw, &comments); err != nil {
		return orderdomain.OrderComments{}
	}
	return comments
}

// orderItemsByOrderIDs loads items for many orders in a single query and groups
// them by order id, avoiding the N+1 of one query per order. Items come back
// decorated with the production fact from the journal.
func (r *OrderRepository) orderItemsByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64][]orderdomain.OrderItem, error) {
	grouped := make(map[int64][]orderdomain.OrderItem, len(orderIDs))
	if len(orderIDs) == 0 {
		return grouped, nil
	}
	rows, err := r.queries.GetOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		grouped[item.OrderID] = append(grouped[item.OrderID], orderdomain.OrderItem{
			Code:             item.ProductCode,
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		})
	}
	if err := decorateProductionFacts(ctx, r.queries, grouped); err != nil {
		return nil, err
	}
	return grouped, nil
}

// loadOrderItems returns one order's items decorated with the production fact.
func loadOrderItems(ctx context.Context, q sqlc.Querier, orderID int64) ([]orderdomain.OrderItem, error) {
	rows, err := q.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	items := mapOrderItems(rows)
	if err := decorateProductionFacts(ctx, q, map[int64][]orderdomain.OrderItem{orderID: items}); err != nil {
		return nil, err
	}
	return items, nil
}

// decorateProductionFacts — read-time «декоратор» отработки: заказ не хранит
// факт выпечки, он подтягивается из журнала (production_sheet_items, свежий
// лист побеждает) и накладывается на позиции при чтении.
func decorateProductionFacts(ctx context.Context, q sqlc.Querier, grouped map[int64][]orderdomain.OrderItem) error {
	if len(grouped) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(grouped))
	for id := range grouped {
		ids = append(ids, id)
	}
	facts, err := q.GetOrderProductionFacts(ctx, ids)
	if err != nil {
		return fmt.Errorf("load production facts: %w", err)
	}
	if len(facts) == 0 {
		return nil
	}
	byOrder := make(map[int64]map[string]sqlc.GetOrderProductionFactsRow, len(facts))
	for _, fact := range facts {
		byKey := byOrder[fact.OrderID]
		if byKey == nil {
			byKey = make(map[string]sqlc.GetOrderProductionFactsRow)
			byOrder[fact.OrderID] = byKey
		}
		byKey[fact.ProductKey] = fact
	}
	for orderID, items := range grouped {
		byKey := byOrder[orderID]
		if byKey == nil {
			continue
		}
		for i := range items {
			key := strings.ToLower(strings.TrimSpace(items[i].ProductName))
			if fact, ok := byKey[key]; ok {
				quantity := fact.ProducedQuantity
				items[i].ProducedQuantity = &quantity
				items[i].ProducedReason = fact.Reason
			}
		}
	}
	return nil
}

func mapOrderItems(items []sqlc.GetOrderItemsByOrderIDRow) []orderdomain.OrderItem {
	result := make([]orderdomain.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, orderdomain.OrderItem{
			Code:             item.ProductCode,
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		})
	}
	return result
}

func dishCatalogItem(row sqlc.DishCatalog, theme string) orderuc.DishCatalogItem {
	return orderuc.DishCatalogItem{
		Code:       row.Code,
		Name:       row.Name,
		Theme:      theme,
		CategoryID: row.CategoryID,
		SortOrder:  row.SortOrder,
	}
}

// groupNames returns id → group name for mapping a dish's group_id back to the
// «группа» string the domain/UI still work with.
func (r *OrderRepository) groupNames(ctx context.Context) (map[int64]string, error) {
	groups, err := r.queries.ListDishGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dish groups: %w", err)
	}
	names := make(map[int64]string, len(groups))
	for _, g := range groups {
		names[g.ID] = g.Name
	}
	return names, nil
}

func themeFor(names map[int64]string, groupID *int64) string {
	if groupID == nil {
		return ""
	}
	return names[*groupID]
}

// resolveDishGroupID find-or-creates the group for a dish's «группа» name under
// its category, returning the group_id (nil for an empty group name).
func (r *OrderRepository) resolveDishGroupID(ctx context.Context, theme string, categoryID *int64) (*int64, error) {
	theme = strings.TrimSpace(theme)
	if theme == "" {
		return nil, nil
	}
	id, err := r.queries.EnsureDishGroup(ctx, sqlc.EnsureDishGroupParams{
		CategoryID: categoryID,
		Name:       theme,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure dish group: %w", err)
	}
	return &id, nil
}
