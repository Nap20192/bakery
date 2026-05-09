package sharedkernel

type AggregateRoot struct {
	Entity
	events []DomainEvent
}

func NewAggregateRoot() AggregateRoot {
	return AggregateRoot{Entity: NewEntity()}
}

func NewAggregateRootWithID(id string) AggregateRoot {
	return AggregateRoot{Entity: NewEntityWithID(id)}
}

func (a *AggregateRoot) RecordEvent(event DomainEvent) {
	if event == nil {
		return
	}
	a.events = append(a.events, event)
}

func (a *AggregateRoot) DomainEvents() []DomainEvent {
	events := make([]DomainEvent, len(a.events))
	copy(events, a.events)
	return events
}

func (a *AggregateRoot) PullDomainEvents() []DomainEvent {
	events := a.DomainEvents()
	a.ClearDomainEvents()
	return events
}

func (a *AggregateRoot) ClearDomainEvents() {
	a.events = nil
}
