package orderrepo

import (
	"context"
	"errors"
	"fmt"

	sqlc "bakery/internal/outbound/db/sqlc"
	"bakery/internal/pkg/helpers"
	orderdomain "bakery/internal/services/order/domain"
	"bakery/internal/services/order/eventsourced"
	orderuc "bakery/internal/services/order/usecase/order"

	"github.com/hallgren/eventsourcing"
	"github.com/jackc/pgx/v5"
)

// OrderRepository реализует eventsourced.ReadModelWriter: проецирует события
// заказа в существующие таблицы orders/order_items (read model CQRS).
// До этапа 4 проекция также кладёт legacy-события в outbox — уведомления
// бота продолжают работать без изменений.
//
// Идемпотентность: Created отсекается конфликтом по номеру, Cancelled/
// Restored — проверкой текущего состояния, ItemsUpdated и Production* —
// семантикой полной замены (outbox-дубли возможны только при ручном
// повторном прогоне проекции — приемлемо для сценария восстановления).
var _ eventsourced.ReadModelWriter = (*OrderRepository)(nil)

func (r *OrderRepository) Apply(ctx context.Context, event eventsourcing.Event) error {
	number := event.AggregateID()
	switch data := event.Data().(type) {
	case *eventsourced.Created:
		return r.applyCreated(ctx, number, event, *data)
	case *eventsourced.ItemsUpdated:
		return r.applyItemsUpdated(ctx, number, *data)
	case *eventsourced.Cancelled:
		return r.applyCancelled(ctx, number, *data)
	case *eventsourced.Restored:
		return r.applyRestored(ctx, number, *data)
	case *eventsourced.ProductionRecorded:
		return r.applyProductionRecorded(ctx, number, *data)
	case *eventsourced.ProductionCleared:
		return r.applyProductionCleared(ctx, number, *data)
	}
	return nil
}

func (r *OrderRepository) applyCreated(ctx context.Context, number string, event eventsourcing.Event, data eventsourced.Created) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		location := ""
		if data.FromDepartmentID != nil {
			if department, err := q.GetDepartmentByID(ctx, *data.FromDepartmentID); err == nil {
				location = department.Name
			}
		}
		items := esItemsToDomain(data.Items)
		row, err := q.InsertOrderProjection(ctx, sqlc.InsertOrderProjectionParams{
			Number:            number,
			Location:          location,
			FromDepartmentID:  data.FromDepartmentID,
			ToDepartmentID:    data.ToDepartmentID,
			CategoryID:        data.CategoryID,
			CreatedAt:         helpers.Timestamptz(event.Timestamp()),
			FulfillmentDate:   helpers.DateOf(data.FulfillmentDate),
			CreatedByUsername: data.CreatedByUsername,
			Comments:          marshalComments(esCommentsToDomain(data.Comments)),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			// Конфликт по номеру — событие уже применено.
			return nil
		}
		if err != nil {
			return fmt.Errorf("project order created %s: %w", number, err)
		}
		if err := r.createOrderItems(ctx, q, row.ID, items); err != nil {
			return err
		}
		return r.emitLegacyEvent(ctx, q, row, func(order *orderdomain.Order) { order.RecordCreated() })
	})
}

func (r *OrderRepository) applyItemsUpdated(ctx context.Context, number string, data eventsourced.ItemsUpdated) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		before, err := q.GetOrderByNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("project order updated %s: %w", number, err)
		}
		oldItems, err := q.GetOrderItemsByOrderID(ctx, before.ID)
		if err != nil {
			return err
		}
		items := esItemsToDomain(data.Items)
		row, err := q.UpdateOrderProjection(ctx, sqlc.UpdateOrderProjectionParams{
			Number:          number,
			FulfillmentDate: helpers.DateOf(data.FulfillmentDate),
			Comments:        marshalComments(esCommentsToDomain(data.Comments)),
		})
		if err != nil {
			return fmt.Errorf("project order updated %s: %w", number, err)
		}
		if err := q.DeleteOrderItemsByOrderID(ctx, row.ID); err != nil {
			return err
		}
		if err := r.createOrderItems(ctx, q, row.ID, items); err != nil {
			return err
		}
		if history := orderuc.DiffOrderItems(mapOrderItems(oldItems), items); len(history) > 0 {
			if err := r.createOrderHistory(ctx, q, row.ID, data.ChangedByUsername, history); err != nil {
				return err
			}
		}
		return r.emitLegacyEvent(ctx, q, row, func(order *orderdomain.Order) { order.RecordUpdated() })
	})
}

