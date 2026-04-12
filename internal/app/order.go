package app

import (
	"context"
	"fmt"
	"time"

	"bakery/internal/domain/client"
	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/order"
	"bakery/internal/domain/product"
)

type OrderService struct {
	orders order.Repository
}

func NewOrderService(orders order.Repository) *OrderService {
	return &OrderService{orders: orders}
}

type CreateOrderInput struct {
	ClientID          client.ID
	KitchenID         *kitchen.ID
	TelegramChatID    int64
	TelegramMessageID int64
	OrderDate         time.Time
	RawText           string
	Note              string
}

type AddOrderItemInput struct {
	OrderID   order.ID
	ProductID product.ID
	Quantity  float64
	Unit      string
}

func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (*order.Order, error) {
	var tg *order.TelegramSource
	if in.TelegramChatID != 0 || in.TelegramMessageID != 0 {
		tg = &order.TelegramSource{ChatID: in.TelegramChatID, MessageID: in.TelegramMessageID}
	}
	o, err := order.New(in.ClientID, in.KitchenID, in.OrderDate, tg, in.RawText, in.Note)
	if err != nil {
		return nil, err
	}
	if err := s.orders.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("order: save: %w", err)
	}
	return o, nil
}

func (s *OrderService) GetByID(ctx context.Context, id order.ID) (*order.Order, error) {
	return s.orders.GetByID(ctx, id)
}

func (s *OrderService) ListByDate(ctx context.Context, date time.Time) ([]*order.Order, error) {
	from := date
	to := date
	return s.orders.List(ctx, order.Filter{DateFrom: &from, DateTo: &to})
}

func (s *OrderService) ListByDateAndStatus(ctx context.Context, date time.Time, status order.Status) ([]*order.Order, error) {
	from := date
	to := date
	return s.orders.List(ctx, order.Filter{DateFrom: &from, DateTo: &to, Status: &status})
}

func (s *OrderService) UpdateStatus(ctx context.Context, id order.ID, next order.Status) (*order.Order, error) {
	o, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := o.Transition(next); err != nil {
		return nil, err
	}
	if err := s.orders.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("order: save: %w", err)
	}
	return o, nil
}

func (s *OrderService) Cancel(ctx context.Context, id order.ID) error {
	o, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := o.Transition(order.StatusCancelled); err != nil {
		return err
	}
	return s.orders.Save(ctx, o)
}

func (s *OrderService) AddItem(ctx context.Context, in AddOrderItemInput) (*order.Item, error) {
	o, err := s.orders.GetByID(ctx, in.OrderID)
	if err != nil {
		return nil, err
	}
	item, err := o.AddItem(in.ProductID, in.Quantity, product.Unit(in.Unit))
	if err != nil {
		return nil, err
	}
	if err := s.orders.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("order: save: %w", err)
	}
	return item, nil
}

func (s *OrderService) ListItems(ctx context.Context, id order.ID) ([]*order.Item, error) {
	o, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return o.Items(), nil
}
