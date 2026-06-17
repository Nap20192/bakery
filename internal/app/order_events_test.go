package app

import (
	"testing"

	orderdomain "bakery/internal/domain/order"
)

func TestOrderEventBusPublishesCreatedOrderCopy(t *testing.T) {
	bus := NewOrderEventBus()
	events := make([]OrderEvent, 0, 1)
	if err := bus.SubscribeOrderCreated(func(event OrderEvent) {
		events = append(events, event)
	}); err != nil {
		t.Fatalf("SubscribeOrderCreated returned error: %v", err)
	}

	order := orderdomain.Order{
		Number: "GG-1",
		Items: []orderdomain.OrderItem{
			{Code: "15635", ProductName: "Пирожок", Quantity: 2},
		},
	}
	bus.PublishOrderCreated(order)
	order.Items[0].Quantity = 99

	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Topic != OrderEventTopicCreated {
		t.Fatalf("topic = %q, want %q", events[0].Topic, OrderEventTopicCreated)
	}
	if events[0].Order.Number != "GG-1" {
		t.Fatalf("order number = %q", events[0].Order.Number)
	}
	if events[0].Order.Items[0].Quantity != 2 {
		t.Fatalf("event item quantity = %v, want 2", events[0].Order.Items[0].Quantity)
	}
}
