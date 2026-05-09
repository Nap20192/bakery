package sharedkernel

import (
	"testing"
	"time"
)

type testEvent struct {
	name       string
	occurredAt time.Time
}

func (e testEvent) EventName() string {
	return e.name
}

func (e testEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func TestAggregateRootDomainEvents(t *testing.T) {
	aggregate := NewAggregateRootWithID("order-1")
	event := testEvent{name: "order.created", occurredAt: time.Now().UTC()}

	aggregate.RecordEvent(nil)
	aggregate.RecordEvent(event)

	events := aggregate.DomainEvents()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].EventName() != "order.created" {
		t.Fatalf("event name = %s", events[0].EventName())
	}

	events[0] = testEvent{name: "changed"}
	if aggregate.DomainEvents()[0].EventName() != "order.created" {
		t.Fatal("DomainEvents should return a copy")
	}

	pulled := aggregate.PullDomainEvents()
	if len(pulled) != 1 {
		t.Fatalf("pulled events = %d, want 1", len(pulled))
	}
	if len(aggregate.DomainEvents()) != 0 {
		t.Fatal("PullDomainEvents should clear events")
	}
}
