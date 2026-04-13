package order

import "fmt"

type Status string

const (
	StatusNew       Status = "new"
	StatusParsed    Status = "parsed"
	StatusConfirmed Status = "confirmed"
	StatusProcessed Status = "processed"
	StatusCancelled Status = "cancelled"
)

var allowedTransitions = map[Status][]Status{
	StatusNew:       {StatusParsed, StatusConfirmed, StatusCancelled},
	StatusParsed:    {StatusConfirmed, StatusCancelled},
	StatusConfirmed: {StatusProcessed, StatusCancelled},
	StatusProcessed: {},
	StatusCancelled: {},
}

func (s Status) CanTransitionTo(next Status) bool {
	for _, a := range allowedTransitions[s] {
		if a == next {
			return true
		}
	}
	return false
}

func (s Status) AllowedNext() []Status {
	return allowedTransitions[s]
}

type InvalidTransitionError struct {
	From    Status
	To      Status
	Allowed []Status
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("order: cannot transition from %q to %q (allowed: %v)", e.From, e.To, e.Allowed)
}
