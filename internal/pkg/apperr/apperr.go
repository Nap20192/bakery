// Package apperr provides a small, transport-agnostic error taxonomy shared by
// every service. Use cases return *Error values carrying a Kind (what kind of
// failure), a machine Code (for logs/clients), and a safe user-facing Message.
// Delivery adapters map Kind to a transport status (e.g. HTTP) through a single
// place, instead of switching on individual sentinel errors.
//
// Sentinel errors stay usable: declare them as package-level *Error values and
// errors.Is keeps working by identity, while As extracts the taxonomy via
// errors.As — including for wrapped causes.
package apperr

import (
	"errors"
	"fmt"
)

// Kind classifies a failure independently of any transport.
type Kind int

const (
	// KindUnknown is the zero value; treat as an internal error.
	KindUnknown Kind = iota
	// KindInvalid — bad input / validation failure.
	KindInvalid
	// KindNotFound — the requested entity does not exist.
	KindNotFound
	// KindConflict — the request conflicts with current state (e.g. duplicate).
	KindConflict
	// KindUnauthorized — the caller is not authenticated.
	KindUnauthorized
	// KindForbidden — the caller is authenticated but not allowed.
	KindForbidden
	// KindInternal — an unexpected server-side failure.
	KindInternal
)

// Error is the shared application error.
type Error struct {
	Kind    Kind
	Code    string // machine-readable, e.g. "auth.user_not_found"
	Message string // safe, user-facing message
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// New builds an *Error without a wrapped cause.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Kind constructors for the kinds use cases actually return. Unauthorized/
// forbidden responses are produced directly by the auth middlewares.

func Invalid(code, message string) *Error  { return New(KindInvalid, code, message) }
func NotFound(code, message string) *Error { return New(KindNotFound, code, message) }
func Conflict(code, message string) *Error { return New(KindConflict, code, message) }

// As returns the first *Error in err's chain, or nil.
func As(err error) *Error {
	ae, _ := errors.AsType[*Error](err)
	return ae
}
