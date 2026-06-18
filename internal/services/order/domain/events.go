package order

import (
	sharedkernel "bakery/internal/pkg/sharedkernel"
)

const (
	EventOrderCreated = "order.created"
	EventOrderUpdated = "order.updated"
)

var (
	_ sharedkernel.DomainEvent = OrderCreatedEvent{}
	_ sharedkernel.DomainEvent = OrderUpdatedEvent{}
)

// OrderCreatedEvent carries a snapshot of a newly created order.
type OrderCreatedEvent struct {
	sharedkernel.Event
	Order Order `json:"order"`
}

func (OrderCreatedEvent) Identity() string { return EventOrderCreated }

// OrderUpdatedEvent carries a snapshot of an updated order.
type OrderUpdatedEvent struct {
	sharedkernel.Event
	Order Order `json:"order"`
}

func (OrderUpdatedEvent) Identity() string { return EventOrderUpdated }

// RecordCreated appends an order-created domain event to the aggregate.
func (o *Order) RecordCreated() {
	o.ApplyDomain(OrderCreatedEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot()})
}

// RecordUpdated appends an order-updated domain event to the aggregate.
func (o *Order) RecordUpdated() {
	o.ApplyDomain(OrderUpdatedEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot()})
}

// snapshot returns a copy without pending domain events, safe to embed in an
// event payload.
func (o Order) snapshot() Order {
	o.AggregateRoot = sharedkernel.AggregateRoot{}
	return o
}
