package order

import (
	sharedkernel "bakery/internal/pkg/sharedkernel"
)

const (
	EventOrderCreated   = "order.created"
	EventOrderUpdated   = "order.updated"
	EventOrderCancelled = "order.cancelled"
	EventOrderRestored  = "order.restored"
)

var (
	_ sharedkernel.DomainEvent = OrderCreatedEvent{}
	_ sharedkernel.DomainEvent = OrderUpdatedEvent{}
	_ sharedkernel.DomainEvent = OrderCancelledEvent{}
	_ sharedkernel.DomainEvent = OrderRestoredEvent{}
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

// OrderCancelledEvent carries a snapshot of a cancelled order.
type OrderCancelledEvent struct {
	sharedkernel.Event
	Order Order `json:"order"`
}

func (OrderCancelledEvent) Identity() string { return EventOrderCancelled }

// OrderRestoredEvent carries a snapshot of a restored (un-cancelled) order.
type OrderRestoredEvent struct {
	sharedkernel.Event
	Order Order `json:"order"`
}

func (OrderRestoredEvent) Identity() string { return EventOrderRestored }

// RecordCreated appends an order-created domain event to the aggregate.
func (o *Order) RecordCreated() {
	o.ApplyDomain(OrderCreatedEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot()})
}

// RecordUpdated appends an order-updated domain event to the aggregate.
func (o *Order) RecordUpdated() {
	o.ApplyDomain(OrderUpdatedEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot()})
}

// RecordCancelled appends an order-cancelled domain event to the aggregate.
func (o *Order) RecordCancelled() {
	o.ApplyDomain(OrderCancelledEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot()})
}

// RecordRestored appends an order-restored domain event to the aggregate.
func (o *Order) RecordRestored() {
	o.ApplyDomain(OrderRestoredEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot()})
}

// snapshot returns a copy without pending domain events, safe to embed in an
// event payload.
func (o Order) snapshot() Order {
	o.AggregateRoot = sharedkernel.AggregateRoot{}
	return o
}
