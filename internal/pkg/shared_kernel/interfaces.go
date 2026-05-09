package sharedkernel

import "time"

type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

type EntityLike interface {
	ID() string
	IsZero() bool
}

type AggregateRootLike interface {
	EntityLike
	RecordEvent(event DomainEvent)
	DomainEvents() []DomainEvent
	PullDomainEvents() []DomainEvent
	ClearDomainEvents()
}
