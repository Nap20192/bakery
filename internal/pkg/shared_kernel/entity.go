package sharedkernel

import (
	"strings"

	"github.com/google/uuid"
)

type Entity struct {
	id string
}

func NewEntity() Entity {
	return Entity{id: uuid.NewString()}
}

func NewEntityWithID(id string) Entity {
	return Entity{id: strings.TrimSpace(id)}
}

func (e Entity) ID() string {
	return e.id
}

func (e Entity) IsZero() bool {
	return e.id == ""
}

func (e Entity) Equals(other Entity) bool {
	return e.id != "" && e.id == other.id
}
