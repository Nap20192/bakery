// Package sharedkernel holds cross-service domain primitives: the domain event
// contract and the aggregate root that records events.
package sharedkernel

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is anything a domain aggregate emits.
type DomainEvent interface {
	Identity() string
	CreatedAt() time.Time
}

// Event is the base embedded in every concrete domain event.
type Event struct {
	id        uuid.UUID
	createdAt time.Time
}

func NewEvent() Event {
	return Event{id: uuid.New(), createdAt: time.Now().UTC()}
}

func (e Event) EventID() uuid.UUID { return e.id }

func (e Event) CreatedAt() time.Time { return e.createdAt }
