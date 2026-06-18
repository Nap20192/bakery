// Package apperr provides a small, transport-agnostic error taxonomy shared by
// every service. Use cases return *Error values carrying a Kind (what kind of
// failure), a machine Code (for logs/clients), and a safe user-facing Message.
// Delivery adapters map Kind to a transport status (e.g. HTTP) through a single
// place, instead of switching on individual sentinel errors.
//
// Sentinel errors stay usable: declare them as package-level *Error values and
// errors.Is keeps working by identity, while KindOf/MessageOf extract the
// taxonomy via errors.As — including for wrapped causes.
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
	err     error  // wrapped cause, optional
}

func (e *Error) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.err)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.err }

// New builds an *Error without a wrapped cause.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Wrap builds an *Error that wraps a cause (preserved for logs and errors.Is).
func Wrap(kind Kind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, err: cause}
}

// Kind constructors. Use the wrapping variant (…f) when carrying a cause.

func Invalid(code, message string) *Error      { return New(KindInvalid, code, message) }
func NotFound(code, message string) *Error     { return New(KindNotFound, code, message) }
func Conflict(code, message string) *Error     { return New(KindConflict, code, message) }
func Unauthorized(code, message string) *Error { return New(KindUnauthorized, code, message) }
func Forbidden(code, message string) *Error    { return New(KindForbidden, code, message) }
func Internal(code, message string) *Error     { return New(KindInternal, code, message) }

// KindOf returns the Kind of the first *Error in err's chain, or KindUnknown.
func KindOf(err error) Kind {
	if ae := asError(err); ae != nil {
		return ae.Kind
	}
	return KindUnknown
}

// MessageOf returns the safe Message of the first *Error in err's chain.
func MessageOf(err error) (string, bool) {
	if ae := asError(err); ae != nil {
		return ae.Message, true
	}
	return "", false
}

// As returns the first *Error in err's chain, or nil.
func As(err error) *Error { return asError(err) }

func asError(err error) *Error {
	ae, _ := errors.AsType[*Error](err)
	return ae
}
