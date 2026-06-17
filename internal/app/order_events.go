package app

import (
	"time"

	orderdomain "bakery/internal/domain/order"

	evbus "github.com/asaskevich/EventBus"
)

const (
	OrderEventTopicCreated    = "order.created"
	OrderEventTopicUpdated    = "order.updated"
	OrderEventTopicDeletedOld = "order.deleted_old"
)

type OrderEvent struct {
	Topic string
	Order orderdomain.Order
	At    time.Time
}

type OrderCleanupEvent struct {
	Topic     string
	Deleted   int64
	Cutoff    time.Time
	Retention time.Duration
	At        time.Time
}

type OrderEventBus interface {
	PublishOrderCreated(order orderdomain.Order)
	PublishOrderUpdated(order orderdomain.Order)
	PublishOldOrdersDeleted(deleted int64, cutoff time.Time, retention time.Duration)
	SubscribeOrderCreated(handler func(OrderEvent)) error
	SubscribeOrderUpdated(handler func(OrderEvent)) error
	SubscribeOldOrdersDeleted(handler func(OrderCleanupEvent)) error
}

type eventBusOrderBus struct {
	bus evbus.Bus
}

func NewOrderEventBus() OrderEventBus {
	return &eventBusOrderBus{bus: evbus.New()}
}

func (b *eventBusOrderBus) PublishOrderCreated(order orderdomain.Order) {
	b.publishOrder(OrderEventTopicCreated, order)
}

func (b *eventBusOrderBus) PublishOrderUpdated(order orderdomain.Order) {
	b.publishOrder(OrderEventTopicUpdated, order)
}

func (b *eventBusOrderBus) PublishOldOrdersDeleted(deleted int64, cutoff time.Time, retention time.Duration) {
	if b == nil || b.bus == nil {
		return
	}
	b.bus.Publish(OrderEventTopicDeletedOld, OrderCleanupEvent{
		Topic:     OrderEventTopicDeletedOld,
		Deleted:   deleted,
		Cutoff:    cutoff,
		Retention: retention,
		At:        time.Now().UTC(),
	})
}

func (b *eventBusOrderBus) SubscribeOrderCreated(handler func(OrderEvent)) error {
	return b.subscribe(OrderEventTopicCreated, handler)
}

func (b *eventBusOrderBus) SubscribeOrderUpdated(handler func(OrderEvent)) error {
	return b.subscribe(OrderEventTopicUpdated, handler)
}

func (b *eventBusOrderBus) SubscribeOldOrdersDeleted(handler func(OrderCleanupEvent)) error {
	if b == nil || b.bus == nil {
		return nil
	}
	return b.bus.Subscribe(OrderEventTopicDeletedOld, handler)
}

func (b *eventBusOrderBus) publishOrder(topic string, order orderdomain.Order) {
	if b == nil || b.bus == nil {
		return
	}
	b.bus.Publish(topic, OrderEvent{
		Topic: topic,
		Order: cloneOrderEventOrder(order),
		At:    time.Now().UTC(),
	})
}

func (b *eventBusOrderBus) subscribe(topic string, handler func(OrderEvent)) error {
	if b == nil || b.bus == nil {
		return nil
	}
	return b.bus.Subscribe(topic, handler)
}

func cloneOrderEventOrder(order orderdomain.Order) orderdomain.Order {
	order.Items = append([]orderdomain.OrderItem(nil), order.Items...)
	order.History = append([]orderdomain.OrderHistory(nil), order.History...)
	for i := range order.History {
		order.History[i].Items = append([]orderdomain.OrderHistoryItem(nil), order.History[i].Items...)
	}
	return order
}
