package app

import (
	"context"
	"fmt"
	"sort"
	"time"

	"bakery/internal/domain/client"
	"bakery/internal/domain/kitchen"
	"bakery/internal/domain/order"
	"bakery/internal/domain/product"
)

type OrderService struct {
	orders   order.Repository
	products product.Repository
	clients  client.Repository
	kitchens kitchen.Repository
}

func NewOrderService(
	orders order.Repository,
	products product.Repository,
	clients client.Repository,
	kitchens kitchen.Repository,
) *OrderService {
	return &OrderService{orders: orders, products: products, clients: clients, kitchens: kitchens}
}

type CreateOrderInput struct {
	ClientID          client.ID
	KitchenID         *kitchen.ID
	TelegramChatID    int64
	TelegramMessageID int64
	OrderDate         time.Time
	RawText           string
	Note              string
	Items             []CreateOrderItem
}

type CreateOrderItem struct {
	ProductID product.ID
	Quantity  float64
	Unit      string
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
	for _, it := range in.Items {
		if _, err := o.AddItem(it.ProductID, it.Quantity, product.Unit(it.Unit)); err != nil {
			return nil, fmt.Errorf("order: add item: %w", err)
		}
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

func (s *OrderService) GetOrderOverview(ctx context.Context, id order.ID) (*order.OrderOverview, error) {
	o, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildOverview(ctx, o)
}

func (s *OrderService) buildOverview(ctx context.Context, o *order.Order) (*order.OrderOverview, error) {
	items := make([]order.OrderItemOverview, 0, len(o.Items()))
	for _, it := range o.Items() {
		name, unit := "", string(it.Unit())
		p, err := s.products.GetByID(ctx, it.ProductID())
		if err == nil {
			name = p.Name()
			unit = string(p.Unit())
		}
		items = append(items, order.OrderItemOverview{
			ProductID:   it.ProductID(),
			ProductName: name,
			Quantity:    it.Quantity(),
			Unit:        unit,
		})
	}
	totals, err := s.orders.GetOrderIngredients(ctx, o.ID())
	if err != nil {
		return nil, fmt.Errorf("order: overview ingredients: %w", err)
	}
	clientName := ""
	if c, err := s.clients.GetByID(ctx, o.ClientID()); err == nil {
		clientName = c.Name()
	}
	kitchenName := ""
	if k := o.KitchenID(); k != nil {
		if kk, err := s.kitchens.GetByID(ctx, *k); err == nil {
			kitchenName = kk.Name()
		}
	}
	return &order.OrderOverview{
		OrderID:          o.ID(),
		OrderDate:        o.OrderDate(),
		Status:           o.Status(),
		Client:           clientName,
		Kitchen:          kitchenName,
		Note:             o.Note(),
		Items:            items,
		TotalIngredients: totals,
	}, nil
}

func (s *OrderService) GetDailyOverview(ctx context.Context, date time.Time) (*order.DayGroup, error) {
	from, to := date, date
	orders, err := s.orders.List(ctx, order.Filter{DateFrom: &from, DateTo: &to})
	if err != nil {
		return nil, fmt.Errorf("order: daily list: %w", err)
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt().Before(orders[j].CreatedAt())
	})
	overviews := make([]order.OrderOverview, 0, len(orders))
	for _, o := range orders {
		ov, err := s.buildOverview(ctx, o)
		if err != nil {
			return nil, err
		}
		overviews = append(overviews, *ov)
	}
	return &order.DayGroup{
		Date:             date,
		Orders:           overviews,
		TotalIngredients: aggregateIngredients(overviews),
	}, nil
}

func (s *OrderService) GetPeriodOverview(ctx context.Context, from, to time.Time) (*order.PeriodOverview, error) {
	orders, err := s.orders.List(ctx, order.Filter{DateFrom: &from, DateTo: &to})
	if err != nil {
		return nil, fmt.Errorf("order: period list: %w", err)
	}
	byDate := make(map[time.Time][]*order.Order)
	for _, o := range orders {
		d := o.OrderDate()
		byDate[d] = append(byDate[d], o)
	}
	dates := make([]time.Time, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	days := make([]order.DayGroup, 0, len(dates))
	for _, d := range dates {
		dayOrders := byDate[d]
		sort.Slice(dayOrders, func(i, j int) bool {
			return dayOrders[i].CreatedAt().Before(dayOrders[j].CreatedAt())
		})
		overviews := make([]order.OrderOverview, 0, len(dayOrders))
		for _, o := range dayOrders {
			ov, err := s.buildOverview(ctx, o)
			if err != nil {
				return nil, err
			}
			overviews = append(overviews, *ov)
		}
		days = append(days, order.DayGroup{
			Date:             d,
			Orders:           overviews,
			TotalIngredients: aggregateIngredients(overviews),
		})
	}
	grand := make([]order.IngredientLine, 0)
	for _, dg := range days {
		grand = mergeIngredients(grand, dg.TotalIngredients)
	}
	sort.Slice(grand, func(i, j int) bool { return grand[i].IngredientName < grand[j].IngredientName })
	return &order.PeriodOverview{From: from, To: to, Days: days, GrandTotal: grand}, nil
}

func aggregateIngredients(overviews []order.OrderOverview) []order.IngredientLine {
	acc := make([]order.IngredientLine, 0)
	for _, ov := range overviews {
		acc = mergeIngredients(acc, ov.TotalIngredients)
	}
	sort.Slice(acc, func(i, j int) bool { return acc[i].IngredientName < acc[j].IngredientName })
	return acc
}

func mergeIngredients(dst, src []order.IngredientLine) []order.IngredientLine {
	idx := make(map[string]int, len(dst))
	for i, l := range dst {
		idx[l.IngredientID.String()] = i
	}
	for _, l := range src {
		key := l.IngredientID.String()
		if i, ok := idx[key]; ok {
			dst[i].TotalQty += l.TotalQty
			continue
		}
		idx[key] = len(dst)
		dst = append(dst, l)
	}
	return dst
}
