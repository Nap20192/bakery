package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	db "bakery/internal/adapter/outbound/db"
	"bakery/internal/domain/client"
	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/order"
	"bakery/internal/domain/product"
)

type OrderRepository struct {
	q *db.Queries
}

func NewOrderRepository(q *db.Queries) *OrderRepository {
	return &OrderRepository{q: q}
}

func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	note := nullString(o.Note())
	orderDate := o.OrderDate()

	_, err := r.q.GetOrderByID(ctx, o.ID())
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		var chatID, msgID *int64
		if tg := o.Telegram(); tg != nil {
			c, m := tg.ChatID, tg.MessageID
			chatID, msgID = &c, &m
		}
		raw := nullString(o.RawText())
		status := string(o.Status())
		if _, err := r.q.CreateOrder(ctx, db.CreateOrderParams{
			ID:                o.ID(),
			ClientID:          o.ClientID(),
			KitchenID:         o.KitchenID(),
			TelegramChatID:    chatID,
			TelegramMessageID: msgID,
			OrderDate:         orderDate,
			RawText:           raw,
			Note:              note,
			Status:            &status,
		}); err != nil {
			return fmt.Errorf("order repo: create: %w", err)
		}
	case err != nil:
		return fmt.Errorf("order repo: probe: %w", err)
	default:
		if _, err := r.q.UpdateOrder(ctx, db.UpdateOrderParams{
			ID:        o.ID(),
			KitchenID: o.KitchenID(),
			OrderDate: orderDate,
			Status:    string(o.Status()),
			Note:      note,
		}); err != nil {
			return fmt.Errorf("order repo: update: %w", err)
		}
	}

	return r.syncItems(ctx, o)
}

func (r *OrderRepository) syncItems(ctx context.Context, o *order.Order) error {
	if err := r.q.DeleteOrderItems(ctx, o.ID()); err != nil {
		return fmt.Errorf("order repo: delete items: %w", err)
	}
	for _, it := range o.Items() {
		if _, err := r.q.CreateOrderItem(ctx, db.CreateOrderItemParams{
			ID:        it.ID(),
			OrderID:   o.ID(),
			ProductID: it.ProductID(),
			Quantity:  it.Quantity(),
			Unit:      string(it.Unit()),
		}); err != nil {
			return fmt.Errorf("order repo: create item: %w", err)
		}
	}
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id order.ID) (*order.Order, error) {
	row, err := r.q.GetOrderByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, order.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order repo: get by id: %w", err)
	}
	items, err := r.loadItems(ctx, id)
	if err != nil {
		return nil, err
	}
	return toOrder(row, items), nil
}

func (r *OrderRepository) GetByTelegram(ctx context.Context, chatID, messageID int64) (*order.Order, error) {
	return nil, order.ErrNotFound
}

func (r *OrderRepository) List(ctx context.Context, f order.Filter) ([]*order.Order, error) {
	var rows []db.Order
	var err error
	switch {
	case f.DateFrom != nil && f.DateTo != nil:
		rows, err = r.q.ListOrdersByDateRange(ctx, db.ListOrdersByDateRangeParams{
			OrderDate:   *f.DateFrom,
			OrderDate_2: *f.DateTo,
			Status:      statusPtr(f.Status),
		})
	case f.ClientID != nil:
		rows, err = r.q.ListOrdersByClient(ctx, db.ListOrdersByClientParams{
			ClientID: *f.ClientID,
			Status:   statusPtr(f.Status),
		})
	case f.KitchenID != nil:
		rows, err = r.q.ListOrdersByKitchen(ctx, db.ListOrdersByKitchenParams{
			KitchenID: f.KitchenID,
			Status:    statusPtr(f.Status),
		})
	default:
		rows, err = r.q.ListOrders(ctx, statusPtr(f.Status))
	}
	if err != nil {
		return nil, fmt.Errorf("order repo: list: %w", err)
	}
	out := make([]*order.Order, 0, len(rows))
	for _, row := range rows {
		items, err := r.loadItems(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, toOrder(row, items))
	}
	return out, nil
}

func (r *OrderRepository) GetOrderIngredients(ctx context.Context, id order.ID) ([]order.IngredientLine, error) {
	rows, err := r.q.GetOrderIngredients(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("order repo: get ingredients: %w", err)
	}
	out := make([]order.IngredientLine, 0, len(rows))
	for _, row := range rows {
		out = append(out, order.IngredientLine{
			IngredientID:   row.ID,
			IngredientName: row.IngredientName,
			Unit:           row.Unit,
			TotalQty:       row.TotalQty,
		})
	}
	return out, nil
}

func (r *OrderRepository) Delete(ctx context.Context, id order.ID) error {
	if err := r.q.DeleteOrderItems(ctx, id); err != nil {
		return fmt.Errorf("order repo: delete items: %w", err)
	}
	if err := r.q.DeleteOrder(ctx, id); err != nil {
		return fmt.Errorf("order repo: delete: %w", err)
	}
	return nil
}

func (r *OrderRepository) loadItems(ctx context.Context, id order.ID) ([]*order.Item, error) {
	rows, err := r.q.ListOrderItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("order repo: list items: %w", err)
	}
	items := make([]*order.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, order.RestoreItem(
			row.ID, row.OrderID, row.ProductID, row.Quantity, product.Unit(row.Unit), row.CreatedAt, row.UpdatedAt,
		))
	}
	return items, nil
}

func toOrder(row db.Order, items []*order.Item) *order.Order {
	var tg *order.TelegramSource
	if row.TelegramChatID != nil && row.TelegramMessageID != nil {
		tg = &order.TelegramSource{ChatID: *row.TelegramChatID, MessageID: *row.TelegramMessageID}
	}
	date := row.OrderDate
	var raw, note string
	if row.RawText != nil {
		raw = *row.RawText
	}
	if row.Note != nil {
		note = *row.Note
	}
	var kitchenID *kitchen.ID
	if row.KitchenID != nil {
		k := *row.KitchenID
		kitchenID = &k
	}
	return order.Restore(
		row.ID,
		client.ID(row.ClientID),
		kitchenID,
		tg,
		date,
		order.Status(row.Status),
		raw,
		note,
		items,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func statusPtr(s *order.Status) *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}
