package order

import (
	sharedkernel "bakery/internal/pkg/sharedkernel"
)

const (
	EventOrderCreated           = "order.created"
	EventOrderUpdated           = "order.updated"
	EventOrderCancelled         = "order.cancelled"
	EventOrderRestored          = "order.restored"
	EventOrderProduced          = "order.produced"
	EventOrderProductionCleared = "order.production_cleared"
)

var (
	_ sharedkernel.DomainEvent = OrderCreatedEvent{}
	_ sharedkernel.DomainEvent = OrderUpdatedEvent{}
	_ sharedkernel.DomainEvent = OrderCancelledEvent{}
	_ sharedkernel.DomainEvent = OrderRestoredEvent{}
	_ sharedkernel.DomainEvent = OrderProducedEvent{}
	_ sharedkernel.DomainEvent = OrderProductionClearedEvent{}
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

// OrderProducedEvent carries a snapshot of an order after the отработка
// (production fact) was recorded.
type OrderProducedEvent struct {
	sharedkernel.Event
	Order Order `json:"order"`
	// ProducedByUsername — пекарь, внёсший отработку.
	ProducedByUsername string `json:"produced_by_username"`
}

func (OrderProducedEvent) Identity() string { return EventOrderProduced }

// OrderProductionClearedEvent carries a snapshot of an order whose отработка
// was reset.
type OrderProductionClearedEvent struct {
	sharedkernel.Event
	Order Order `json:"order"`
	// ProducedByUsername — пекарь, снявший отработку.
	ProducedByUsername string `json:"produced_by_username"`
}

func (OrderProductionClearedEvent) Identity() string { return EventOrderProductionCleared }

// RecordProduced appends an отработка-recorded domain event to the aggregate.
func (o *Order) RecordProduced(byUsername string) {
	o.ApplyDomain(OrderProducedEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot(), ProducedByUsername: byUsername})
}

// RecordProductionCleared appends an отработка-reset domain event.
func (o *Order) RecordProductionCleared(byUsername string) {
	o.ApplyDomain(OrderProductionClearedEvent{Event: sharedkernel.NewEvent(), Order: o.snapshot(), ProducedByUsername: byUsername})
}

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