func (r *OrderRepository) applyCancelled(ctx context.Context, number string, data eventsourced.Cancelled) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		before, err := q.GetOrderByNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("project order cancelled %s: %w", number, err)
		}
		if before.CancelledAt.Valid {
			return nil // уже применено
		}
		row, err := q.CancelOrder(ctx, sqlc.CancelOrderParams{
			Number:              number,
			CancelledAt:         helpers.TimestamptzNow(),
			CancelledByUsername: data.ByUsername,
		})
		if err != nil {
			return err
		}
		return r.emitLegacyEvent(ctx, q, row, func(order *orderdomain.Order) { order.RecordCancelled() })
	})
}

func (r *OrderRepository) applyRestored(ctx context.Context, number string, data eventsourced.Restored) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		before, err := q.GetOrderByNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("project order restored %s: %w", number, err)
		}
		if !before.CancelledAt.Valid {
			return nil // уже применено
		}
		row, err := q.RestoreOrder(ctx, number)
		if err != nil {
			return err
		}
		_ = data
		return r.emitLegacyEvent(ctx, q, row, func(order *orderdomain.Order) { order.RecordRestored() })
	})
}

func (r *OrderRepository) applyProductionRecorded(ctx context.Context, number string, data eventsourced.ProductionRecorded) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		row, err := q.GetOrderByNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("project production recorded %s: %w", number, err)
		}
		// Полная замена: сброс + отклонения из события.
		if err := q.ClearOrderProductionProjection(ctx, row.ID); err != nil {
			return err
		}
		for _, item := range data.Items {
			quantity := item.Quantity
			var reason *string
			if item.Reason != "" {
				value := item.Reason
				reason = &value
			}
			if err := q.SetOrderItemProducedProjection(ctx, sqlc.SetOrderItemProducedProjectionParams{
				OrderID:          row.ID,
				ProductName:      item.ProductName,
				ProducedQuantity: &quantity,
				ProducedReason:   reason,
			}); err != nil {
				return err
			}
		}
		return r.emitLegacyEvent(ctx, q, row, func(order *orderdomain.Order) { order.RecordProduced(data.ByUsername) })
	})
}

func (r *OrderRepository) applyProductionCleared(ctx context.Context, number string, data eventsourced.ProductionCleared) error {
	return r.withTx(ctx, func(q sqlc.Querier) error {
		row, err := q.GetOrderByNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("project production cleared %s: %w", number, err)
		}
		if err := q.ClearOrderProductionProjection(ctx, row.ID); err != nil {
			return err
		}
		return r.emitLegacyEvent(ctx, q, row, func(order *orderdomain.Order) { order.RecordProductionCleared(data.ByUsername) })
	})
}

// emitLegacyEvent кладёт legacy-событие (sharedkernel) в outbox: бот получает
// уведомления по прежнему каналу, пока этап 4 не переведёт его на поток ES.
func (r *OrderRepository) emitLegacyEvent(ctx context.Context, q sqlc.Querier, row sqlc.Order, record func(*orderdomain.Order)) error {
	order, err := r.hydrateOrder(ctx, q, row)
	if err != nil {
		return err
	}
	record(&order)
	return r.persistOutbox(ctx, q, order.Number, order.PullDomainEvents())
}

// NextOrderNumber — командная сторона ES: инкрементирует счётчик (день +
// магазин + тип) и строит номер заказа доменной логикой. Номер фиксируется
// в событии Created.
func (r *OrderRepository) NextOrderNumber(ctx context.Context, input orderuc.NextOrderNumberInput) (string, error) {
	var number string
	err := r.withTx(ctx, func(q sqlc.Querier) error {
		if err := q.CreateOrderCounterDay(ctx, sqlc.CreateOrderCounterDayParams{
			Day: input.CounterDay, DepartmentID: input.Shop.ID, CategoryID: input.Category.ID,
		}); err != nil {
			return fmt.Errorf("init order counter: %w", err)
		}
		counter, err := q.NextOrderCounter(ctx, sqlc.NextOrderCounterParams{
			Day: input.CounterDay, DepartmentID: input.Shop.ID, CategoryID: input.Category.ID,
		})
		if err != nil {
			return fmt.Errorf("increment order counter: %w", err)
		}
		number = r.domain.BuildOrderNumber(input.Shop.Code, input.Shop.Name, input.Category.Letter, input.CreatedAt, counter)
		return nil
	})
	return number, err
}

func esItemsToDomain(items []eventsourced.Item) []orderdomain.OrderItem {
	result := make([]orderdomain.OrderItem, 0, len(items))
	for _, item := range items {
		result = append(result, orderdomain.OrderItem{
			Code:             item.Code,
			ProductName:      item.ProductName,
			Quantity:         item.Quantity,
			ReservedQuantity: item.ReservedQuantity,
		})
	}
	return result
}

func esCommentsToDomain(comments eventsourced.Comments) orderdomain.OrderComments {
	result := orderdomain.OrderComments{General: comments.General}
	for _, item := range comments.Items {
		result.Items = append(result.Items, orderdomain.ItemComment{ProductName: item.ProductName, Comment: item.Comment})
	}
	return result
}
