package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestKindOf(t *testing.T) {
	sentinel := NotFound("x.not_found", "missing")
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "direct", err: sentinel, want: KindNotFound},
		{name: "wrapped fmt", err: fmt.Errorf("load: %w", sentinel), want: KindNotFound},
		{name: "wrapped apperr cause", err: Wrap(KindInternal, "x.io", "boom", sentinel), want: KindInternal},
		{name: "plain error", err: errors.New("nope"), want: KindUnknown},
		{name: "nil", err: nil, want: KindUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.err); got != tt.want {
				t.Fatalf("KindOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSentinelIdentity(t *testing.T) {
	sentinel := Conflict("x.dup", "duplicate")
	wrapped := fmt.Errorf("create: %w", sentinel)
	if !errors.Is(wrapped, sentinel) {
		t.Fatal("errors.Is must match the wrapped sentinel by identity")
	}
	if msg, ok := MessageOf(wrapped); !ok || msg != "duplicate" {
		t.Fatalf("MessageOf() = (%q, %v), want (%q, true)", msg, ok, "duplicate")
	}
}
