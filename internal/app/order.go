package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bakery/internal/domain"
	"bakery/internal/adapter/outbound/postgres"
)

type OrderService struct {
	q *postgres.Queries
}

func NewOrderService(q *postgres.Queries) *OrderService {
	return &OrderService{q: q}
}

type CreateOrderInput struct {
	ClientID          uuid.UUID
	KitchenID         *uuid.UUID
	TelegramChatID    int64
	TelegramMessageID int64
	OrderDate         time.Time
	RawText           string
	Note              string
}

func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (domain.Order, error) {
	if in.ClientID == uuid.Nil {
		return domain.Order{}, fmt.Errorf("client_id is required")
	}
	if in.OrderDate.IsZero() {
		return domain.Order{}, fmt.Errorf("order_date is required")
	}

	row, err := s.q.OrderInsert(ctx, postgres.OrderInsertParams{
		ClientID:          in.ClientID,
		KitchenID:         in.KitchenID,
		TelegramChatID:    int64Ptr(in.TelegramChatID),
		TelegramMessageID: int64Ptr(in.TelegramMessageID),
		OrderDate:         in.OrderDate.Format("2006-01-02"),
		Status:            string(domain.OrderNew),
		RawText:           strPtr(in.RawText),
		Note:              strPtr(in.Note),
	})
	if err != nil {
		return domain.Order{}, fmt.Errorf("create order: %w", err)
	}
	return orderToDomain(row), nil
}

func (s *OrderService) GetByID(ctx context.Context, id uuid.UUID) (domain.Order, error) {
	row, err := s.q.OrderGetByID(ctx, id)
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order %s: %w", id, err)
	}
	o := orderToDomain(row)

	items, err := s.q.OrderItemListByOrder(ctx, id)
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order items %s: %w", id, err)
	}
	o.Items = make([]domain.OrderItem, len(items))
	for i, r := range items {
		o.Items[i] = orderItemToDomain(r)
	}
	return o, nil
}

func (s *OrderService) ListByDate(ctx context.Context, date time.Time) ([]domain.Order, error) {
	rows, err := s.q.OrderListByDate(ctx, date.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("list orders by date: %w", err)
	}
	out := make([]domain.Order, len(rows))
	for i, r := range rows {
		out[i] = orderToDomain(r)
	}
	return out, nil
}

func (s *OrderService) ListByDateAndStatus(ctx context.Context, date time.Time, status domain.OrderStatus) ([]domain.Order, error) {
	rows, err := s.q.OrderListByDateAndStatus(ctx, postgres.OrderListByDateAndStatusParams{
		OrderDate: date.Format("2006-01-02"),
		Status:    string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("list orders by date and status: %w", err)
	}
	out := make([]domain.Order, len(rows))
	for i, r := range rows {
		out[i] = orderToDomain(r)
	}
	return out, nil
}

func (s *OrderService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (domain.Order, error) {
	row, err := s.q.OrderUpdateStatus(ctx, postgres.OrderUpdateStatusParams{
		ID:     id,
		Status: string(status),
	})
	if err != nil {
		return domain.Order{}, fmt.Errorf("update order status %s: %w", id, err)
	}
	return orderToDomain(row), nil
}

func (s *OrderService) Cancel(ctx context.Context, id uuid.UUID) error {
	if err := s.q.OrderSoftDelete(ctx, id); err != nil {
		return fmt.Errorf("cancel order %s: %w", id, err)
	}
	return nil
}

// ─── Order Items ────────────────────────────────────────────────────────────

type AddOrderItemInput struct {
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Quantity  float64
	Unit      string
}

func (s *OrderService) AddItem(ctx context.Context, in AddOrderItemInput) (domain.OrderItem, error) {
	if in.Quantity <= 0 {
		return domain.OrderItem{}, fmt.Errorf("quantity must be positive")
	}
	if in.Unit == "" {
		in.Unit = "шт"
	}

	row, err := s.q.OrderItemInsert(ctx, postgres.OrderItemInsertParams{
		OrderID:   in.OrderID,
		ProductID: in.ProductID,
		Quantity:  in.Quantity,
		Unit:      in.Unit,
	})
	if err != nil {
		return domain.OrderItem{}, fmt.Errorf("add order item: %w", err)
	}
	return domain.OrderItem{
		ID:        row.ID,
		OrderID:   row.OrderID,
		ProductID: row.ProductID,
		Quantity:  row.Quantity,
		Unit:      row.Unit,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *OrderService) ListItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	rows, err := s.q.OrderItemListByOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	out := make([]domain.OrderItem, len(rows))
	for i, r := range rows {
		out[i] = orderItemToDomain(r)
	}
	return out, nil
}

func (s *OrderService) ClearItems(ctx context.Context, orderID uuid.UUID) error {
	if err := s.q.OrderItemDeleteByOrder(ctx, orderID); err != nil {
		return fmt.Errorf("clear order items %s: %w", orderID, err)
	}
	return nil
}

// ─── mappers ────────────────────────────────────────────────────────────────

func orderToDomain(r postgres.Order) domain.Order {
	orderDate, _ := time.Parse("2006-01-02", r.OrderDate)
	return domain.Order{
		ID:                r.ID,
		ClientID:          r.ClientID,
		KitchenID:         r.KitchenID,
		TelegramChatID:    derefInt64(r.TelegramChatID),
		TelegramMessageID: derefInt64(r.TelegramMessageID),
		OrderDate:         orderDate,
		Status:            domain.OrderStatus(r.Status),
		RawText:           derefStr(r.RawText),
		Note:              derefStr(r.Note),
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		DeletedAt:         r.DeletedAt,
	}
}

func orderItemToDomain(r postgres.OrderItemListByOrderRow) domain.OrderItem {
	return domain.OrderItem{
		ID:        r.ID,
		OrderID:   r.OrderID,
		ProductID: r.ProductID,
		Quantity:  r.Quantity,
		Unit:      r.Unit,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
