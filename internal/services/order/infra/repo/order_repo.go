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
	if err := q.CreateOrderCounterDay(ctx, sqlc.CreateOrderCounterDayParams{Day: input.CounterDay, DepartmentID: input.Shop.ID}); err != nil {
		return orderdomain.Order{}, fmt.Errorf("init order counter: %w", err)
	}
	counter, err := q.NextOrderCounter(ctx, sqlc.NextOrderCounterParams{Day: input.CounterDay, DepartmentID: input.Shop.ID})
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("increment order counter: %w", err)
	}

	number := r.domain.BuildOrderNumber(input.Shop.Code, input.Shop.Name, input.CreatedAt, counter)
	row, err := q.CreateOrder(ctx, sqlc.CreateOrderParams{
		Number:            number,
		Location:          input.Input.Location,
		FromDepartmentID:  input.Input.FromDepartmentID,
		ToDepartmentID:    input.Input.ToDepartmentID,
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
		FromDepartmentID:  input.FromDepartmentID,
		ToDepartmentID:    input.ToDepartmentID,
		FulfillmentDate:   helpers.DateOf(input.FulfillmentDate),
		CreatedByUsername: input.CreatedByUsername,
		Number:            input.Number,
		Comments:          marshalComments(input.Comments),
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
		if err := r.createOrderHistory(ctx, q, row.ID, input.CreatedByUsername, input.HistoryItems); err != nil {
			return orderdomain.Order{}, err
		}
	}
	history, err := r.listOrderHistory(ctx, q, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	order := orderFromRow(row, input.Items, history)
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
	items, err := r.queries.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	history, err := r.listOrderHistory(ctx, r.queries, order.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	return orderFromRow(order, mapOrderItems(items), history), nil
}

func (r *OrderRepository) ListOrders(ctx context.Context, input orderuc.ListOrdersInput) (orderuc.ListOrdersResult, error) {
	var fulfillmentDate pgtype.Date
	if !input.FulfillmentDate.IsZero() {
		fulfillmentDate = helpers.DateOf(input.FulfillmentDate)
	}
	total, err := r.queries.CountOrders(ctx, sqlc.CountOrdersParams{
		FromDepartmentID: input.FromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
	})
	if err != nil {
		return orderuc.ListOrdersResult{}, err
	}
	rows, err := r.queries.ListOrders(ctx, sqlc.ListOrdersParams{
		FromDepartmentID: input.FromDepartmentID,
		FulfillmentDate:  fulfillmentDate,
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
	return orderuc.ListOrdersResult{
		Orders: result,
		Total:  total,
		Limit:  input.Limit,
		Offset: input.Offset,
	}, nil
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
		return dishCatalogItem(row), nil
	}
	return orderuc.DishCatalogItem{}, orderuc.ErrDishCatalogItemNotFound
}

func (r *OrderRepository) ListDishCatalog(ctx context.Context) ([]orderuc.DishCatalogItem, error) {
	rows, err := r.queries.ListDishCatalogItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("list dish catalog items: %w", err)
	}
	items := make([]orderuc.DishCatalogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dishCatalogItem(row))
	}
	return items, nil
}

func (r *OrderRepository) UpsertDishCatalogItem(ctx context.Context, item orderuc.DishCatalogItem) error {
	now := helpers.TimestamptzNow()
	_, err := r.queries.UpsertDishCatalogItem(ctx, sqlc.UpsertDishCatalogItemParams{
		Code:      item.Code,
		Name:      item.Name,
		Theme:     item.Theme,
		SortOrder: item.SortOrder,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}

func (r *OrderRepository) SearchAvailableDishes(ctx context.Context, query string, limit int) ([]orderdomain.AvailableDish, error) {
	var q *string
	if query != "" {
		q = &query
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
	row, err := r.queries.UpdateDishCatalogItem(ctx, sqlc.UpdateDishCatalogItemParams{
		Code:      code,
		NewCode:   item.Code,
		Name:      item.Name,
		Theme:     item.Theme,
		SortOrder: item.SortOrder,
		UpdatedAt: helpers.TimestamptzNow(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return orderuc.DishCatalogItem{}, orderuc.ErrDishCatalogItemNotFound
		}
		return orderuc.DishCatalogItem{}, fmt.Errorf("update dish catalog item: %w", err)
	}
	return dishCatalogItem(row), nil
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
		ID:                fmt.Sprintf("%d", row.ID),
		Number:            row.Number,
		Location:          row.Location,
		FromDepartmentID:  row.FromDepartmentID,
		ToDepartmentID:    row.ToDepartmentID,
		CreatedByUsername: row.CreatedByUsername,
		Items:             items,
		CreatedAt:         row.CreatedAt.Time,
		FulfillmentDate:   row.FulfillmentDate.Time,
		Comments:          parseComments(row.Comments),
		Favorite:          row.IsFavorite,
		History:           history,
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
	items, err := r.queries.GetOrderItemsByOrderID(ctx, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	history, err := r.listOrderHistory(ctx, r.queries, row.ID)
	if err != nil {
		return orderdomain.Order{}, err
	}
	return orderFromRow(row, mapOrderItems(items), history), nil
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
// them by order id, avoiding the N+1 of one query per order.
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
	return grouped, nil
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

func dishCatalogItem(row sqlc.DishCatalog) orderuc.DishCatalogItem {
	return orderuc.DishCatalogItem{
		Code:      row.Code,
		Name:      row.Name,
		Theme:     row.Theme,
		SortOrder: row.SortOrder,
	}
}
