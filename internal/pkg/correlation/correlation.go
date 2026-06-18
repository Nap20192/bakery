// Package correlation carries a request-scoped correlation id through the
// context so a single logical operation can be traced across the API/bot,
// the use case, the outbox, and the published event.
package correlation

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{}

// WithID attaches a correlation id to the context.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the correlation id, or "" if none is set.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// EnsureID returns the context's correlation id, generating and attaching a new
// one if absent.
func EnsureID(ctx context.Context) (context.Context, string) {
	if id := FromContext(ctx); id != "" {
		return ctx, id
	}
	id := uuid.NewString()
	return WithID(ctx, id), id
}
