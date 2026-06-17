package sharedkernel

// AggregateRoot collects domain events recorded during a unit of work. The
// application layer pulls them after persistence and publishes them.
type AggregateRoot struct {
	domainEvents []DomainEvent
}

func (ar *AggregateRoot) ApplyDomain(e DomainEvent) {
	ar.domainEvents = append(ar.domainEvents, e)
}

func (ar *AggregateRoot) DomainEvents() []DomainEvent {
	return ar.domainEvents
}

// PullDomainEvents returns the recorded events and clears them.
func (ar *AggregateRoot) PullDomainEvents() []DomainEvent {
	events := ar.domainEvents
	ar.domainEvents = nil
	return events
}
